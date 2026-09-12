<script>
  import { UploadGrades, CancelUpload, BrowseCSVFile } from '../../bindings/canvaslms-gui/app.js'
  import { Events } from '@wailsio/runtime'

  export let course
  export let assignment
  export let onClose

  // Wizard state: 'file' | 'validate' | 'upload' | 'done'
  let step = 'file'
  let csvPath = ''
  let fileProgress = { done: 0, total: 0, student: '', file: '' }
  let batchProgress = { completion: 0, state: 'not_started' }
  let error = null
  let statusMsg = ''


  function startUpload() {
    if (!csvPath.trim()) return
    step = 'upload'
    error = null
    UploadGrades(course.id, assignment.assignment_id || assignment.id, csvPath.trim())
  }

  function cancel() {
    CancelUpload()
    step = 'file'
  }

  import { onMount } from 'svelte'

  onMount(() => {
    // Store unsubscribe functions: Events.Off(name) would remove ALL listeners
    // for the event, including the root app's upload:done/upload:error handlers.
    const unsubs = [
      Events.On('upload:file_start', (p) => {
        fileProgress = { done: 0, total: p.data.total, students: p.data.students, student: '', file: '' }
      }),
      Events.On('upload:file_progress', (p) => {
        fileProgress = { done: p.data.done, total: p.data.total, student: p.data.student, file: p.data.file }
      }),
      Events.On('upload:status', (e) => {
        statusMsg = e.data.message
      }),
      Events.On('upload:batch_progress', (p) => {
        batchProgress = p.data
      }),
      Events.On('upload:error', (e) => {
        error = e.data.error
      }),
      Events.On('upload:done', () => {
        step = 'done'
      }),
    ]
    return () => unsubs.forEach((off) => off())
  })

  async function browseForFile() {
    const path = await BrowseCSVFile()
    if (path) csvPath = path
  }

</script>

<div class="modal-overlay">
  <div class="modal wizard-modal">
    <h2>Upload Grades — {assignment?.name || 'Assignment'}</h2>

    <!-- Step: File Selection -->
    {#if step === 'file'}
      <p class="step-label">Step 1: Choose CSV file</p>
      <div class="file-picker">
        <input
          type="text"
          readonly
          placeholder="No file selected…"
          bind:value={csvPath}
          class="file-path-input"
        />
        <button class="btn-sm" on:click={browseForFile}>Browse…</button>
      </div>
      <p class="hint">Enter the full path to your grades CSV file.</p>
      <div class="wizard-footer">
        <button class="secondary" on:click={onClose}>Cancel</button>
        <button class="primary" disabled={!csvPath.trim()} on:click={startUpload}>
          Start Upload
        </button>
      </div>

    <!-- Step: Uploading -->
    {:else if step === 'upload'}
      <div class="progress-section">
        {#if error}
          <div class="error-box">
            <p class="error-title">Upload Failed</p>
            <p>{error}</p>
          </div>
          <div class="wizard-footer">
            <button class="secondary" on:click={onClose}>Close</button>
          </div>
        {:else}
          <p class="step-label">{statusMsg || 'Uploading...'}</p>

          <!-- File progress -->
          {#if fileProgress.total > 0}
            <div class="phase-block">
              <div class="phase-header">
                <span class="phase-label">Phase 1 — Uploading files</span>
                <span class="phase-count">{fileProgress.done} / {fileProgress.total}</span>
              </div>
              {#if fileProgress.student}
                <p class="current-student">
                  Uploading files from student <strong>{fileProgress.student}</strong>
                  {#if fileProgress.file}
                    <span class="file-name">({fileProgress.file})</span>
                  {/if}
                </p>
              {/if}
              <div class="progress-bar">
                <div class="fill" style="width: {(fileProgress.done / fileProgress.total) * 100}%"></div>
              </div>
            </div>
          {/if}

          <!-- Batch progress -->
          {#if batchProgress.state !== 'not_started'}
            <div class="phase-block">
              <div class="phase-header">
                <span class="phase-label">Phase 2 — Canvas processing grades</span>
                <span class="phase-count">{batchProgress.completion}%</span>
              </div>
              <div class="progress-bar">
                <div class="fill" style="width: {batchProgress.completion}%"></div>
              </div>
              <p class="current-student">State: <strong>{batchProgress.state}</strong></p>
            </div>
          {/if}

          <div class="wizard-footer">
            <button class="danger" on:click={cancel}>Cancel</button>
          </div>
        {/if}
      </div>

    <!-- Step: Done -->
    {:else if step === 'done'}
      <div class="done-section">
        <p class="done-icon">&#10003;</p>
        <p class="done-title">Upload Complete</p>
        <p>Grades have been submitted to Canvas for {course?.name}.</p>
      </div>
      <div class="wizard-footer">
        <button class="primary" on:click={onClose}>Close</button>
      </div>
    {/if}
  </div>
</div>

<style>
  .wizard-modal {
    max-width: 480px;
  }

  .step-label {
    font-size: 13px;
    color: var(--text-secondary);
    margin-bottom: 12px;
  }

  .hint {
    font-size: 12px;
    color: var(--text-secondary);
    margin-top: 8px;
  }

  .wizard-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 24px;
  }

  .progress-section {
    padding: 16px 0;
  }

  .progress-bar {
    height: 8px;
    background: var(--border-color);
    border-radius: 4px;
    margin: 12px 0 4px;
    overflow: hidden;
  }

  .progress-bar .fill {
    height: 100%;
    background: linear-gradient(90deg, var(--frost-mid), var(--frost-blue));
    border-radius: 4px;
    transition: width 0.3s;
  }

  .error-box {
    background: #fdf0ef;
    border: 1px solid #f5c6cb;
    border-radius: 8px;
    padding: 16px;
    color: var(--danger);
  }

  .error-title {
    font-weight: 600;
    margin-bottom: 4px;
  }

  .done-section {
    text-align: center;
    padding: 32px 0;
  }

  .done-icon {
    font-size: 48px;
    color: var(--success);
    margin-bottom: 12px;
  }

  .done-title {
    font-size: 18px;
    font-weight: 600;
    margin-bottom: 8px;
  }

  /* --- File picker --- */
  .file-picker {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .file-path-input {
    flex: 1;
    font-family: monospace;
    font-size: 12px;
    color: var(--text-secondary);
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 6px 10px;
    cursor: default;
  }

  .btn-sm {
    padding: 6px 14px;
    font-size: 12px;
    font-weight: 500;
    background: var(--frost-blue);
    color: white;
    border: 1px solid transparent;
    border-radius: 6px;
    cursor: pointer;
    white-space: nowrap;
    transition: background 0.15s;
  }

  .btn-sm:hover {
    background: var(--frost-mid);
  }

  /* Uploader progress */
  .phase-block {
    margin-bottom: 16px;
  }

  .phase-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  .phase-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--frost-blue);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .phase-count {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
    font-family: monospace;
  }

  .current-student {
    font-size: 12px;
    color: var(--text-secondary);
    margin: 4px 0 8px;
  }

  .file-name {
    color: var(--text-tertiary);
    font-family: monospace;
  }  
</style>
