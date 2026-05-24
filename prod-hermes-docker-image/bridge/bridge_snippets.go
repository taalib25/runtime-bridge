package bridge

// This file is a reference snippet, not drop-in production code.
// Move the functions into your bridge/helm.go buildValues() flow.

func hermesPersistentEnv() map[string]string {
	h := "/home/hermeswebui/.hermes"
	return map[string]string{
		"HERMES_WEBUI_HOST":              "0.0.0.0",
		"HERMES_WEBUI_STATE_DIR":         h + "/webui",
		"HERMES_WEBUI_DEFAULT_WORKSPACE": "/workspace",
		"HERMES_HOME":                    h,
		"HERMES_SKIP_SETUP":              "1",
		"HERMES_EXEC_ASK":                "false",
		"HERMES_TIMEZONE":                "UTC",

		// User/agent-installed tools persist in PVC.
		"PATH":               h + "/bin:/home/hermeswebui/.local/bin:/opt/hermes-webui/.venv/bin:/usr/local/bin:/usr/bin:/bin",
		"GH_CONFIG_DIR":      h + "/gh",
		"XDG_CONFIG_HOME":    h + "/.config",
		"PYTHONUSERBASE":     h + "/python",
		"PIP_CACHE_DIR":      h + "/cache/pip",
		"PIPX_HOME":          h + "/pipx",
		"PIPX_BIN_DIR":       h + "/bin",
		"UV_CACHE_DIR":       h + "/cache/uv",
		"UV_TOOL_DIR":        h + "/uv/tools",
		"UV_TOOL_BIN_DIR":    h + "/bin",
		"NPM_CONFIG_PREFIX":  h + "/npm",
		"NPM_CONFIG_CACHE":   h + "/cache/npm",
		"PNPM_HOME":          h + "/pnpm",
		"YARN_GLOBAL_FOLDER": h + "/yarn/global",
		"YARN_CACHE_FOLDER":  h + "/cache/yarn",
		"COREPACK_HOME":      h + "/corepack",
		"BUN_INSTALL":        h + "/bun",
		"DENO_INSTALL":       h + "/deno",
		"CARGO_HOME":         h + "/cargo",
		"RUSTUP_HOME":        h + "/rustup",
		"GOPATH":             h + "/go",
		"GOBIN":              h + "/bin",
		"GEM_HOME":           h + "/gem",
		"GEM_PATH":           h + "/gem",
		"COMPOSER_HOME":      h + "/composer",
		"DOTNET_CLI_HOME":    h + "/dotnet",
	}
}

// RuntimeImageMode controls whether the chart should seed Hermes Agent at pod boot.
// init-install: use ghcr.io/ashneil12/hermes-webui + install-agent init container.
// prebuilt: use ghcr.io/<you>/hermescloud-runtime + no install-agent init container.
type RuntimeImageMode string

const (
	RuntimeImageModeInitInstall RuntimeImageMode = "init-install"
	RuntimeImageModePrebuilt    RuntimeImageMode = "prebuilt"
)

func hermesWebUIAgentDir(mode RuntimeImageMode) string {
	if mode == RuntimeImageModePrebuilt {
		return "/opt/hermes-agent"
	}
	return "/home/hermeswebui/.hermes/hermes-agent"
}

func shouldAddInstallAgentInitContainer(mode RuntimeImageMode) bool {
	return mode == RuntimeImageModeInitInstall
}

// In buildValues():
//
// env := hermesPersistentEnv()
// env["HERMES_WEBUI_PORT"] = strconv.Itoa(spec.RuntimePort)
// env["HERMES_WEBUI_AGENT_DIR"] = hermesWebUIAgentDir(spec.RuntimeImageMode)
// for k, v := range env {
//   if _, exists := spec.EnvMap[k]; !exists {
//     spec.EnvMap[k] = v
//   }
// }
//
// if shouldAddInstallAgentInitContainer(spec.RuntimeImageMode) {
//   values["extraInitContainers"] = append(values["extraInitContainers"].([]any), map[string]any{
//     "name": "install-agent",
//     "image": spec.AgentImage + ":" + spec.AgentImageTag,
//     "imagePullPolicy": "IfNotPresent",
//     "command": []any{"sh", "-c", `set -e
//       dst=/mnt/hermes-home/hermes-agent
//       if [ -d "$dst" ] && [ -f "$dst/pyproject.toml" ]; then
//         echo "hermes-agent already present, skipping"
//         exit 0
//       fi
//       if [ -d /opt/hermes ] && [ -f /opt/hermes/pyproject.toml ]; then
//         mkdir -p "$dst"
//         cp -a /opt/hermes/. "$dst/"
//       elif [ -d /app ] && [ -f /app/pyproject.toml ]; then
//         mkdir -p "$dst"
//         cp -a /app/. "$dst/"
//       else
//         echo "ERROR: agent source not found" >&2
//         find / -maxdepth 4 -name pyproject.toml 2>/dev/null | head -50 >&2
//         exit 1
//       fi
//       chown -R 1024:1024 /mnt/hermes-home
//       echo "hermes-agent installed"`},
//     "volumeMounts": []any{map[string]any{"name": "data", "mountPath": "/mnt"}},
//   })
// }
