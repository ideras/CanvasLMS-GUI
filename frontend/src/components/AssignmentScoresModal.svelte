<script>
  import { ListSubmissions } from '../../bindings/canvaslms-gui/app.js'
  import { onMount } from 'svelte'
  import { toasts } from 'svelte-toasts'

  export let course
  export let assignment   // full Assignment object
  export let students     // User[] already loaded in CourseView
  export let onClose

  let submissions = []
  let loading = true

  // Build student lookup: id → name
  $: studentsByID = (students || []).reduce((acc, s) => {
    acc[s.id] = s
    return acc
  }, {})

  onMount(async () => {
    try {
      submissions = await ListSubmissions(course.id, assignment.id)
    } catch (e) {
      toasts.error(e.message || 'Failed to load submissions')
    } finally {
      loading = false
    }
  })

  // ---- computed stats ----
  $: graded   = submissions.filter(s => s.workflow_state === 'graded')
  $: submitted = submissions.filter(s => s.workflow_state !== 'unsubmitted')
  $: missing  = submissions.filter(s => s.missing)
  $: late     = submissions.filter(s => s.late && !s.missing)

  $: scores = graded
    .map(s => typeof s.score === 'number' ? s.score : parseFloat(s.score))
    .filter(n => !isNaN(n))

  $: average = scores.length
    ? (scores.reduce((a, b) => a + b, 0) / scores.length)
    : null

  $: highest = scores.length ? Math.max(...scores) : null
  $: lowest  = scores.length ? Math.min(...scores) : null

  const pts = assignment.points_possible

  function fmtScore(score, grade) {
    if (score == null || score === '') return '—'
    const n = typeof score === 'number' ? score : parseFloat(score)
    if (isNaN(n)) return '—'
    // Trim trailing decimal zeros: 25.50 → "25.5", 25.00 → "25"
    const display = parseFloat(n.toFixed(2)).toString()
    const base = pts ? `${display} / ${pts}` : display
    // Suppress the grade label when it's just the same number in a different
    // string format (e.g. "25.5" vs "25.50"). Compare numerically first,
    // then fall back to string for genuine letter/percentage grades.
    if (grade != null) {
      const gradeNum = parseFloat(grade)
      if (!isNaN(gradeNum) && gradeNum === n) return base
      if (String(grade).trim() === display)   return base
      return `${base} (${grade})`
    }
    return base
  }

  function fmtPct(score) {
    if (score == null || !pts) return ''
    const n = typeof score === 'number' ? score : parseFloat(score)
    if (isNaN(n) || pts === 0) return ''
    return ` (${Math.round((n / pts) * 100)}%)`
  }

  function fmtDate(d) {
    if (!d) return '—'
    return d.slice(0, 10)
  }

  function statusLabel(sub) {
    if (sub.excused)                      return { text: 'Excused',     cls: 'excused'   }
    if (sub.missing)                      return { text: 'Missing',     cls: 'missing'   }
    if (sub.workflow_state === 'graded')  return { text: 'Graded',      cls: 'graded'    }
    if (sub.workflow_state === 'submitted' || sub.submitted_at)
                                          return { text: 'Submitted',   cls: 'submitted' }
    return                                       { text: 'Not submitted', cls: 'none'    }
  }

  // Sort: graded first (by score desc), then submitted, then not submitted
  $: sortedRows = (() => {
    const withName = submissions.map(s => ({
      ...s,
      _name: studentsByID[s.user_id]?.name ?? `User ${s.user_id}`,
    }))
    return [...withName].sort((a, b) => {
      const aScore = typeof a.score === 'number' ? a.score : parseFloat(a.score)
      const bScore = typeof b.score === 'number' ? b.score : parseFloat(b.score)
      if (!isNaN(aScore) && !isNaN(bScore)) return bScore - aScore
      if (!isNaN(aScore)) return -1
      if (!isNaN(bScore)) return  1
      return a._name.localeCompare(b._name)
    })
  })()
</script>

<div class="modal-overlay" on:click={onClose}>
  <div class="modal" on:click|stopPropagation>

    <!-- Header -->
    <div class="modal-header">
      <div>
        <h2>Scores</h2>
        <p class="subtitle">{assignment.name}</p>
      </div>
      <button class="close-btn" on:click={onClose} title="Close">✕</button>
    </div>

    {#if loading}
      <p class="status">Loading submissions…</p>

    {:else}
      <!-- Stats bar -->
      <div class="stats-bar">
        <div class="stat">
          <span class="stat-val">{submissions.length}</span>
          <span class="stat-label">Total</span>
        </div>
        <div class="stat">
          <span class="stat-val">{graded.length}</span>
          <span class="stat-label">Graded</span>
        </div>
        <div class="stat">
          <span class="stat-val">{submitted.length}</span>
          <span class="stat-label">Submitted</span>
        </div>
        {#if average != null}
          <div class="stat highlight">
            <span class="stat-val">{average.toFixed(1)}{fmtPct(average)}</span>
            <span class="stat-label">Average</span>
          </div>
          <div class="stat">
            <span class="stat-val">{highest}{fmtPct(highest)}</span>
            <span class="stat-label">Highest</span>
          </div>
          <div class="stat">
            <span class="stat-val">{lowest}{fmtPct(lowest)}</span>
            <span class="stat-label">Lowest</span>
          </div>
        {/if}
        {#if missing.length > 0}
          <div class="stat warn">
            <span class="stat-val">{missing.length}</span>
            <span class="stat-label">Missing</span>
          </div>
        {/if}
        {#if late.length > 0}
          <div class="stat warn">
            <span class="stat-val">{late.length}</span>
            <span class="stat-label">Late</span>
          </div>
        {/if}
      </div>

      <!-- Scores table -->
      {#if sortedRows.length === 0}
        <p class="status">No submissions found.</p>
      {:else}
        <div class="table-wrap">
          <table>
            <thead>
              <tr>
                <th>Student</th>
                <th class="num-col">Score</th>
                <th>Status</th>
                <th>Submitted</th>
              </tr>
            </thead>
            <tbody>
              {#each sortedRows as sub (sub.user_id)}
                {@const st = statusLabel(sub)}
                <tr class:late-row={sub.late && !sub.missing}>
                  <td class="name-cell">{sub._name}</td>
                  <td class="num-col">{fmtScore(sub.score, sub.grade)}</td>
                  <td>
                    <span class="badge badge-{st.cls}">{st.text}</span>
                    {#if sub.late && !sub.missing && !sub.excused}
                      <span class="badge badge-late">Late</span>
                    {/if}
                  </td>
                  <td class="date-cell">{fmtDate(sub.submitted_at)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    {/if}

    <div class="modal-footer">
      <button class="primary" on:click={onClose}>Close</button>
    </div>
  </div>
</div>

<style>
  .modal {
    max-width: 680px;
    width: 95vw;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 18px;
  }

  .modal-header h2 {
    margin: 0 0 2px;
  }

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
    line-height: 1;
  }

  .close-btn:hover {
    color: var(--text-primary);
  }

  /* Stats bar */
  .stats-bar {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 18px;
  }

  .stat {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 8px 14px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    min-width: 64px;
  }

  .stat.highlight {
    background: #eaf2f8;
    border-color: var(--frost-blue);
  }

  .stat.warn {
    background: #fef9e7;
    border-color: #f0c040;
  }

  .stat-val {
    font-size: 16px;
    font-weight: 700;
    color: var(--frost-dark);
  }

  .stat-label {
    font-size: 10px;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    margin-top: 2px;
  }

  /* Table */
  .table-wrap {
    max-height: 380px;
    overflow-y: auto;
    border: 1px solid var(--border-color);
    border-radius: 8px;
  }

  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  thead th {
    position: sticky;
    top: 0;
    background: var(--bg-primary);
    padding: 8px 12px;
    text-align: left;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
    border-bottom: 1px solid var(--border-color);
  }

  tbody tr {
    border-bottom: 1px solid var(--border-color);
    transition: background 0.1s;
  }

  tbody tr:last-child {
    border-bottom: none;
  }

  tbody tr:hover {
    background: var(--bg-primary);
  }

  tbody tr.late-row {
    background: #fffbf0;
  }

  td {
    padding: 8px 12px;
    vertical-align: middle;
  }

  .name-cell {
    font-weight: 500;
  }

  .num-col {
    text-align: right;
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }

  .date-cell {
    color: var(--text-secondary);
    font-size: 12px;
    white-space: nowrap;
  }

  /* Badges */
  .badge {
    display: inline-block;
    font-size: 10px;
    font-weight: 600;
    padding: 2px 7px;
    border-radius: 8px;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .badge + .badge {
    margin-left: 4px;
  }

  .badge-graded    { background: #d4efdf; color: #1e8449; }
  .badge-submitted { background: #d6eaf8; color: #1a5276; }
  .badge-missing   { background: #fde8e8; color: #a93226; }
  .badge-excused   { background: #f0e6ff; color: #6c3483; }
  .badge-late      { background: #fef0cd; color: #9a6700; }
  .badge-none      { background: var(--border-color); color: var(--text-secondary); }

  .status {
    text-align: center;
    padding: 32px;
    color: var(--text-secondary);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
  }
</style>
