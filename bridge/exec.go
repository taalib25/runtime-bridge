package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// wsUpgrader returns a WebSocket upgrader that validates the request Origin against
// the bridge's configured CORS allowlist. An empty Origin (curl, mobile, programmatic
// clients) is always allowed — browsers always send Origin on cross-origin requests,
// so absence of the header cannot be a browser cross-origin attempt.
func (b *Bridge) wsUpgrader() websocket.Upgrader {
	allowed := parseCORSOrigins(b.Config.DefaultCORSOrigins)
	return websocket.Upgrader{
		HandshakeTimeout: 10 * time.Second,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, a := range allowed {
				if strings.EqualFold(origin, a) {
					return true
				}
			}
			return false
		},
	}
}

// execMsg is the client→server envelope sent as JSON text frames.
//
//	type "input"  — data: raw bytes to write to the PTY stdin
//	type "resize" — cols/rows: new terminal dimensions
type execMsg struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols uint16 `json:"cols,omitempty"`
	Rows uint16 `json:"rows,omitempty"`
}

// handleExec upgrades to WebSocket and proxies a kubectl exec PTY session
// into the running pod for the given workspace.
//
// Query params (all optional):
//
//	container — which container to exec into (defaults to first container)
//	cmd       — command to run (defaults to /bin/sh)
//
// Protocol:
//   - Server → client: binary frames carrying raw PTY output
//   - Client → server: JSON text frames (execMsg)
func (b *Bridge) handleExec(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId"))
		return
	}

	container := r.URL.Query().Get("container")
	cmd := r.URL.Query().Get("cmd")
	if cmd == "" {
		cmd = "/bin/sh"
	}

	pod, err := b.findExecPod(r.Context(), instanceID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("no running pod: %w", err))
		return
	}
	if container == "" {
		container = pod.Spec.Containers[0].Name
	}

	upgrader := b.wsUpgrader()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		b.Logger.Printf("[exec] WebSocket upgrade failed for %s: %v", instanceID, err)
		return
	}
	defer conn.Close()

	b.Logger.Printf("[exec] open  ws=%s pod=%s container=%s cmd=%s", instanceID, pod.Name, container, cmd)

	execErr := b.streamExec(r.Context(), conn, pod.Name, instanceID, container, cmd)

	b.Logger.Printf("[exec] close ws=%s: %v", instanceID, execErr)
}

// findExecPod returns the first Running pod in the workspace namespace.
func (b *Bridge) findExecPod(ctx context.Context, namespace string) (*corev1.Pod, error) {
	pods, err := b.KubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for i := range pods.Items {
		if pods.Items[i].Status.Phase == corev1.PodRunning {
			return &pods.Items[i], nil
		}
	}
	return nil, fmt.Errorf("no running pod in namespace %s", namespace)
}

// streamExec wires up the SPDY executor to the WebSocket connection.
func (b *Bridge) streamExec(ctx context.Context, conn *websocket.Conn, podName, namespace, container, cmd string) error {
	execCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	sizeQueue := &wsSizeQueue{ch: make(chan remotecommand.TerminalSize, 4)}
	defer sizeQueue.drain()

	// Pipe WebSocket input → exec stdin. A goroutine pumps WS reads so that
	// blocking on ReadMessage doesn't hold up the exec stream lifecycle.
	stdinR, stdinW := io.Pipe()
	go func() {
		defer stdinW.Close()
		for {
			_, raw, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var m execMsg
			if json.Unmarshal(raw, &m) != nil {
				continue
			}
			switch m.Type {
			case "input":
				if _, err := stdinW.Write([]byte(m.Data)); err != nil {
					return
				}
			case "resize":
				sizeQueue.push(m.Cols, m.Rows)
			}
		}
	}()

	// When exec finishes (shell exit or error), unblock the WS reader above.
	defer conn.SetReadDeadline(time.Now())

	req := b.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   []string{cmd},
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(b.RESTConfig, http.MethodPost, req.URL())
	if err != nil {
		return fmt.Errorf("create spdy executor: %w", err)
	}

	out := &wsWriter{conn: conn}

	return executor.StreamWithContext(execCtx, remotecommand.StreamOptions{
		Stdin:             stdinR,
		Stdout:            out,
		Stderr:            out,
		Tty:               true,
		TerminalSizeQueue: sizeQueue,
	})
}

// wsWriter writes PTY output to the WebSocket as binary frames.
// gorilla/websocket requires serialised writes — the mutex enforces that.
type wsWriter struct {
	mu   sync.Mutex
	conn *websocket.Conn
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.conn.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// wsSizeQueue implements remotecommand.TerminalSizeQueue.
type wsSizeQueue struct {
	ch chan remotecommand.TerminalSize
}

func (q *wsSizeQueue) push(cols, rows uint16) {
	if cols == 0 || rows == 0 {
		return
	}
	select {
	case q.ch <- remotecommand.TerminalSize{Width: cols, Height: rows}:
	default: // drop if buffer full; client will send another resize
	}
}

// drain closes the channel so Next() unblocks and returns nil.
func (q *wsSizeQueue) drain() {
	// Closing a channel that might already be closed panics — use recover.
	defer func() { recover() }() //nolint:errcheck
	close(q.ch)
}

func (q *wsSizeQueue) Next() *remotecommand.TerminalSize {
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &size
}

// gatewayRunningFromOutput parses the output of `hermes gateway status`.
// The command exits 0 in both states, so we read the text instead:
// running outputs contain "running" without "not running".
func gatewayRunningFromOutput(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "running") && !strings.Contains(lower, "not running")
}

// podRunCommand runs a command in the given pod+container and captures stdout/stderr.
// Non-interactive — no PTY. Returns an error if the command exits non-zero.
func (b *Bridge) podRunCommand(ctx context.Context, ns, podName, container string, cmd []string) (stdout, stderr string, err error) {
	req := b.KubeClient.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   cmd,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(b.RESTConfig, http.MethodPost, req.URL())
	if err != nil {
		return "", "", fmt.Errorf("create executor: %w", err)
	}

	var outBuf, errBuf bytes.Buffer
	err = executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &outBuf,
		Stderr: &errBuf,
	})
	return outBuf.String(), errBuf.String(), err
}

// EnsureGatewayRunning checks if the Hermes gateway is running in the instance
// pod and starts it if not. Called after every integration config change.
// A start failure is logged but not returned — the integration config is already
// applied and the pod owns gateway lifecycle from this point.
func (b *Bridge) EnsureGatewayRunning(ctx context.Context, instanceID string) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pod, err := b.findExecPod(ctx, instanceID)
	if err != nil {
		b.Logger.Printf("[EnsureGatewayRunning] no running pod for %s: %v", instanceID, err)
		return
	}
	container := pod.Spec.Containers[0].Name

	stdout, _, _ := b.podRunCommand(ctx, instanceID, pod.Name, container, []string{"hermes", "gateway", "status"})
	if gatewayRunningFromOutput(stdout) {
		return // already running
	}

	b.Logger.Printf("[EnsureGatewayRunning] gateway not running in %s, starting...", instanceID)
	_, stderr, startErr := b.podRunCommand(ctx, instanceID, pod.Name, container,
		[]string{"hermes", "gateway", "start"})
	if startErr != nil {
		b.Logger.Printf("[EnsureGatewayRunning] gateway start failed for %s: %v (stderr: %s)",
			instanceID, startErr, strings.TrimSpace(stderr))
	}
}

// handleGatewayStatus GET /v1/instances/{id}/gateway/status
// Runs `hermes gateway status` in the instance pod and returns whether it's running.
func (b *Bridge) handleGatewayStatus(w http.ResponseWriter, r *http.Request) {
	instanceID := mux.Vars(r)["id"]
	if !validInstanceID(instanceID) {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid instanceId"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	pod, err := b.findExecPod(ctx, instanceID)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, fmt.Errorf("no running pod: %w", err))
		return
	}
	container := pod.Spec.Containers[0].Name

	stdout, stderr, _ := b.podRunCommand(ctx, instanceID, pod.Name, container,
		[]string{"hermes", "gateway", "status"})

	// hermes gateway status exits 0 regardless of state — parse output text.
	running := gatewayRunningFromOutput(stdout)
	output := strings.TrimSpace(stdout)
	if output == "" {
		output = strings.TrimSpace(stderr)
	}
	writeJSON(w, http.StatusOK, GatewayStatus{
		InstanceID: instanceID,
		Running:    running,
		Output:     output,
	})
}
