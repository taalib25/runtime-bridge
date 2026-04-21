package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const sharedSecretHeader = "X-Bridge-Secret"

type Bridge struct {
	Config         Config
	HelmConfig     *action.Configuration
	KubeClient     *kubernetes.Clientset
	ClusterName    string
	KubeconfigPath string
	ChartPath      string
	HTTPClient     *http.Client
	Logger         *log.Logger
	Metrics        *Metrics

	ready      atomic.Bool
	mu         sync.RWMutex
	operations map[string]*Operation
}

func NewBridge(cfg Config) (*Bridge, error) {
	restConfig, err := buildRESTConfig(cfg.KubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("build kubernetes config: %w", err)
	}

	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	actionConfig, err := newHelmActionConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize helm action config: %w", err)
	}

	bridge := &Bridge{
		Config:         cfg,
		HelmConfig:     actionConfig,
		KubeClient:     kubeClient,
		ClusterName:    cfg.ClusterName,
		KubeconfigPath: cfg.KubeconfigPath,
		ChartPath:      cfg.ChartPath,
		HTTPClient:     &http.Client{Timeout: cfg.HTTPClientTimeout},
		Logger:         log.New(os.Stdout, "bridge ", log.LstdFlags|log.LUTC),
		Metrics:        NewMetrics(cfg.ClusterName, nil),
		operations:     make(map[string]*Operation),
	}

	bridge.ready.Store(true)
	return bridge, nil
}

func buildRESTConfig(kubeconfigPath string) (*rest.Config, error) {
	if strings.TrimSpace(kubeconfigPath) != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	clientCfg := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	return clientCfg.ClientConfig()
}

func newHelmActionConfig(cfg Config) (*action.Configuration, error) {
	flags := genericclioptions.NewConfigFlags(true)
	flags.Namespace = &cfg.Namespace
	if strings.TrimSpace(cfg.KubeconfigPath) != "" {
		flags.KubeConfig = &cfg.KubeconfigPath
	}

	helmDriver := os.Getenv("HELM_DRIVER")
	if helmDriver == "" {
		helmDriver = "secret"
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(flags, cfg.Namespace, helmDriver, func(format string, args ...interface{}) {
		log.Printf("[helm] "+format, args...)
	}); err != nil {
		return nil, fmt.Errorf("init helm action config: %w", err)
	}
	return actionConfig, nil
}

func (b *Bridge) Router() http.Handler {
	r := mux.NewRouter()
	r.HandleFunc("/healthz", b.handleHealthz).Methods(http.MethodGet)
	r.HandleFunc("/readyz", b.handleReadyz).Methods(http.MethodGet)
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Use(b.authMiddleware)
	v1.HandleFunc("/workspaces", b.handleListWorkspaces).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}", b.handleCreateWorkspace).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}", b.handleGetWorkspace).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}", b.handleUpdateWorkspace).Methods(http.MethodPut)
	v1.HandleFunc("/workspaces/{id}", b.handleDeleteWorkspace).Methods(http.MethodDelete)
	v1.HandleFunc("/workspaces/{id}/status", b.handleGetStatus).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}/health", b.handleHealth).Methods(http.MethodGet)

	return r
}

func (b *Bridge) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get(sharedSecretHeader)
		if subtle.ConstantTimeCompare([]byte(provided), []byte(b.Config.BridgeSecret)) != 1 {
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (b *Bridge) trackOperation(operation, result string, started time.Time) {
	b.Metrics.OperationLatency.WithLabelValues(b.ClusterName, operation, result).Observe(time.Since(started).Seconds())
	b.Metrics.OperationResults.WithLabelValues(b.ClusterName, operation, result).Inc()
}

func (b *Bridge) recordOperation(op *Operation) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.operations[op.ID] = op
}

func (b *Bridge) updateOperation(id string, mutate func(*Operation)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if op, ok := b.operations[id]; ok {
		mutate(op)
	}
}

func (b *Bridge) getOperation(id string) *Operation {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if op, ok := b.operations[id]; ok {
		copy := *op
		return &copy
	}
	return nil
}

func (b *Bridge) submitOperation(operationType, workspaceID string, fn func(context.Context) error) *Operation {
	op := &Operation{
		ID:          newOperationID(),
		Type:        operationType,
		WorkspaceID: workspaceID,
		Status:      "running",
		Message:     fmt.Sprintf("%s scheduled", operationType),
		StartedAt:   time.Now().UTC(),
	}
	b.recordOperation(op)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), b.Config.OperationTimeout)
		defer cancel()

		if err := fn(ctx); err != nil {
			completedAt := time.Now().UTC()
			b.updateOperation(op.ID, func(existing *Operation) {
				existing.Status = "failed"
				existing.Error = err.Error()
				existing.Message = fmt.Sprintf("%s failed", operationType)
				existing.CompletedAt = &completedAt
			})
			return
		}

		completedAt := time.Now().UTC()
		b.updateOperation(op.ID, func(existing *Operation) {
			existing.Status = "succeeded"
			existing.Message = fmt.Sprintf("%s completed", operationType)
			existing.CompletedAt = &completedAt
		})
	}()

	return op
}

func (b *Bridge) releaseName(workspaceID string) string {
	if strings.TrimSpace(b.Config.ReleasePrefix) == "" {
		return workspaceID
	}
	return b.Config.ReleasePrefix + workspaceID
}

func (b *Bridge) workspaceNamespace(spec WorkspaceSpec) string {
	if strings.TrimSpace(spec.Namespace) != "" {
		return spec.Namespace
	}
	return b.Config.Namespace
}

func (b *Bridge) Ready() bool {
	return b.ready.Load()
}

func (b *Bridge) CheckReadiness(ctx context.Context) error {
	if !b.Ready() {
		return fmt.Errorf("bridge is not ready")
	}
	_, err := b.KubeClient.Discovery().ServerVersion()
	if err != nil {
		return err
	}
	return nil
}

func (b *Bridge) getWorkspace(ctx context.Context, workspaceID string) (*Workspace, error) {
	rel, err := b.lookupRelease(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	spec, err := workspaceSpecFromRelease(workspaceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return nil, err
	}
	status, err := b.GetWorkspaceStatus(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	status.Spec = spec
	return &Workspace{Spec: spec, Status: status}, nil
}

func (b *Bridge) lookupRelease(ctx context.Context, workspaceID string) (*release.Release, error) {
	releaseName := b.releaseName(workspaceID)
	lister := action.NewList(b.HelmConfig)
	lister.All = true
	lister.Filter = fmt.Sprintf("^%s$", releaseName)
	releases, err := lister.Run()
	if err != nil {
		return nil, err
	}
	for _, rel := range releases {
		if rel.Name == releaseName {
			return rel, nil
		}
	}
	return nil, errWorkspaceNotFound(workspaceID)
}

func errWorkspaceNotFound(workspaceID string) error {
	return fmt.Errorf("workspace %q not found", workspaceID)
}

func isWorkspaceNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
}

func newOperationID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("op-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, `{"error":"encode response"}`+"\n", http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

func deploymentConditions(conditions []metav1.Condition) []string {
	result := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		result = append(result, fmt.Sprintf("%s=%s:%s", condition.Type, condition.Status, condition.Reason))
	}
	sort.Strings(result)
	return result
}
