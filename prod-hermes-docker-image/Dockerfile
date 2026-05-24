# syntax=docker/dockerfile:1.7

ARG HERMES_AGENT_IMAGE=nousresearch/hermes-agent:latest

FROM ${HERMES_AGENT_IMAGE} AS agent_source
SHELL ["/bin/sh", "-c"]
RUN set -eux; \
    mkdir -p /agent-out; \
    if [ -d /opt/hermes ] && [ -f /opt/hermes/pyproject.toml ]; then \
      cp -a /opt/hermes/. /agent-out/; \
    elif [ -d /app ] && [ -f /app/pyproject.toml ]; then \
      cp -a /app/. /agent-out/; \
    elif [ -d /workspace ] && [ -f /workspace/pyproject.toml ]; then \
      cp -a /workspace/. /agent-out/; \
    else \
      echo "ERROR: could not find hermes-agent source with pyproject.toml" >&2; \
      find / -maxdepth 4 \( -name pyproject.toml -o -name '*hermes*' \) 2>/dev/null | head -100 >&2; \
      exit 1; \
    fi; \
    test -f /agent-out/pyproject.toml

FROM python:3.12-slim AS runtime

ARG HERMES_WEBUI_REPO=https://github.com/nesquena/hermes-webui.git
ARG HERMES_WEBUI_REF=main
ARG BUILD_DATE=unknown
ARG VCS_REF=unknown

LABEL org.opencontainers.image.title="HermesCloud Runtime" \
      org.opencontainers.image.description="Prebuilt Hermes WebUI + Hermes Agent runtime for HermesCloud pods" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.revision="${VCS_REF}"

ENV DEBIAN_FRONTEND=noninteractive \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

ENV HERMES_HOME=/home/hermeswebui/.hermes \
    HERMES_WEBUI_HOST=0.0.0.0 \
    HERMES_WEBUI_PORT=8787 \
    HERMES_WEBUI_STATE_DIR=/home/hermeswebui/.hermes/webui \
    HERMES_WEBUI_DEFAULT_WORKSPACE=/workspace \
    HERMES_WEBUI_AGENT_DIR=/opt/hermes-agent \
    HERMES_WEBUI_SKIP_ONBOARDING=1 \
    HERMES_EXEC_ASK=false \
    HERMES_TIMEZONE=UTC \
    HERMES_WEBUI_EXTENSION_DIR=/opt/hermescloud-extension \
    HERMES_WEBUI_EXTENSION_STYLESHEET_URLS=/extensions/hermescloud.css \
    HERMES_WEBUI_EXTENSION_SCRIPT_URLS=/extensions/hermescloud.js

# All tool/cache paths point into the PVC so user-installed tools survive pod restarts.
ENV PATH=/home/hermeswebui/.hermes/bin:/home/hermeswebui/.local/bin:/opt/hermes-webui/.venv/bin:/usr/local/bin:/usr/bin:/bin \
    GH_CONFIG_DIR=/home/hermeswebui/.hermes/gh \
    XDG_CONFIG_HOME=/home/hermeswebui/.hermes/.config \
    PYTHONUSERBASE=/home/hermeswebui/.hermes/python \
    PIP_CACHE_DIR=/home/hermeswebui/.hermes/cache/pip \
    PIPX_HOME=/home/hermeswebui/.hermes/pipx \
    PIPX_BIN_DIR=/home/hermeswebui/.hermes/bin \
    UV_CACHE_DIR=/home/hermeswebui/.hermes/cache/uv \
    UV_TOOL_DIR=/home/hermeswebui/.hermes/uv/tools \
    UV_TOOL_BIN_DIR=/home/hermeswebui/.hermes/bin \
    NPM_CONFIG_PREFIX=/home/hermeswebui/.hermes/npm \
    NPM_CONFIG_CACHE=/home/hermeswebui/.hermes/cache/npm \
    PNPM_HOME=/home/hermeswebui/.hermes/pnpm \
    YARN_GLOBAL_FOLDER=/home/hermeswebui/.hermes/yarn/global \
    YARN_CACHE_FOLDER=/home/hermeswebui/.hermes/cache/yarn \
    COREPACK_HOME=/home/hermeswebui/.hermes/corepack \
    BUN_INSTALL=/home/hermeswebui/.hermes/bun \
    DENO_INSTALL=/home/hermeswebui/.hermes/deno \
    CARGO_HOME=/home/hermeswebui/.hermes/cargo \
    RUSTUP_HOME=/home/hermeswebui/.hermes/rustup \
    GOPATH=/home/hermeswebui/.hermes/go \
    GOBIN=/home/hermeswebui/.hermes/bin \
    GEM_HOME=/home/hermeswebui/.hermes/gem \
    GEM_PATH=/home/hermeswebui/.hermes/gem \
    COMPOSER_HOME=/home/hermeswebui/.hermes/composer \
    DOTNET_CLI_HOME=/home/hermeswebui/.hermes/dotnet

RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      jq \
      ripgrep \
      unzip \
      build-essential \
      nodejs \
      npm \
      tini \
      procps \
      lsof \
      sudo \
      uidmap

RUN npm install -g pnpm yarn --no-audit --no-fund && npm cache clean --force

# Users own their pod — full sudo mirrors a VPS experience within the container boundary.
# Direct /etc/passwd write avoids depending on adduser/addgroup/useradd which differ
# across Debian versions (addgroup removed in adduser 3.x on trixie).
RUN echo 'hermeswebui:x:1024:1024::/home/hermeswebui:/bin/sh' >> /etc/passwd \
    && echo 'hermeswebui:x:1024:' >> /etc/group \
    && echo 'hermeswebui:!:19900:0:99999:7:::' >> /etc/shadow \
    && mkdir -p /home/hermeswebui/.hermes /opt/hermes-agent /opt/hermes-webui /workspace \
    && chown -R 1024:1024 /home/hermeswebui /opt/hermes-agent /opt/hermes-webui /workspace \
    && echo 'hermeswebui ALL=(root) NOPASSWD: ALL' > /etc/sudoers.d/hermeswebui \
    && chmod 440 /etc/sudoers.d/hermeswebui \
    && echo 'hermeswebui:100000:65536' >> /etc/subuid \
    && echo 'hermeswebui:100000:65536' >> /etc/subgid

COPY --from=agent_source --chown=1024:1024 /agent-out /opt/hermes-agent

RUN git clone --depth 1 --branch "${HERMES_WEBUI_REF}" "${HERMES_WEBUI_REPO}" /opt/hermes-webui

WORKDIR /opt/hermes-webui

RUN --mount=type=cache,target=/root/.cache/pip \
    python -m venv /opt/hermes-webui/.venv \
    && /opt/hermes-webui/.venv/bin/pip install --upgrade pip setuptools wheel \
    && if [ -f requirements.txt ]; then /opt/hermes-webui/.venv/bin/pip install -r requirements.txt; fi \
    && if [ -f pyproject.toml ]; then /opt/hermes-webui/.venv/bin/pip install -e .; fi \
    && /opt/hermes-webui/.venv/bin/pip install /opt/hermes-agent \
    && /opt/hermes-webui/.venv/bin/python -c "import run_agent; print('run_agent import ok')"

COPY --chown=1024:1024 extension/ /opt/hermescloud-extension/
COPY --chown=1024:1024 start.sh /usr/local/bin/start-hermescloud-runtime
COPY --chown=1024:1024 scripts/healthcheck.sh /usr/local/bin/hermescloud-healthcheck
COPY --chown=1024:1024 scripts/diagnose.sh /usr/local/bin/hermescloud-diagnose
RUN chmod +x /usr/local/bin/start-hermescloud-runtime \
              /usr/local/bin/hermescloud-healthcheck \
              /usr/local/bin/hermescloud-diagnose

USER 1024:1024

EXPOSE 8787

HEALTHCHECK --interval=20s --timeout=5s --start-period=60s --retries=10 \
    CMD /usr/local/bin/hermescloud-healthcheck

ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/start-hermescloud-runtime"]
