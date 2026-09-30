<script>
  import { UploadGrades, CancelUpload, BrowseCSVFile, ResolveUpload, ExportUploadIssues } from '../../bindings/canvaslms-gui/app.js'
  import { Events } from '@wailsio/runtime'
  import { normalizeCSVPath, pasteCSVPath, jobLabel, formatElapsed } from '../lib/upload.js'

  export let course
  export let assignment
  export let onClose
  export let onBackground

  // Wizard state: 'file' | 'validate' | 'upload' | 'done'
  let step = 'file'
  let csvPath = ''
  let fileProgress = { done: 0, total: 0, student: '', file: '' }
  let batchProgress = { state: 'not_started', elapsed_seconds: 0 }
  let batchStartedAt = 0
  let batchFinishedAt = 0
  let clock = Date.now()
  let cancelling = false
  let exportMessage = ''
  $: elapsed = formatElapsed(batchStartedAt ? ((batchFinishedAt || clock) - batchStartedAt) / 1000 : 0)
  let error = null
  let retryable = false
  let fileIssues = []
  let statusMsg = ''
  let unmatched = []
  let summary = { total: 0, skipped: [], failed: [] }


  async function startUpload() {
    csvPath = normalizeCSVPath(csvPath)
    if (!csvPath) return
    step = 'upload'
    error = null
    retryable = false
    fileIssues = []
    unmatched = []
    summary = { total: 0, skipped: [], failed: [] }
    fileProgress = { done: 0, total: 0, student: '', file: '' }
    batchProgress = { state: 'not_started', elapsed_seconds: 0 }
    batchStartedAt = 0
    batchFinishedAt = 0
    cancelling = false
    statusMsg = 'Reading CSV and fetching course roster…'
    try {
      await UploadGrades(course.id, assignment.assignment_id || assignment.id, csvPath)
    } catch (e) {
      error = e.message || String(e)
    }
  }

  async function cancel() {
    if (batchProgress.state !== 'not_started' && !window.confirm('Stop local monitoring? Grades already submitted will continue processing in Canvas.')) return
    cancelling = true
    try { await CancelUpload() } catch (e) { error = e.message || String(e); cancelling = false }
  }

  async function exportUnmatched(students) {
    try {
      const path = await ExportUploadIssues(JSON.stringify(students))
      exportMessage = path ? `Exported to ${path}` : ''
    } catch (e) { exportMessage = e.message || String(e) }
  }

  import { onMount } from 'svelte'

  onMount(() => {
    // Store unsubscribe functions: Events.Off(name) would remove ALL listeners
    // for the event, including the root app's upload:done/upload:error handlers.
    const timer = setInterval(() => { clock = Date.now() }, 1000)
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
        batchStartedAt = Date.now() - (p.data.elapsed_seconds || 0) * 1000
        clock = Date.now()
      }),
      Events.On('upload:cancelled', () => {
        cancelling = false
        error = null
        step = 'file'
        fileProgress = { done: 0, total: 0, student: '', file: '' }
        batchProgress = { state: 'not_started', elapsed_seconds: 0 }
      }),
      Events.On('upload:error', (e) => {
        error = e.data.error
        retryable = e.data.retryable === true
        fileIssues = e.data.files || []
        batchFinishedAt = Date.now()
      }),
      Events.On('upload:unmatched', (e) => {
        unmatched = e.data.students
        step = 'validate'
      }),
      Events.On('upload:summary', (e) => { summary = e.data }),
      Events.On('upload:done', (e) => {
        summary = e.data
        batchFinishedAt = Date.now()
        step = 'done'
      }),
    ]
    return () => { clearInterval(timer); unsubs.forEach((off) => off()) }
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
          on:paste={(event) => csvPath = pasteCSVPath(event, csvPath)}
          aria-label="Grades CSV path"
          placeholder="Type, paste, or browse for a CSV path…"
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

    {:else if step === 'validate'}
      <p>These CSV rows do not match Canvas user IDs in the course. Their files and grades will not be uploaded.</p>
      <ul class="issue-list">
        {#each unmatched as student}
          <li>Row {student.row} — ID {student.student_id}: {student.reason}</li>
        {/each}
      </ul>
      <div class="wizard-footer">
        <button class="secondary" disabled={cancelling} on:click={cancel}>Cancel</button>
        <button class="secondary" on:click={() => exportUnmatched(unmatched)}>Export unmatched list</button>
        <button class="primary" disabled={cancelling} on:click={() => { step = 'upload'; ResolveUpload('ignore') }}>Ignore these students and continue</button>
      </div>

    <!-- Step: Uploading -->
    {:else if step === 'upload'}
      <div class="progress-section">
        {#if error}
          <div class="error-box">
            <p class="error-title">Upload needs attention</p>
            {#if batchStartedAt}<p>{batchProgress.polling_error ? 'Monitoring stopped' : batchProgress.state === 'submitting' ? 'Submission outcome unknown' : jobLabel(batchProgress.state)} — {elapsed} elapsed</p>{/if}
            <p>{fileIssues.length ? 'Fix the files or edit the CSV, then retry the entire workflow.' : error}</p>
            {#if fileIssues.length}
              <ul class="issue-list">
                {#each fileIssues as issue}<li>Student {issue.student_id} — {issue.file}: {issue.reason}</li>{/each}
              </ul>
            {/if}
          </div>
          {#if retryable}
            <p class="hint">Retry re-reads this CSV and the course roster. Confirmed, unchanged uploads from this session are reused.</p>
            <div class="file-picker">
              <input type="text" class="file-path-input" aria-label="Grades CSV path for retry" bind:value={csvPath} on:paste={(event) => csvPath = pasteCSVPath(event, csvPath)} />
              <button class="btn-sm" on:click={browseForFile}>Browse…</button>
            </div>
          {/if}
          <div class="wizard-footer">
            <button class="secondary" on:click={onClose}>Cancel</button>
            {#if retryable}<button class="primary" disabled={!csvPath.trim()} on:click={startUpload}>Retry</button>{/if}
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
              </div>
              <div class="job-status" role="status" aria-live="polite">
                <span class="job-spinner" aria-hidden="true"></span>
                <span><strong>{jobLabel(batchProgress.state)}</strong> — {elapsed} elapsed</span>
              </div>
              {#if batchProgress.polling_error}<p class="hint">{batchProgress.polling_error}</p>{/if}
              <p class="hint">Canvas reports job status, not reliable grading percentages. Closing the app stops monitoring, not a job already sent to Canvas.</p>
            </div>
          {/if}

          <div class="wizard-footer">
            <button class="danger" disabled={cancelling} on:click={cancel}>{cancelling ? 'Stopping…' : batchProgress.state === 'not_started' ? 'Cancel' : 'Stop monitoring'}</button>
            {#if batchProgress.progress_url}
              <button class="secondary" on:click={onBackground}>Continue in background</button>
            {/if}
          </div>
        {/if}
      </div>

    <!-- Step: Done -->
    {:else if step === 'done'}
      <div class="done-section">
        <p class="done-icon">{summary.failed?.length ? '!' : '✓'}</p>
        <p class="done-title">{summary.failed?.length ? 'Canvas finished — review results' : 'Upload Complete'}</p>
        {#if summary.total}
          <p>Canvas finished processing {summary.total} matched students for {course?.name}.</p>
        {:else}<p>No matched students; no files or grades were submitted.</p>{/if}
        {#if batchStartedAt}<p class="hint">{jobLabel(batchProgress.state)} — {elapsed} elapsed</p>{/if}
      </div>
      <div class="wizard-footer">
        <button class="primary" on:click={onClose}>Close</button>
      </div>
    {/if}
    {#if exportMessage}<p class="hint">{exportMessage}</p>{/if}
    {#if summary.skipped?.length || summary.failed?.length || summary.message}
      <div class="upload-summary">
        <p>{summary.skipped?.length || 0} skipped; {summary.failed?.length || 0} Canvas failures/result warnings.</p>
        <ul class="issue-list">
          {#each summary.skipped || [] as student}<li>Skipped ID {student.student_id} (row {student.row}): {student.reason}</li>{/each}
          {#each summary.failed || [] as student}<li>Canvas {student.student_id || 'result'}: {student.reason}</li>{/each}
        </ul>
        {#if summary.message}<p>{summary.message}</p>{/if}
        {#if summary.skipped?.length}<button class="secondary" on:click={() => exportUnmatched(summary.skipped)}>Export skipped list</button>{/if}
      </div>
    {/if}
  </div>
</div>

<style>
  .job-status { display: flex; align-items: center; gap: 10px; font-size: 13px; }
  .job-spinner { width: 18px; height: 18px; border: 3px solid var(--border-color); border-top-color: var(--frost-blue); border-radius: 50%; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  .issue-list { max-height: 220px; overflow: auto; overflow-wrap: anywhere; padding-left: 20px; font-size: 12px; }
  .upload-summary { margin-top: 16px; font-size: 12px; }
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
    flex-wrap: wrap;
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
    cursor: text;
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
