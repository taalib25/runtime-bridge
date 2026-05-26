/* HermesCloud extension — friction-reduction for new users */
/* Injected via HERMES_WEBUI_EXTENSION_SCRIPT_URLS — additive only, no core mutations */
(() => {
  'use strict';

  const SKIN_KEY    = 'hermes-skin';
  const HINTS_KEY   = 'hc-hints-dismissed';
  const TOUR_KEY    = 'hc-tour-done';
  const KB_HINT_KEY = 'hc-kb-hint-done';
  const BRAND_SKIN  = 'hermescloud';
  const SESSION_LS  = 'hermes-webui-session'; // key webui uses to persist last session

  const STARTER_HINTS = [
    'Help me start a Python project',
    'Explain this code to me',
    'Write a README for my project',
    'Fix the bug in this function',
    'Create a simple REST API',
  ];

  const TOUR_STEPS = [
    { labels: ['Chat', 'chat'],       title: 'Chat',     desc: 'Your main workspace. Just type what you need — the AI reads your files and writes code for you.' },
    { labels: ['Terminal','terminal'], title: 'Terminal', desc: 'A real terminal inside your instance. Run commands, install packages, or manage files.' },
    { labels: ['Tasks','tasks'],       title: 'Tasks',    desc: 'For longer jobs, the AI tracks progress here step by step.' },
    { labels: ['Memory','memory'],     title: 'Memory',   desc: 'Facts the AI remembers about you across sessions — your preferences, how you like code written.' },
    { labels: ['Settings','settings'], title: 'Settings', desc: 'Change appearance, font size, and theme. Your provider is managed by HermesCloud.' },
  ];

  const HIDDEN_SETTING_LABELS = [
    'Agent directory', 'Memory backend', 'State directory',
    'Extension dir', 'Extension scripts', 'Extension stylesheets',
    'HERMES_HOME', 'Agent dir',
  ];

  const SHELL_LANGS = new Set(['bash', 'sh', 'shell', 'console', 'zsh', 'fish']);

  const AUTH_PATTERNS = [
    /hermes\s+auth/i,
    /gh\s+auth/i,
    /git\s+config\s+.*user/i,
    /docker\s+login/i,
    /aws\s+configure/i,
    /gcloud\s+auth/i,
    /npm\s+(login|adduser)/i,
    /pip.*login/i,
    /ssh-keygen/i,
    /gpg\s+--gen-key/i,
  ];

  /* ── 1. Brand the tab ──────────────────────────────────────────── */
  function setTitle() {
    document.title = 'HermesCloud';
    new MutationObserver(() => {
      if (document.title !== 'HermesCloud') document.title = 'HermesCloud';
    }).observe(document.head, { childList: true, subtree: true });
  }

  /* ── 2. Apply brand skin — always, every page load ─────────────── */
  function applyDefaultSkin() {
    const skin = localStorage.getItem(SKIN_KEY) || BRAND_SKIN;
    localStorage.setItem(SKIN_KEY, skin);

    function apply() {
      if (document.documentElement.dataset.skin !== skin)
        document.documentElement.dataset.skin = skin;
    }
    apply();
    // Re-apply if webui resets data-skin during its own boot sequence
    new MutationObserver(apply).observe(document.documentElement, {
      attributes: true, attributeFilter: ['data-skin'],
    });
  }

  /* ── 3. No-provider banner ─────────────────────────────────────── */
  async function showNoProviderBanner() {
    if (document.getElementById('hc-no-provider-banner')) return;
    try {
      const res  = await fetch('/api/onboarding/status');
      if (!res.ok) return;
      const data = await res.json();
      if (data.system?.chat_ready !== false) return;
    } catch { return; }

    const banner = document.createElement('div');
    banner.id = 'hc-no-provider-banner';
    banner.innerHTML =
      '<span>⚠️ No AI provider configured for this instance.</span>' +
      '<button class="hc-link" id="hc-open-terminal">Open terminal</button>' +
      '<span>and run <code style="background:var(--code-bg);padding:1px 5px;border-radius:3px;font-size:0.9em">hermes-provider setup</code></span>';
    banner.querySelector('#hc-open-terminal').addEventListener('click', () => ensureTerminal());
    waitFor('main, [data-main], #app-root', (t) => t.insertAdjacentElement('afterbegin', banner));
  }

  /* ── 4. Intercept provider settings panel ──────────────────────── */
  function watchProviderSettings() {
    new MutationObserver(() => {
      const targets = [
        ...document.querySelectorAll('[data-settings-section="providers"],[data-section="providers"]'),
        ...[...document.querySelectorAll('h2,h3,[class*="settings-heading"]')]
          .filter((el) => /provider/i.test(el.textContent)),
      ];
      targets.forEach((el) => {
        if (el.dataset.hcPatched) return;
        el.dataset.hcPatched = '1';
        const redirect = document.createElement('div');
        redirect.id = 'hc-provider-redirect';
        redirect.innerHTML =
          '<strong>Provider settings are managed by HermesCloud.</strong><br>' +
          'Your AI provider and API key are pre-configured. To change them, ' +
          'open the <button class="hc-link" style="font:inherit;text-decoration:underline;' +
          'background:none;border:none;color:var(--accent);cursor:pointer" ' +
          'id="hc-redirect-terminal">terminal</button> and run <code>hermes-provider setup</code>.';
        redirect.querySelector('#hc-redirect-terminal')?.addEventListener('click', () => ensureTerminal());
        el.after(redirect);
        el.style.display = 'none';
      });
    }).observe(document.body, { childList: true, subtree: true });
  }

  /* ── 5. Hide internal settings rows ────────────────────────────── */
  function hideInternalSettings() {
    new MutationObserver(() => {
      document.querySelectorAll('label,[class*="settings-label"],[class*="setting-row"],[class*="form-row"]')
        .forEach((el) => {
          if (el.dataset.hcHidden) return;
          const text = el.textContent.trim();
          if (HIDDEN_SETTING_LABELS.some((l) => text.startsWith(l))) {
            el.dataset.hcHidden = '1';
            (el.closest('[class*="settings-row"],[class*="form-row"],[class*="setting-item"],li,tr') ?? el)
              .style.display = 'none';
          }
        });
    }).observe(document.body, { childList: true, subtree: true });
  }

  /* ── 6. Starter hints ──────────────────────────────────────────── */
  function showStarterHints() {
    if (localStorage.getItem(HINTS_KEY)) return;
    waitFor('textarea[name="message"],#composer-input,[data-composer],textarea[placeholder]', (composer) => {
      if (document.getElementById('hc-starter-hints')) return;
      const container = document.createElement('div');
      container.id = 'hc-starter-hints';
      STARTER_HINTS.forEach((text) => {
        const chip = document.createElement('button');
        chip.className = 'hc-hint-chip';
        chip.textContent = text;
        chip.addEventListener('click', () => {
          composer.value = text;
          composer.dispatchEvent(new Event('input', { bubbles: true }));
          composer.focus();
          dismiss();
        });
        container.appendChild(chip);
      });
      (composer.closest('form') ?? composer.parentElement)?.insertAdjacentElement('beforebegin', container);
      composer.addEventListener('input', () => { if (composer.value.length > 0) dismiss(); }, { once: true });
      function dismiss() {
        document.getElementById('hc-starter-hints')?.remove();
        localStorage.setItem(HINTS_KEY, '1');
      }
    });
  }

  /* ── 7. Sidebar tour ───────────────────────────────────────────── */
  function runSidebarTour() {
    if (localStorage.getItem(TOUR_KEY)) return;
    waitFor('nav,[data-sidebar],aside', () => {
      const highlight = document.createElement('div'); highlight.id = 'hc-tour-highlight';
      const popover   = document.createElement('div'); popover.id   = 'hc-tour-popover';
      document.body.appendChild(highlight);
      document.body.appendChild(popover);

      function findNavItem(labels) {
        for (const label of labels) {
          const el = document.querySelector(`[aria-label="${label}"]`) ??
            document.querySelector(`[title="${label}"]`) ??
            [...document.querySelectorAll('nav a,nav button,[data-sidebar] a,[data-sidebar] button,aside a,aside button')]
              .find((el) => el.textContent.trim() === label || el.getAttribute('aria-label') === label);
          if (el) return el;
        }
        return null;
      }

      const validSteps = TOUR_STEPS.filter((s) => findNavItem(s.labels));

      function showStep(index) {
        if (index >= validSteps.length) { endTour(); return; }
        const step = validSteps[index];
        const target = findNavItem(step.labels);
        if (!target) { showStep(index + 1); return; }
        const rect = target.getBoundingClientRect();
        highlight.style.cssText = `top:${rect.top-3}px;left:${rect.left-3}px;width:${rect.width+6}px;height:${rect.height+6}px;display:block`;
        const popLeft = rect.right + 12;
        const popTop  = Math.min(rect.top, window.innerHeight - 220);
        popover.innerHTML = `<strong>${step.title}</strong><span>${step.desc}</span>
          <div class="hc-tour-actions">
            <button class="hc-tour-skip">Skip tour</button>
            <span class="hc-tour-step">${index+1} / ${validSteps.length}</span>
            <button class="hc-tour-next">${index+1 < validSteps.length ? 'Next →' : 'Done'}</button>
          </div>`;
        popover.style.cssText = `left:${Math.min(popLeft,window.innerWidth-260)}px;top:${popTop}px;display:block`;
        popover.querySelector('.hc-tour-next').onclick = () => showStep(index + 1);
        popover.querySelector('.hc-tour-skip').onclick = endTour;
      }

      function endTour() {
        highlight.remove(); popover.remove();
        localStorage.setItem(TOUR_KEY, '1');
      }
      setTimeout(() => showStep(0), 800);
    });
  }

  /* ── 8. ensureTerminal — navigate to session then open terminal ── */
  //
  // Full flow:
  //   a) If no session is loaded (S.session is null), load the last-used session
  //      from localStorage, or create a new one via newSession().
  //   b) Call toggleComposerTerminal(true) — this opens the panel AND starts
  //      the terminal session, setting TERMINAL_UI.sessionId when ready.
  //   c) Wait for TERMINAL_UI.sessionId to be populated.
  //   d) Return the session ID so the caller can send a command.
  //
  async function ensureTerminal() {
    // ── a. Ensure a chat session is loaded ──────────────────────────
    const hasSesh = () => typeof S !== 'undefined' && S.session && S.session.session_id;

    if (!hasSesh()) {
      const lastSid = localStorage.getItem(SESSION_LS);
      if (lastSid && typeof loadSession === 'function') {
        try { await loadSession(lastSid); } catch (_) {}
      }
      if (!hasSesh() && typeof newSession === 'function') {
        try { await newSession(); } catch (_) {}
      }
    }

    // ── b. Open terminal panel (webui global or DOM fallback) ───────
    if (typeof toggleComposerTerminal === 'function') {
      try { await toggleComposerTerminal(true); } catch (_) {}
    } else {
      // DOM fallback — find the terminal nav button
      const btn =
        document.querySelector('[aria-label="Terminal"],[aria-label="terminal"],[data-panel="terminal"],[title="Terminal"]') ??
        [...document.querySelectorAll('nav a,nav button,aside a,aside button')]
          .find((el) => /terminal/i.test(el.textContent + el.getAttribute('aria-label') + el.getAttribute('title')));
      btn?.click();
    }

    // ── c. Wait for TERMINAL_UI.sessionId (set by _startComposerTerminal) ──
    const termSid = await new Promise((resolve) => {
      const check = () => typeof TERMINAL_UI !== 'undefined' && TERMINAL_UI.sessionId;
      if (check()) { resolve(TERMINAL_UI.sessionId); return; }
      let elapsed = 0;
      const id = setInterval(() => {
        elapsed += 150;
        if (check()) { clearInterval(id); resolve(TERMINAL_UI.sessionId); }
        else if (elapsed > 6000) { clearInterval(id); resolve(null); }
      }, 150);
    });

    return termSid;
  }

  /* ── 9. runInTerminal — ensure session + terminal, then send ───── */
  async function runInTerminal(command) {
    const sid = await ensureTerminal();

    if (!sid) {
      hcToast('Could not start a terminal session — please open the Terminal tab manually and try again.', 4500, 'warn');
      return;
    }

    try {
      await fetch('/api/terminal/input', {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sid, data: command + '\r' }),
      });
      hcToast('Running in terminal…', 1800);
    } catch {
      hcToast('Could not send to terminal — is the session still open?', 3000, 'error');
    }
  }

  /* ── 10. Watch code blocks — add Run buttons ────────────────────── */
  function watchCodeBlocks() {
    const target =
      document.getElementById('msgInner') ??
      document.getElementById('messages') ??
      document.querySelector('[data-messages],[data-chat-messages]');

    if (!target) { setTimeout(watchCodeBlocks, 1200); return; }

    function processBlock(header) {
      if (header.dataset.hcRun) return;
      const pre  = header.nextElementSibling;
      if (!pre || pre.tagName !== 'PRE') return;
      const code = pre.querySelector('code');
      if (!code) return;
      const langClass = [...code.classList].find((c) => c.startsWith('language-'));
      const lang = langClass ? langClass.replace('language-', '') : '';
      if (!SHELL_LANGS.has(lang)) return;

      header.dataset.hcRun = '1';
      const command = code.textContent.trim();
      const isAuth  = AUTH_PATTERNS.some((p) => p.test(command));

      const btn = document.createElement('button');
      btn.className   = isAuth ? 'hc-run-btn hc-run-btn--auth' : 'hc-run-btn';
      btn.textContent = isAuth ? '▶ Run (needs your input)' : '▶ Run';
      btn.title       = isAuth
        ? 'Opens terminal and runs this command — you may need to interact with it'
        : 'Open terminal and run this command';
      btn.addEventListener('click', (e) => { e.stopPropagation(); runInTerminal(command); });
      header.appendChild(btn);

      // Auth commands: pre-open the terminal immediately so it's ready
      if (isAuth) ensureTerminal();
    }

    target.querySelectorAll('.pre-header').forEach(processBlock);
    new MutationObserver(() => {
      target.querySelectorAll('.pre-header:not([data-hc-run])').forEach(processBlock);
    }).observe(target, { childList: true, subtree: true });
  }

  /* ── 11. Keyboard shortcut hint ────────────────────────────────── */
  function showKeyboardHint() {
    if (localStorage.getItem(KB_HINT_KEY)) return;
    waitFor('button[type="submit"],[data-send-btn],#btnSend,button[aria-label*="Send" i]', (sendBtn) => {
      if (document.querySelector('.hc-kb-hint')) return;
      const isMac = /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);
      const hint  = document.createElement('span');
      hint.className   = 'hc-kb-hint';
      hint.textContent = isMac ? '⌘↵ to send' : 'Ctrl↵ to send';
      sendBtn.insertAdjacentElement('beforebegin', hint);
      const messages = document.getElementById('messages') ?? document.getElementById('msgInner');
      if (!messages) return;
      new MutationObserver((_, obs) => {
        if (messages.querySelector('[class*="user-msg"],[data-role="user"],.msg-user')) {
          hint.classList.add('hc-kb-hint--fade');
          setTimeout(() => { hint.remove(); localStorage.setItem(KB_HINT_KEY, '1'); }, 600);
          obs.disconnect();
        }
      }).observe(messages, { childList: true, subtree: true });
    });
  }

  /* ── Helper: self-contained toast (no webui dependency) ────────── */
  function hcToast(message, duration = 2400, type = 'info') {
    if (typeof showToast === 'function') {
      showToast(message, duration, type === 'error' ? 'error' : undefined);
      return;
    }
    document.getElementById('hc-toast')?.remove();
    const toast = document.createElement('div');
    toast.id = 'hc-toast';
    const bg = type === 'error' ? '#c0392b' : type === 'warn' ? '#e67e22' : '#2d6cdf';
    toast.style.cssText = `position:fixed;bottom:24px;left:50%;transform:translateX(-50%);background:${bg};color:#fff;padding:10px 18px;border-radius:8px;font-size:13px;z-index:99999;box-shadow:0 4px 16px rgba(0,0,0,0.25);max-width:420px;text-align:center;pointer-events:none`;
    toast.textContent = message;
    document.body.appendChild(toast);
    setTimeout(() => toast.remove(), duration);
  }

  /* ── Helper: poll for element then run callback ─────────────────── */
  function waitFor(selector, cb, timeout = 8000) {
    const el = document.querySelector(selector);
    if (el) { cb(el); return; }
    const start = Date.now();
    const id = setInterval(() => {
      const found = document.querySelector(selector);
      if (found) { clearInterval(id); cb(found); }
      else if (Date.now() - start > timeout) clearInterval(id);
    }, 300);
  }

  /* ── Boot ──────────────────────────────────────────────────────── */
  function boot() {
    setTitle();
    applyDefaultSkin();
    showNoProviderBanner();
    watchProviderSettings();
    hideInternalSettings();
    showStarterHints();
    runSidebarTour();
    watchCodeBlocks();
    showKeyboardHint();
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', boot);
  } else {
    boot();
  }
})();
