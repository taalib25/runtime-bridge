package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const sharedSecretHeader = "X-Bridge-Secret"

type Bridge struct {
	Config         Config
	HelmConfig     *action.Configuration
	KubeClient     *kubernetes.Clientset
	DynamicClient  dynamic.Interface
	RESTConfig     *rest.Config
	ClusterName    string
	KubeconfigPath string
	ChartPath      string
	HTTPClient     *http.Client
	Logger         *log.Logger
	Metrics        *Metrics

	ready          atomic.Bool
	mu             sync.RWMutex
	operations     map[string]*Operation
	pendingCreates sync.Map // workspaceID → pendingCreate; throttles duplicate creates
	execTokens     sync.Map // token(string) → execToken; short-lived WebSocket auth tokens
}

type execToken struct {
	workspaceID string
	expiry      time.Time
}

type pendingCreate struct {
	apiKey string
	until  time.Time
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

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	actionConfig, err := newHelmActionConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize helm action config: %w", err)
	}

	bridge := &Bridge{
		Config:         cfg,
		HelmConfig:     actionConfig,
		KubeClient:     kubeClient,
		DynamicClient:  dynamicClient,
		RESTConfig:     restConfig,
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
	return newHelmActionConfigForNamespace(cfg, cfg.Namespace)
}

func newHelmActionConfigForNamespace(cfg Config, namespace string) (*action.Configuration, error) {
	flags := genericclioptions.NewConfigFlags(true)
	flags.Namespace = &namespace
	if strings.TrimSpace(cfg.KubeconfigPath) != "" {
		flags.KubeConfig = &cfg.KubeconfigPath
	}

	helmDriver := os.Getenv("HELM_DRIVER")
	if helmDriver == "" {
		helmDriver = "secret"
	}

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(flags, namespace, helmDriver, func(format string, args ...interface{}) {
		log.Printf("[helm] "+format, args...)
	}); err != nil {
		return nil, fmt.Errorf("init helm action config: %w", err)
	}
	return actionConfig, nil
}

func (b *Bridge) helmConfigForNamespace(namespace string) (*action.Configuration, error) {
	return newHelmActionConfigForNamespace(b.Config, namespace)
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
	v1.HandleFunc("/workspaces/{id}/exec", b.handleExec).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}/events", b.handleGetEvents).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}/operations", b.handleListWorkspaceOperations).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}/restart", b.handleRestartWorkspace).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/redeploy", b.handleRedeployWorkspace).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/rollback", b.handleRollbackWorkspace).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/repair", b.handleRepairWorkspace).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/terminal/recreate", b.handleRecreateTerminal).Methods(http.MethodPost)
	v1.HandleFunc("/operations/{id}", b.handleGetOperation).Methods(http.MethodGet)

	// Provider config
	v1.HandleFunc("/workspaces/{id}/config/providers", b.handleGetWorkspaceProviders).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}/config/providers", b.handleSetWorkspaceProvider).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/config/providers/{name}", b.handleUpdateWorkspaceProvider).Methods(http.MethodPut)
	v1.HandleFunc("/workspaces/{id}/config/providers/{name}", b.handleDeleteWorkspaceProvider).Methods(http.MethodDelete)
	v1.HandleFunc("/workspaces/{id}/config/model", b.handleSetWorkspaceModel).Methods(http.MethodPut)

	// Messaging integrations
	v1.HandleFunc("/workspaces/{id}/integrations", b.handleGetWorkspaceIntegrations).Methods(http.MethodGet)
	v1.HandleFunc("/workspaces/{id}/integrations/{platform}", b.handleEnableIntegration).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/integrations/{platform}", b.handleDisableIntegration).Methods(http.MethodDelete)

	// Agent templates
	v1.HandleFunc("/agents", b.handleListAgentTemplates).Methods(http.MethodGet)
	v1.HandleFunc("/agents", b.handleCreateAgentTemplate).Methods(http.MethodPost)
	v1.HandleFunc("/agents/{agentId}", b.handleGetAgentTemplate).Methods(http.MethodGet)
	v1.HandleFunc("/agents/{agentId}", b.handleUpdateAgentTemplate).Methods(http.MethodPut)
	v1.HandleFunc("/agents/{agentId}", b.handleDeleteAgentTemplate).Methods(http.MethodDelete)
	v1.HandleFunc("/workspaces/{id}/agent", b.handleApplyAgentTemplate).Methods(http.MethodPost)
	v1.HandleFunc("/workspaces/{id}/agent", b.handleGetWorkspaceAgent).Methods(http.MethodGet)

	return b.metricsMiddleware(r)
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Hijack delegates to the underlying ResponseWriter so WebSocket upgrades work
// through the metrics middleware wrapper.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("underlying ResponseWriter does not support hijacking")
	}
	return h.Hijack()
}

func (b *Bridge) metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)

		path := r.URL.Path
		if route := mux.CurrentRoute(r); route != nil {
			if tmpl, err := route.GetPathTemplate(); err == nil {
				path = tmpl
			}
		}
		code := strconv.Itoa(rw.statusCode)
		b.Metrics.HTTPRequests.WithLabelValues(r.Method, path, code).Inc()
		b.Metrics.HTTPDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
	})
}

func (b *Bridge) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Short-lived exec tokens are accepted on the WebSocket upgrade path so
		// browsers can connect without custom headers (WebSocket API doesn't support them).
		if tok := r.URL.Query().Get("token"); tok != "" {
			vars := mux.Vars(r)
			wsID := vars["id"]
			if b.consumeExecToken(tok, wsID) {
				next.ServeHTTP(w, r)
				return
			}
			b.Logger.Printf("[Auth] 401 %s %s — invalid or expired exec token", r.Method, r.URL.Path)
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		provided := r.Header.Get(sharedSecretHeader)
		if subtle.ConstantTimeCompare([]byte(provided), []byte(b.Config.BridgeSecret)) != 1 {
			b.Logger.Printf("[Auth] 401 %s %s — missing or wrong %s (provided len=%d)", r.Method, r.URL.Path, sharedSecretHeader, len(provided))
			writeJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// issueExecToken mints a single-use 32-byte random token tied to workspaceID.
// The token expires after 2 minutes — enough for a browser to open the WebSocket.
func (b *Bridge) issueExecToken(workspaceID string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate exec token: %w", err)
	}
	tok := hex.EncodeToString(buf)
	b.execTokens.Store(tok, execToken{workspaceID: workspaceID, expiry: time.Now().Add(2 * time.Minute)})
	return tok, nil
}

// consumeExecToken validates and atomically deletes a token.
// Returns true only if the token exists, hasn't expired, and matches workspaceID.
func (b *Bridge) consumeExecToken(tok, workspaceID string) bool {
	v, ok := b.execTokens.LoadAndDelete(tok)
	if !ok {
		return false
	}
	et := v.(execToken)
	return et.workspaceID == workspaceID && time.Now().Before(et.expiry)
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

func (b *Bridge) startOperationCleanup(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Evict completed operations older than 1 hour.
			cutoff := time.Now().Add(-1 * time.Hour)
			b.mu.Lock()
			for id, op := range b.operations {
				if op.CompletedAt != nil && op.CompletedAt.Before(cutoff) {
					delete(b.operations, id)
				}
			}
			b.mu.Unlock()
			// Evict expired pendingCreates entries. These are checked on read
			// but never deleted, causing the map to grow unbounded over time.
			b.pendingCreates.Range(func(k, v any) bool {
				if rec := v.(pendingCreate); time.Now().After(rec.until) {
					b.pendingCreates.Delete(k)
				}
				return true
			})
			// Evict expired exec tokens (normally consumed on first use, but
			// clean up any that were never redeemed).
			b.execTokens.Range(func(k, v any) bool {
				if et := v.(execToken); time.Now().After(et.expiry) {
					b.execTokens.Delete(k)
				}
				return true
			})
		case <-ctx.Done():
			return
		}
	}
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

// checkClusterCapacity returns an error if the cluster has no nodes able to
// accept new workloads — all nodes are cordoned, not Ready, or under pressure.
func (b *Bridge) checkClusterCapacity(ctx context.Context) error {
	nodes, err := b.KubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("check cluster capacity: %w", err)
	}
	if len(nodes.Items) == 0 {
		return fmt.Errorf("cluster at capacity: no nodes registered")
	}

	var issues []string
	schedulable := 0
	for _, node := range nodes.Items {
		if node.Spec.Unschedulable {
			continue
		}
		ready := false
		for _, c := range node.Status.Conditions {
			switch c.Type {
			case corev1.NodeReady:
				if c.Status == corev1.ConditionTrue {
					ready = true
				}
			case corev1.NodeMemoryPressure:
				if c.Status == corev1.ConditionTrue {
					issues = append(issues, fmt.Sprintf("node %s: memory pressure", node.Name))
				}
			case corev1.NodeDiskPressure:
				if c.Status == corev1.ConditionTrue {
					issues = append(issues, fmt.Sprintf("node %s: disk pressure", node.Name))
				}
			}
		}
		if ready {
			schedulable++
		}
	}

	if schedulable == 0 {
		if len(issues) > 0 {
			return fmt.Errorf("cluster at capacity: %s", strings.Join(issues, "; "))
		}
		return fmt.Errorf("cluster at capacity: no ready nodes available")
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

func (b *Bridge) lookupRelease(_ context.Context, workspaceID string) (*release.Release, error) {
	releaseName := b.releaseName(workspaceID)
	// Workspace releases live in their own namespace — use a per-workspace config.
	helmCfg, err := newHelmActionConfigForNamespace(b.Config, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("helm config for workspace %s: %w", workspaceID, err)
	}
	lister := action.NewList(helmCfg)
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

func (b *Bridge) ensureNamespace(ctx context.Context, name string) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	_, err := b.KubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		return nil
	}
	return err
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
