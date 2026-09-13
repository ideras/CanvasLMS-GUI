<script>
  import { ExportScoresCSV } from '../../bindings/canvaslms-gui/app.js'
  import { toast } from 'svelte-sonner'

  export let course
  export let stats

  let exporting = false

  async function downloadScores() {
    exporting = true
    try {
      const path = await ExportScoresCSV(course.id)
      if (path) toast.success('Scores saved: ' + path.split('/').pop())
    } catch (e) {
      toast.error(e.message || 'Failed to export scores')
    } finally {
      exporting = false
    }
  }
</script>

<div class="info-grid">
  <div class="info-card">
    <div class="info-value">{stats?.student_count || 0}</div>
    <div class="info-label">Students</div>
  </div>
  <div class="info-card">
    <div class="info-value">{stats?.total_items || 0}</div>
    <div class="info-label">Total Items</div>
  </div>
  <div class="info-card">
    <div class="info-value">{stats?.published_items || 0}</div>
    <div class="info-label">Published</div>
  </div>
  <div class="info-card highlight">
    <div class="info-value">{stats?.needs_grading || 0}</div>
    <div class="info-label">Needs Grading</div>
  </div>
</div>

<div class="info-detail">
  <h3>Course Details</h3>
  <table>
    <tbody>
      <tr><td class="label">Name</td><td>{course?.name || 'N/A'}</td></tr>
      <tr><td class="label">Course Code</td><td>{course?.course_code || 'N/A'}</td></tr>
      <tr><td class="label">Term</td><td>{course?.term?.name || 'N/A'}</td></tr>
      <tr><td class="label">Course ID</td><td class="id-cell">{course?.id || 'N/A'}</td></tr>
    </tbody>
  </table>
</div>

<div class="exports-section">
  <h3>Exports</h3>
  <div class="export-actions">
    <button class="export-btn" on:click={downloadScores} disabled={exporting}>
      {#if exporting}
        <span class="spinner"></span> Exporting…
      {:else}
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/>
          <line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
        Scores CSV
      {/if}
    </button>
  </div>
  <p class="export-hint">One row per student · one column per assignment and quiz</p>
</div>

<!-- InfoTab deliberately NOT migrated to DataTable (handoff §5): a static
     4-row key/value list has no dynamic row model, no headers, no sorting —
     a plain <table> is more honest here. -->

<style>
  /* .id-cell now global in style.css; kept here only as the InfoTab alias
     (same rules) until this component's styles are consolidated. */
  .id-cell {
    font-family: monospace;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 16px;
    margin-bottom: 24px;
  }

  .info-card {
    text-align: center;
    padding: 20px 12px;
    border-radius: 10px;
    background: var(--bg-primary);
  }

  .info-card.highlight {
    background: #fef9e7;
  }

  .info-value {
    font-size: 28px;
    font-weight: 700;
    color: var(--frost-dark);
  }

  .info-card.highlight .info-value {
    color: var(--warning);
  }

  .info-label {
    font-size: 11px;
    color: var(--text-secondary);
    margin-top: 4px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .info-detail {
    margin-top: 16px;
  }

  .info-detail h3 {
    font-size: 14px;
    color: var(--frost-dark);
    margin-bottom: 12px;
  }

  .info-detail .label {
    font-weight: 600;
    width: 120px;
    color: var(--text-secondary);
  }

  /* Exports */
  .exports-section {
    margin-top: 24px;
  }

  .exports-section h3 {
    font-size: 14px;
    color: var(--frost-dark);
    margin-bottom: 10px;
  }

  .export-actions {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }

  .export-btn {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 7px 14px;
    font-size: 12px;
    font-weight: 500;
    background: var(--frost-blue);
    color: white;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    transition: background 0.15s;
  }

  .export-btn:hover:not(:disabled) {
    background: #2e86c1;
  }

  .export-btn:disabled {
    opacity: 0.6;
    cursor: default;
  }

  .spinner {
    width: 10px;
    height: 10px;
    border: 2px solid rgba(255,255,255,0.4);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
    display: inline-block;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .export-hint {
    margin-top: 8px;
    font-size: 11px;
    color: var(--text-secondary);
  }
</style>
