<script>
  import { ExportQuizQuestions, ExportQuizSubmissions } from '../../bindings/canvaslms-gui/app.js'
  import { Events } from '@wailsio/runtime'
  import { toast } from 'svelte-sonner'
  import { onMount } from 'svelte'
  import { fly } from 'svelte/transition'
  import DataTable from './DataTable.svelte'
  import { renderSnippet } from '@tanstack/svelte-table'
  import iconUpload from '../assets/images/icon-upload.svg'
  import iconQuestions from '../assets/images/icon-questions.svg'
  import iconSubmissions from '../assets/images/icon-submissions.svg'

  export let items
  export let course
  export let onUpload
  export let onCreate = null
  export let onEdit = null
  export let onViewScores = null
  export let onDownloadSubmissions = null

  let typeFilter = 'all'
  let openDropdownId = null
  let submissionsProgress = null

  $: filteredItems = typeFilter === 'all'
    ? items
    : items.filter(i => i.type === typeFilter)

  onMount(() => {
    const unsubs = [
      Events.On('submissions:start', () => {
        submissionsProgress = { current: 0, total: 1, loading: true }
      }),
      Events.On('submissions:progress', (p) => {
        submissionsProgress = p.data
      }),
      Events.On('submissions:done', (d) => {
        submissionsProgress = { current: d.data.total, total: d.data.total, done: true }
        setTimeout(() => {
          submissionsProgress = null
          toast.success(d.data.message)
        }, 1200)
      }),
      Events.On('submissions:error', (e) => {
        submissionsProgress = null
        toast.error(e.data.error || 'Download failed')
      }),
    ]
    return () => unsubs.forEach((off) => off())
  })

  function formatDate(d) {
    if (!d) return 'N/A'
    return d.slice(0, 10)
  }

  function formatPoints(p) {
    if (p == null) return '-'
    return parseFloat(Number(p).toFixed(2)).toString()
  }

  function toggleDropdown(id) {
    openDropdownId = openDropdownId === id ? null : id
  }

  function closeDropdown() {
    openDropdownId = null
  }

  async function downloadQuizQuestions(item) {
    try {
      const path = await ExportQuizQuestions(course.id, item.id, item.name, 'md')
      if (path) {
        toast.success('Questions saved: ' + path.split('/').pop())
      }
    } catch (e) {
      toast.error(e.message || 'Download failed')
    }
  }

  function submitWithFormat(item, format) {
    closeDropdown()
    ExportQuizSubmissions(course.id, item.id, format)
  }

  // ---- Items table columns (DataTable / TanStack) ----
  function numericSort(rowA, rowB, columnId) {
    const toNum = (v) => (typeof v === 'number' ? v : parseFloat(v))
    const a = toNum(rowA.getValue(columnId))
    const b = toNum(rowB.getValue(columnId))
    if (!isNaN(a) && !isNaN(b)) return a - b
    if (!isNaN(a)) return -1
    if (!isNaN(b)) return 1
    return 0
  }

  const itemColumns = [
    { accessorKey: 'id', header: 'ID', meta: { cellClass: 'id-cell' } },
    { accessorKey: 'name', header: 'Name', meta: { cellClass: 'name-cell' } },
    {
      accessorKey: 'type',
      header: 'Type',
      cell: ({ row }) => renderSnippet(typeCell, row.original)
    },
    {
      accessorKey: 'due_at',
      header: 'Due Date',
      cell: ({ row }) => formatDate(row.original.due_at)
    },
    {
      accessorKey: 'points',
      header: 'Points',
      sortFn: numericSortFn,
      cell: ({ row }) => formatPoints(row.original.points)
    },
    {
      id: 'actions',
      header: 'Actions',
      enableSorting: false,
      cell: ({ row }) => renderSnippet(actionsCell, row.original)
    }
  ]
</script>

<div class="filter-bar">
  <button class="filter-btn" class:active={typeFilter === 'all'} on:click={() => typeFilter = 'all'}>All</button>
  <button class="filter-btn" class:active={typeFilter === 'assignment'} on:click={() => typeFilter = 'assignment'}>Assignments</button>
  <button class="filter-btn" class:active={typeFilter === 'quiz'} on:click={() => typeFilter = 'quiz'}>Quizzes</button>

  {#if onCreate}
    <button class="create-btn" on:click={onCreate}>+ Assignment</button>
  {/if}
</div>

{#if submissionsProgress}
  <div class="progress-bar-wrapper" in:fly|global={{ y: -20, duration: 200 }}>
    <div class="progress-track">
      {#if submissionsProgress.loading}
        <div class="progress-fill indeterminate"></div>
      {:else}
        <div
          class="progress-fill"
          style="width: {submissionsProgress.current / submissionsProgress.total * 100}%"
        ></div>
      {/if}
    </div>
    <span class="progress-label">
      {#if submissionsProgress.done}
        Done
      {:else if submissionsProgress.loading}
        Loading...
      {:else}
        {submissionsProgress.current} / {submissionsProgress.total}
      {/if}
    </span>
  </div>
{/if}

  {#snippet typeCell(item)}
    <span class="type-badge" class:quiz={item.type === 'quiz'}>
      {item.type === 'quiz' ? (item.is_old_quiz ? 'old-quiz' : 'new-quiz') : 'assignment'}
    </span>
  {/snippet}

  {#snippet actionsCell(item)}
    <button class="icon-btn" title="Upload Grades" on:click={() => onUpload(item)}>
      <img src={iconUpload} alt="Upload Grades" width="20" height="20" />
    </button>
    {#if item.type === 'assignment' && onEdit}
      <button class="icon-btn" title="Edit Assignment" on:click={() => onEdit(item)}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
          <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
        </svg>
      </button>
    {/if}
    {#if item.type === 'assignment' && onViewScores}
      <button class="icon-btn" title="View Scores" on:click={() => onViewScores(item)}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="20" x2="18" y2="10"/>
          <line x1="12" y1="20" x2="12" y2="4"/>
          <line x1="6"  y1="20" x2="6"  y2="14"/>
        </svg>
      </button>
    {/if}
    {#if item.type === 'assignment' && onDownloadSubmissions}
      <button class="icon-btn" title="Download Submissions" on:click={() => onDownloadSubmissions(item)}>
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
          <polyline points="7 10 12 15 17 10"/>
          <line x1="12" y1="15" x2="12" y2="3"/>
        </svg>
      </button>
    {/if}
    {#if item.type === 'quiz' && item.is_old_quiz}
      <button class="icon-btn" title="Download Questions" on:click={() => downloadQuizQuestions(item)}>
        <img src={iconQuestions} alt="Download Questions" width="20" height="20" />
      </button>
      <div class="submissions-wrapper">
        <button
          class="icon-btn"
          title="Download Submissions"
          on:click={() => toggleDropdown(item.id)}
          on:focusout={closeDropdown}
        >
          <img src={iconSubmissions} alt="Download Submissions" width="20" height="20" />
        </button>
        <div class="dropdown-menu" class:visible={openDropdownId === item.id}>
          <button class="dropdown-item" on:click|stopPropagation={() => submitWithFormat(item, 'md')}>
            Markdown (.md)
          </button>
          <button class="dropdown-item" on:click|stopPropagation={() => submitWithFormat(item, 'html')}>
            HTML (.html)
          </button>
          <button class="dropdown-item" on:click|stopPropagation={() => submitWithFormat(item, 'json')}>
            JSON (.json)
          </button>
        </div>
      </div>
    {/if}
  {/snippet}

  <DataTable
    data={filteredItems}
    columns={itemColumns}
    rowKey={(item) => item.id + '-' + item.type}
    emptyMessage="No items found."
  />

<style>
  .filter-bar {
    display: flex;
    gap: 6px;
    margin-bottom: 16px;
  }

  .filter-btn {
    padding: 4px 12px;
    font-size: 12px;
    background: var(--bg-primary);
    color: var(--text-secondary);
    border-radius: 12px;
  }

  .filter-btn.active {
    background: var(--frost-blue);
    color: white;
  }

  .create-btn {
    margin-left: auto;
    padding: 4px 14px;
    font-size: 12px;
    background: var(--frost-blue);
    color: white;
    border-radius: 6px;
    font-weight: 500;
  }

  .create-btn:hover {
    background: #2e86c1;
  }

  .type-badge {
    font-size: 11px;
    padding: 2px 8px;
    border-radius: 8px;
    background: #eaf2f8;
    color: var(--frost-blue);
    font-weight: 500;
  }

  .type-badge.quiz {
    background: #fef9e7;
    color: #b7950b;
  }

  .icon-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    padding: 4px;
    background: transparent;
    border: 1px solid var(--border-color);
    border-radius: 6px;
    cursor: pointer;
    color: var(--text-secondary);
    transition: background 0.15s, border-color 0.15s;
    line-height: 0;
    vertical-align: middle;
  }

  .icon-btn:hover {
    background: var(--border-color);
    border-color: var(--frost-mid);
  }

  .icon-btn:active {
    transform: scale(0.93);
  }

  .icon-btn + .icon-btn,
  .icon-btn + .submissions-wrapper {
    margin-left: 4px;
  }

  .submissions-wrapper {
    display: inline-block;
    position: relative;
    vertical-align: middle;
  }

  .dropdown-menu {
    position: absolute;
    top: 100%;
    right: 0;
    margin-top: 4px;
    min-width: 140px;
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
    z-index: 50;
    visibility: hidden;
    opacity: 0;
    transition: visibility 0.15s, opacity 0.15s;
    overflow: hidden;
  }

  .dropdown-menu.visible {
    visibility: visible;
    opacity: 1;
  }

  .dropdown-item {
    display: block;
    width: 100%;
    padding: 7px 14px;
    font-size: 12px;
    text-align: left;
    background: transparent;
    border: none;
    border-radius: 0;
    color: var(--text-primary);
    cursor: pointer;
    transition: background 0.1s;
  }

  .dropdown-item:hover {
    background: var(--bg-primary);
  }

  /* Progress bar */
  .progress-bar-wrapper {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
    padding: 8px 12px;
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 8px;
  }

  .progress-track {
    flex: 1;
    height: 6px;
    background: var(--border-color);
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, var(--frost-mid), var(--frost-blue));
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .progress-fill.indeterminate {
    width: 40%;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { margin-left: 0; width: 30%; }
    50% { margin-left: 55%; width: 30%; }
  }

  .progress-label {
    font-size: 12px;
    color: var(--text-secondary);
    white-space: nowrap;
    font-weight: 500;
  }
</style>
