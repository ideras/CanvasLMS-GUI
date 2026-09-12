<script>
  import { BrowseDirectory, ExportAssignmentSubmissions } from '../../wailsjs/go/main/App.js'
  import { onMount, onDestroy } from 'svelte'
  import { toasts } from 'svelte-toasts'

  export let course
  export let assignment   // full Assignment object
  export let onClose

  let selectedDir = ''
  let state = 'idle'   // idle | downloading | done | error
  let progress = { current: 0, total: 0, student: '' }
  let resultMsg = ''
  let errorMsg = ''

  async function browseFolder() {
    try {
      const dir = await BrowseDirectory()
      if (dir) selectedDir = dir
    } catch (e) {
      toasts.error(e.message || 'Could not open folder picker')
    }
  }

  function startDownload() {
    if (!selectedDir) return
    state = 'downloading'
    progress = { current: 0, total: 0, student: '' }
    ExportAssignmentSubmissions(course.id, assignment.id, selectedDir)
  }

  onMount(() => {
    window.runtime.EventsOn('assign-dl:start', () => {
      state = 'downloading'
      progress = { current: 0, total: 0, student: '' }
    })
    window.runtime.EventsOn('assign-dl:progress', (p) => {
      progress = p
    })
    window.runtime.EventsOn('assign-dl:done', (d) => {
      state = 'done'
      resultMsg = d.message
    })
    window.runtime.EventsOn('assign-dl:error', (e) => {
      state = 'error'
      errorMsg = e.error || 'Download failed'
    })
  })

  onDestroy(() => {
    window.runtime.EventsOff('assign-dl:start', 'assign-dl:progress', 'assign-dl:done', 'assign-dl:error')
  })

  $: pct = progress.total > 0 ? (progress.current / progress.total) * 100 : 0
  $: canStart = selectedDir !== '' && state === 'idle'
</script>

<div class="modal-overlay" on:click={() => { if (state !== 'downloading') onClose() }}>
  <div class="modal" on:click|stopPropagation>

    <!-- Header -->
    <div class="modal-header">
      <div>
        <h2>Download Submissions</h2>
        <p class="subtitle">{assignment.name}</p>
      </div>
      {#if state !== 'downloading'}
        <button class="close-btn" on:click={onClose} title="Close">✕</button>
      {/if}
    </div>

    <!-- Folder selector -->
    <div class="field">
      <label>Output Folder</label>
      <div class="folder-row">
        <span class="folder-path" class:placeholder={!selectedDir}>
          {selectedDir || 'No folder selected'}
        </span>
        <button
          class="secondary browse-btn"
          on:click={browseFolder}
          disabled={state === 'downloading'}
        >
          Browse…
        </button>
      </div>
      <p class="hint">
        One sub-folder per student will be created inside this folder.
      </p>
    </div>

    <!-- Progress / result area -->
    {#if state === 'downloading'}
      <div class="progress-section">
        <div class="progress-track">
          {#if progress.total === 0}
            <div class="progress-fill indeterminate"></div>
          {:else}
            <div class="progress-fill" style="width: {pct}%"></div>
          {/if}
        </div>
        <p class="progress-label">
          {#if progress.total > 0}
            {progress.current} / {progress.total} — {progress.student}
          {:else}
            Starting…
          {/if}
        </p>
      </div>

    {:else if state === 'done'}
      <div class="result success">
        <span class="result-icon">✓</span>
        {resultMsg}
      </div>

    {:else if state === 'error'}
      <div class="result error">
        <span class="result-icon">✗</span>
        {errorMsg}
      </div>
    {/if}

    <!-- Footer -->
    <div class="modal-footer">
      {#if state === 'done' || state === 'error'}
        <button class="primary" on:click={onClose}>Close</button>
      {:else}
        <button class="secondary" on:click={onClose} disabled={state === 'downloading'}>Cancel</button>
        <button class="primary" on:click={startDownload} disabled={!canStart}>
          Download
        </button>
      {/if}
    </div>

  </div>
</div>

<style>
  .modal {
    max-width: 480px;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 20px;
  }

  .modal-header h2 { margin: 0 0 2px; }

  .subtitle {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 16px;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 2px 6px;
  }

  .close-btn:hover { color: var(--text-primary); }

  /* Folder row */
  .field {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 18px;
  }

  .field label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .folder-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .folder-path {
    flex: 1;
    padding: 7px 10px;
    font-size: 12px;
    font-family: monospace;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--text-primary);
  }

  .folder-path.placeholder {
    color: var(--text-secondary);
  }

  .browse-btn {
    padding: 6px 14px;
    font-size: 12px;
    white-space: nowrap;
    flex-shrink: 0;
  }

  .hint {
    font-size: 11px;
    color: var(--text-secondary);
    margin: 0;
  }

  /* Progress */
  .progress-section {
    margin-bottom: 18px;
  }

  .progress-track {
    height: 6px;
    background: var(--border-color);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 6px;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--frost-mid), var(--frost-blue));
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .progress-fill.indeterminate {
    width: 35%;
    animation: slide 1.4s ease-in-out infinite;
  }

  @keyframes slide {
    0%   { margin-left: 0;    width: 30%; }
    50%  { margin-left: 60%;  width: 30%; }
    100% { margin-left: 0;    width: 30%; }
  }

  .progress-label {
    font-size: 12px;
    color: var(--text-secondary);
    margin: 0;
    font-weight: 500;
  }

  /* Result */
  .result {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 500;
    margin-bottom: 18px;
  }

  .result.success {
    background: #d4efdf;
    color: #1e8449;
  }

  .result.error {
    background: #fde8e8;
    color: #a93226;
  }

  .result-icon {
    font-size: 16px;
    flex-shrink: 0;
  }

  /* Footer */
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
  }
</style>
