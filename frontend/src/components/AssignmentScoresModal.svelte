<script>
  import { ListSubmissions } from '../../bindings/canvaslms-gui/app.js'
  import { onMount } from 'svelte'
  import { toast } from 'svelte-sonner'

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
      toast.error(e.message || 'Failed to load submissions')
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
    // TanStack's auto-sort-direction probe calls accessorFn with a synthetic
    // row whose `original` is undefined — must not throw (a throw here would
    // abort the delegated event walk and fall through to the overlay's
    // onClose, closing the dialog).
    if (!sub) return { text: 'Not submitted', cls: 'none' }
    if (sub.excused)                      return { text: 'Excused',     cls: 'excused'   }
    if (sub.missing)                      return { text: 'Missing',     cls: 'missing'   }
    if (sub.workflow_state === 'graded')  return { text: 'Graded',      cls: 'graded'    }
    if (sub.workflow_state === 'submitted' || sub.submitted_at)
                                          return { text: 'Submitted',   cls: 'submitted' }
    return                                       { text: 'Not submitted', cls: 'none'    }
  }

  // ---- Table (DataTable / TanStack) ----
  import DataTable from './DataTable.svelte'

  // Mirrors the previous hand-rolled comparator: numeric score,
  // unscored always last (null maps to -Infinity so the direction
  // multiplier of asc/desc toggling never floats them to the top).
  function scoreSortFn(rowA, rowB, columnId) {
    const toNum = (v) => {
      const n = typeof v === 'number' ? v : parseFloat(v)
      return isNaN(n) ? -Infinity : n
    }
    return toNum(rowA.getValue(columnId)) - toNum(rowB.getValue(columnId))
  }

  const STATUS_ORDER = ['Graded', 'Submitted', 'Excused', 'Missing', 'Not submitted']
  function statusSortFn(rowA, rowB) {
    return (
      STATUS_ORDER.indexOf(statusLabel(rowA.original).text) -
      STATUS_ORDER.indexOf(statusLabel(rowB.original).text)
    )
  }

  const scoreColumns = [
    {
      id: 'student',
      header: 'Student',
      accessorFn: (row) => studentsByID[row.user_id]?.name ?? `User ${row.user_id}`,
      meta: { cellClass: 'name-cell' }
    },
    {
      accessorKey: 'score',
      header: 'Score',
      meta: { cellClass: 'num-col' },
      sortFn: scoreSortFn,
      cell: ({ row }) => fmtScore(row.original.score, row.original.grade)
    },
    {
      id: 'status',
      header: 'Status',
      accessorFn: (row) => statusLabel(row.original).text,
      sortFn: statusSortFn,
      snippet: true
    },
    {
      accessorKey: 'submitted_at',
      header: 'Submitted',
      meta: { cellClass: 'date-cell' },
      cell: ({ row }) => fmtDate(row.original.submitted_at)
    }
  ]
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
      {#snippet statusCell(sub)}
        {@const st = statusLabel(sub)}
        <span class="badge badge-{st.cls}">{st.text}</span>
        {#if sub.late && !sub.missing && !sub.excused}
          <span class="badge badge-late">Late</span>
        {/if}
      {/snippet}

      <div class="table-wrap">
        <DataTable
          data={submissions}
          columns={scoreColumns}
          cells={{ status: statusCell }}
          rowKey="user_id"
          tableClass="compact-table"
          rowClass={(s) => (s.late && !s.missing ? 'late-row' : '')}
          initialSorting={[
            { id: 'score', desc: true },
            { id: 'student', desc: false }
          ]}
          emptyMessage="No submissions found."
        />
      </div>
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

  /* Table cell classes live in global CSS (style.css) — cells render
     inside DataTable.svelte so component-scoped styles can't reach them.
     .name-cell / .num-col / .date-cell / .compact-table rules moved there. */

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
