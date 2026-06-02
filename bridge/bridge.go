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
	"k8s.io/apimachinery/pkg/types"
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
	maintenance    atomic.Bool // true → reject new instance creates; set via PUT /v1/cluster/maintenance
	draining       atomic.Bool // true → drain in progress; prevents concurrent drains
	runner         *OperationRunner
	pendingCreates sync.Map // instanceID → pendingCreate; throttles duplicate creates
	pendingOps     sync.Map // instanceID → *inflightOp; throttles duplicate async ops, allows supersede
	execTokens     sync.Map // token(string) → execToken; short-lived WebSocket auth tokens
}

type execToken struct {
	instanceID string
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
		Logger:  log.New(os.Stdout, "bridge ", log.LstdFlags|log.LUTC),
		Metrics: NewMetrics(cfg.ClusterName, nil),
		runner:  newOperationRunner(cfg.OperationTimeout),
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
	v1.HandleFunc("/instances", b.handleListInstances).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}", b.handleCreateInstance).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}", b.handleGetInstance).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}", b.handleUpdateInstance).Methods(http.MethodPut)
	v1.HandleFunc("/instances/{id}", b.handleDeleteInstance).Methods(http.MethodDelete)
	v1.HandleFunc("/instances/{id}/status", b.handleGetStatus).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/health", b.handleHealth).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/exec", b.handleExec).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/events", b.handleGetEvents).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/operations", b.handleListInstanceOperations).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/restart", b.handleRestartInstance).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/redeploy", b.handleRedeployInstance).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/rollback", b.handleRollbackInstance).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/repair", b.handleRepairInstance).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/upgrade", b.handleUpgradeInstance).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/terminal/recreate", b.handleRecreateTerminal).Methods(http.MethodPost)
	v1.HandleFunc("/operations/{id}", b.handleGetOperation).Methods(http.MethodGet)

	// Instance config (HermesConfig — soul, agent, gateway, model, etc.)
	v1.HandleFunc("/instances/{id}/config", b.handleGetInstanceConfig).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/config", b.handleSetInstanceConfig).Methods(http.MethodPut)
	// Provider config
	v1.HandleFunc("/instances/{id}/config/providers", b.handleGetInstanceProviders).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/config/providers", b.handleSetInstanceProvider).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/config/providers/{name}", b.handleUpdateInstanceProvider).Methods(http.MethodPut)
	v1.HandleFunc("/instances/{id}/config/providers/{name}", b.handleDeleteInstanceProvider).Methods(http.MethodDelete)
	v1.HandleFunc("/instances/{id}/config/model", b.handleSetInstanceModel).Methods(http.MethodPut)
	// Gateway health per instance
	v1.HandleFunc("/instances/{id}/gateway/status", b.handleGatewayStatus).Methods(http.MethodGet)
	// Profiles (proxied from hermes API server)
	v1.HandleFunc("/instances/{id}/profiles", b.handleGetInstanceProfiles).Methods(http.MethodGet)

	// Messaging integrations
	v1.HandleFunc("/instances/{id}/integrations", b.handleGetInstanceIntegrations).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/integrations", b.handleSetIntegrations).Methods(http.MethodPut)
	v1.HandleFunc("/instances/{id}/integrations/{platform}", b.handleEnableIntegration).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/integrations/{platform}", b.handleDisableIntegration).Methods(http.MethodDelete)

	// Admin / diagnostics — backend-only, never called by the frontend directly
	v1.HandleFunc("/cluster/summary", b.handleGetClusterSummary).Methods(http.MethodGet)
	v1.HandleFunc("/cluster/resources", b.handleGetClusterResources).Methods(http.MethodGet)
	v1.HandleFunc("/cluster/maintenance", b.handleGetMaintenanceMode).Methods(http.MethodGet)
	v1.HandleFunc("/cluster/maintenance", b.handleSetMaintenanceMode).Methods(http.MethodPut)
	v1.HandleFunc("/cluster/drain", b.handleDrainCluster).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/diagnostics", b.handleGetInstanceDiagnostics).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/logs", b.handleGetInstanceLogs).Methods(http.MethodGet)
	v1.HandleFunc("/instances/{id}/resources", b.handleGetInstanceResources).Methods(http.MethodGet)

	// Agent templates
	v1.HandleFunc("/agents", b.handleListAgentTemplates).Methods(http.MethodGet)
	v1.HandleFunc("/agents", b.handleCreateAgentTemplate).Methods(http.MethodPost)
	v1.HandleFunc("/agents/{agentId}", b.handleGetAgentTemplate).Methods(http.MethodGet)
	v1.HandleFunc("/agents/{agentId}", b.handleUpdateAgentTemplate).Methods(http.MethodPut)
	v1.HandleFunc("/agents/{agentId}", b.handleDeleteAgentTemplate).Methods(http.MethodDelete)
	v1.HandleFunc("/instances/{id}/agent", b.handleApplyAgentTemplate).Methods(http.MethodPost)
	v1.HandleFunc("/instances/{id}/agent", b.handleGetInstanceAgent).Methods(http.MethodGet)

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

// issueExecToken mints a single-use 32-byte random token tied to instanceID.
// The token expires after 2 minutes — enough for a browser to open the WebSocket.
func (b *Bridge) issueExecToken(instanceID string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate exec token: %w", err)
	}
	tok := hex.EncodeToString(buf)
	b.execTokens.Store(tok, execToken{instanceID: instanceID, expiry: time.Now().Add(2 * time.Minute)})
	return tok, nil
}

// consumeExecToken validates and atomically deletes a token.
// Returns true only if the token exists, hasn't expired, and matches instanceID.
func (b *Bridge) consumeExecToken(tok, instanceID string) bool {
	v, ok := b.execTokens.LoadAndDelete(tok)
	if !ok {
		return false
	}
	et := v.(execToken)
	return et.instanceID == instanceID && time.Now().Before(et.expiry)
}

func (b *Bridge) trackOperation(operation, result string, started time.Time) {
	b.Metrics.OperationLatency.WithLabelValues(b.ClusterName, operation, result).Observe(time.Since(started).Seconds())
	b.Metrics.OperationResults.WithLabelValues(b.ClusterName, operation, result).Inc()
}

func (b *Bridge) submitOperation(operationType, instanceID string, fn func(context.Context) error) Operation {
	return b.runner.Submit(operationType, instanceID, fn)
}

// inflightOp pairs a running Operation with the cancel func for its context, so
// a delete can supersede (cancel) an in-progress restart/upgrade/repair.
type inflightOp struct {
	op     *Operation
	cancel context.CancelFunc
}

// supersedeInFlight cancels any in-progress instance operation for instanceID and
// marks its record as superseded. Used by delete, which must always win — a user
// must be able to remove an instance even mid-upgrade. No-op if nothing is running.
func (b *Bridge) supersedeInFlight(instanceID string) {
	if v, ok := b.pendingOps.Load(instanceID); ok {
		inf := v.(*inflightOp)
		b.runner.Update(inf.op.ID, func(o *Operation) {
			o.Message = fmt.Sprintf("%s superseded by delete", o.Type)
		})
		inf.cancel()
	}
}

// submitInstanceOperation is like submitOperation but enforces one-in-flight per instance.
// fn returns an optional success message (used verbatim if non-empty) and an error.
// The first return is a snapshot of the accepted op; if an op is already running for
// instanceID it returns (zero, existingSnapshot) and the caller should 409.
func (b *Bridge) submitInstanceOperation(operationType, instanceID string, fn func(context.Context) (string, error)) (Operation, *Operation) {
	op := &Operation{
		ID:         newOperationID(),
		Type:       operationType,
		InstanceID: instanceID,
		Status:     "running",
		Message:    fmt.Sprintf("%s scheduled", operationType),
		StartedAt:  time.Now().UTC(),
	}

	// Create the cancellable context up front so the stored inflightOp always has a
	// valid cancel func — no window where a superseding delete reads a nil cancel.
	ctx, cancel := context.WithTimeout(context.Background(), b.Config.OperationTimeout)
	actual, loaded := b.pendingOps.LoadOrStore(instanceID, &inflightOp{op: op, cancel: cancel})
	if loaded {
		cancel() // discard the unused context for the rejected op
		// Return a snapshot of the in-flight op, not the live pointer its goroutine mutates.
		existing, _ := b.runner.Get(actual.(*inflightOp).op.ID)
		return Operation{}, &existing
	}

	b.runner.Record(op)
	// Snapshot before the goroutine launches so the returned value can be serialized
	// without racing the in-place updates below. Callers poll Get for fresh state.
	snapshot := *op

	go func() {
		defer b.pendingOps.Delete(instanceID)
		defer cancel()

		msg, err := fn(ctx)
		completedAt := time.Now().UTC()
		if err != nil {
			// A canceled context means a delete superseded this op — report it as
			// superseded, not failed, so the user doesn't see a scary error for an
			// instance they intentionally removed.
			status, label := "failed", fmt.Sprintf("%s failed", operationType)
			if ctx.Err() == context.Canceled {
				status, label = "superseded", fmt.Sprintf("%s superseded by delete", operationType)
			}
			b.runner.Update(op.ID, func(existing *Operation) {
				existing.Status = status
				existing.Error = err.Error()
				existing.Message = label
				existing.CompletedAt = &completedAt
			})
			return
		}

		successMsg := fmt.Sprintf("%s completed", operationType)
		if msg != "" {
			successMsg = msg
		}
		b.runner.Update(op.ID, func(existing *Operation) {
			existing.Status = "succeeded"
			existing.Message = successMsg
			existing.CompletedAt = &completedAt
		})
	}()

	return snapshot, nil
}

func (b *Bridge) startOperationCleanup(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			b.runner.Cleanup(time.Now().Add(-1 * time.Hour))
			// Evict expired pendingCreates entries. These are checked on read
			// but never deleted, causing the map to grow unbounded over time.
			b.pendingCreates.Range(func(k, v any) bool {
				if rec := v.(pendingCreate); time.Now().After(rec.until) {
					b.pendingCreates.Delete(k)
				}
				return true
			})
			// Evict stale pendingOps entries (guarded by goroutine defer, but
			// clean up any that were abandoned without completing).
			b.pendingOps.Range(func(k, v any) bool {
				if inf := v.(*inflightOp); inf.op.CompletedAt != nil {
					b.pendingOps.Delete(k)
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

func (b *Bridge) releaseName(instanceID string) string {
	if strings.TrimSpace(b.Config.ReleasePrefix) == "" {
		return instanceID
	}
	return b.Config.ReleasePrefix + instanceID
}

func (b *Bridge) instanceNamespace(spec InstanceSpec) string {
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

func (b *Bridge) getInstance(ctx context.Context, instanceID string) (*Instance, error) {
	rel, err := b.lookupRelease(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	spec, err := instanceSpecFromRelease(instanceID, rel.Config, rel.Namespace, b.ClusterName)
	if err != nil {
		return nil, err
	}
	status, err := b.GetInstanceStatus(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	status.Spec = spec
	return &Instance{Spec: spec, Status: status}, nil
}

func (b *Bridge) lookupRelease(_ context.Context, instanceID string) (*release.Release, error) {
	releaseName := b.releaseName(instanceID)
	// Workspace releases live in their own namespace — use a per-workspace config.
	helmCfg, err := newHelmActionConfigForNamespace(b.Config, instanceID)
	if err != nil {
		return nil, fmt.Errorf("helm config for workspace %s: %w", instanceID, err)
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
	return nil, errInstanceNotFound(instanceID)
}

// ensureNamespace creates (or label-patches) the instance namespace. It always
// carries the bridge's operational anchor hermeshq/managed-by=bridge (used by
// drain + cluster summary), plus the contract labels/annotations supplied by the
// caller. The anchor is set last so it can never be displaced.
func (b *Bridge) ensureNamespace(ctx context.Context, name string, lbls, anns map[string]string) error {
	labels := map[string]string{}
	for k, v := range lbls {
		labels[k] = v
	}
	labels[namespaceManagedByKey] = namespaceManagedByValue

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: anns,
		},
	}
	_, err := b.KubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		// Patch labels (and annotations) onto pre-existing namespaces so they stay
		// discoverable and the contract metadata is kept current.
		patch, mErr := json.Marshal(map[string]any{
			"metadata": map[string]any{"labels": labels, "annotations": anns},
		})
		if mErr != nil {
			return mErr
		}
		_, err = b.KubeClient.CoreV1().Namespaces().Patch(
			ctx, name, types.MergePatchType, patch, metav1.PatchOptions{},
		)
		return err
	}
	return err
}

func errInstanceNotFound(instanceID string) error {
	return fmt.Errorf("instance %q not found", instanceID)
}

func isInstanceNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "not found")
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
