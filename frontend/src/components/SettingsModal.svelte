<script>
  import { SaveSettings, GetConfig, TokenFromEnv } from '../../wailsjs/go/main/App.js'
  import { onMount } from 'svelte'

  export let onClose        // called with no args when user cancels (only if app is already configured)
  export let onSaved        // called after a successful save
  export let required = false  // true = first-run, no cancel allowed

  const DEFAULT_URL = 'https://canvas.instructure.com'

  let baseURL = DEFAULT_URL
  let apiToken = ''
  let showToken = false
  let saving = false
  let error = ''
  let tokenLocked = false   // true when token comes from env var

  onMount(async () => {
    try {
      const [cfg, fromEnv] = await Promise.all([GetConfig(), TokenFromEnv()])
      tokenLocked = fromEnv
      if (cfg) {
        baseURL   = cfg.canvas_base_url || DEFAULT_URL
        apiToken  = fromEnv ? '(set via CANVAS_API_TOKEN env var)' : (cfg.api_token || '')
      }
    } catch (_) { /* best-effort */ }
  })

  async function handleSave() {
    error = ''
    if (!baseURL.trim())  { error = 'Canvas Base URL is required.'; return }
    if (!apiToken.trim()) { error = 'API Token is required.'; return }

    saving = true
    try {
      await SaveSettings(baseURL.trim(), apiToken.trim())
      onSaved()
    } catch (e) {
      error = e.message || e.error || 'Failed to save settings.'
    } finally {
      saving = false
    }
  }
</script>

<div class="modal-overlay" on:click={() => { if (!required && !saving) onClose() }}>
  <div class="modal" on:click|stopPropagation>

    <div class="modal-header">
      <h2>Settings</h2>
      {#if !required && !saving}
        <button class="close-btn" on:click={onClose} title="Close">✕</button>
      {/if}
    </div>

    {#if tokenLocked}
      <div class="notice info">
        <strong>Token from environment:</strong> <code>CANVAS_API_TOKEN</code> is set.
        The token field is read-only; unset the env var to manage it here.
      </div>
    {/if}

    <form on:submit|preventDefault={handleSave}>

      <div class="field">
        <label for="base-url">Canvas Base URL</label>
        <input
          id="base-url"
          type="url"
          bind:value={baseURL}
          placeholder={DEFAULT_URL}
          disabled={saving}
          spellcheck="false"
        />
        <span class="hint">e.g. https://yourschool.instructure.com</span>
      </div>

      <div class="field">
        <label for="token">API Token</label>
        <div class="token-row">
          {#if showToken}
            <input
              id="token"
              type="text"
              bind:value={apiToken}
              placeholder="Paste your Canvas API token"
              disabled={saving || tokenLocked}
              spellcheck="false"
              class="token-input"
            />
          {:else}
            <input
              id="token"
              type="password"
              bind:value={apiToken}
              placeholder="Paste your Canvas API token"
              disabled={saving || tokenLocked}
              class="token-input"
            />
          {/if}
          <button
            type="button"
            class="toggle-btn"
            on:click={() => showToken = !showToken}
            title={showToken ? 'Hide token' : 'Show token'}
            disabled={tokenLocked}
          >
            {showToken ? '🙈' : '👁'}
          </button>
        </div>
        <span class="hint">
          Generate one in Canvas → Account → Settings → New Access Token.
        </span>
      </div>

      {#if error}
        <div class="notice error">{error}</div>
      {/if}

      <div class="modal-footer">
        {#if !required}
          <button type="button" class="secondary" on:click={onClose} disabled={saving}>
            Cancel
          </button>
        {/if}
        <button type="submit" class="primary" disabled={saving || tokenLocked}>
          {saving ? 'Saving…' : 'Save & Connect'}
        </button>
      </div>

    </form>
  </div>
</div>

<style>
  .modal { max-width: 460px; }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 20px;
  }

  .modal-header h2 { margin: 0; }

  .close-btn {
    background: none;
    border: none;
    font-size: 16px;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px 6px;
  }
  .close-btn:hover { color: var(--text-primary); }

  form {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  .field label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .hint {
    font-size: 11px;
    color: var(--text-secondary);
  }

  /* Token row */
  .token-row {
    display: flex;
    gap: 6px;
  }

  .token-input { flex: 1; }

  .toggle-btn {
    padding: 0 10px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    cursor: pointer;
    font-size: 14px;
    flex-shrink: 0;
  }
  .toggle-btn:disabled { opacity: 0.4; cursor: default; }

  /* Notices */
  .notice {
    padding: 10px 14px;
    border-radius: 8px;
    font-size: 12px;
    line-height: 1.5;
  }

  .notice.info {
    background: #eaf2f8;
    color: #1a5276;
    border: 1px solid #aed6f1;
  }

  .notice.error {
    background: #fde8e8;
    color: #a93226;
    border: 1px solid #f1948a;
  }

  .notice code {
    font-family: monospace;
    font-size: 11px;
    background: rgba(0,0,0,0.08);
    padding: 1px 4px;
    border-radius: 3px;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 4px;
  }
</style>
