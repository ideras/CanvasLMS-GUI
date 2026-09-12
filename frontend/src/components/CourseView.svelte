<script>
  import StudentsTab from './StudentsTab.svelte'
  import ItemsTab from './ItemsTab.svelte'
  import InfoTab from './InfoTab.svelte'
  import CreateAssignmentModal from './CreateAssignmentModal.svelte'
  import EditAssignmentModal from './EditAssignmentModal.svelte'
  import AssignmentScoresModal from './AssignmentScoresModal.svelte'
  import DownloadSubmissionsModal from './DownloadSubmissionsModal.svelte'
  import AnnouncementsTab from './AnnouncementsTab.svelte'
  import { ListCourseItems, ListStudents, GetCourseStats, InvalidateCourse, ListAssignments } from '../../wailsjs/go/main/App.js'

  export let course
  export let onUpload
  export let onSwitchCourse

  let activeTab = 'students'
  let items = []
  let students = []
  let stats = null
  let loading = false
  let error = null
  let showCreateModal = false
  let editingAssignment = null   // set to an assignment object to show the edit modal
  let scoringAssignment = null   // set to an assignment object to show the scores modal
  let downloadingAssignment = null  // set to an assignment object to show the download submissions modal

  // Full Assignment objects keyed by ID — used to populate the edit modal.
  let assignmentsByID = {}

  // Lookup: assignment ID → name (for submissions display)
  $: itemNameByID = items.reduce((acc, i) => {
    const id = i.type === 'quiz' ? (i.assignment_id || i.id) : i.id
    acc[id] = i.name
    return acc
  }, {})

  $: if (course) loadAll()

  async function loadAll() {
    loading = true
    error = null
    try {
      const [i, s, st, asgn] = await Promise.all([
        ListCourseItems(course.id),
        ListStudents(course.id),
        GetCourseStats(course.id),
        ListAssignments(course.id)
      ])
      items = i || []
      students = s || []
      stats = st
      assignmentsByID = (asgn || []).reduce((acc, a) => { acc[a.id] = a; return acc }, {})
    } catch (e) {
      error = e.message || 'Failed to load course data'
    } finally {
      loading = false
    }
  }

  async function refreshCourseData() {
    await InvalidateCourse(course.id)
    await loadAll()
  }
</script>

<div class="course-panel">
  <!-- Header -->
  <div class="course-header">
    <div>
      <h2>{course?.name || 'Course'}</h2>
      <div class="course-actions">
        <button class="btn-sm" on:click={onSwitchCourse}>Switch course</button>
        <button class="btn-sm secondary" on:click={refreshCourseData}>↺ Refresh</button>
      </div>
    </div>
    {#if stats}
      <div class="quick-stats">
        <span title="Students">{stats.student_count} students</span>
        <span title="Items">{stats.total_items} items</span>
        <span title="Published">{stats.published_items} published</span>
        {#if stats.needs_grading > 0}
          <span class="needs-grading" title="Needs grading">{stats.needs_grading} to grade</span>
        {/if}
      </div>
    {/if}
  </div>

  <!-- Tabs -->
  <div class="tabs">
    <button class="tab" class:active={activeTab === 'students'} on:click={() => activeTab = 'students'}>
      Students
    </button>
    <button class="tab" class:active={activeTab === 'items'} on:click={() => activeTab = 'items'}>
      Assignments &amp; Quizzes
    </button>
    <button class="tab" class:active={activeTab === 'info'} on:click={() => activeTab = 'info'}>
      Info
    </button>
    <button class="tab" class:active={activeTab === 'announcements'} on:click={() => activeTab = 'announcements'}>
      Anuncios
    </button>
  </div>

  <!-- Tab content -->
  <div class="tab-content">
    {#if loading}
      <p class="status">Loading...</p>

    {:else if error}
      <p class="status error">{error}</p>

    {:else if activeTab === 'students'}
      <StudentsTab {course} {students} {itemNameByID} />

    {:else if activeTab === 'items'}
      <ItemsTab
        {items} {course} {onUpload}
        onCreate={() => showCreateModal = true}
        onEdit={(item) => editingAssignment = assignmentsByID[item.id] ?? item}
        onViewScores={(item) => scoringAssignment = assignmentsByID[item.id] ?? item}
        onDownloadSubmissions={(item) => {
          const a = assignmentsByID[item.id]
          if (a?.submission_types?.includes('online_upload')) downloadingAssignment = a
        }}
      />

    {:else if activeTab === 'info'}
      <InfoTab {course} {stats} />

    {:else if activeTab === 'announcements'}
      <AnnouncementsTab {course} />
    {/if}
  </div>
</div>

{#if showCreateModal}
  <CreateAssignmentModal
    {course}
    onClose={() => { showCreateModal = false; loadAll() }}
  />
{/if}

{#if editingAssignment}
  <EditAssignmentModal
    {course}
    assignment={editingAssignment}
    onClose={() => { editingAssignment = null; loadAll() }}
  />
{/if}

{#if scoringAssignment}
  <AssignmentScoresModal
    {course}
    assignment={scoringAssignment}
    {students}
    onClose={() => scoringAssignment = null}
  />
{/if}

{#if downloadingAssignment}
  <DownloadSubmissionsModal
    {course}
    assignment={downloadingAssignment}
    onClose={() => downloadingAssignment = null}
  />
{/if}

<style>
  .course-panel {
    background: var(--bg-card);
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0,0,0,0.06);
  }

  .course-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 24px 24px 0;
  }

  .course-header h2 {
    font-size: 18px;
    color: var(--frost-dark);
    margin-bottom: 4px;
  }

  .link {
    background: none;
    color: var(--frost-blue);
    font-size: 12px;
    padding: 0;
    text-decoration: underline;
  }

  .quick-stats {
    display: flex;
    gap: 16px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .quick-stats .needs-grading {
    color: var(--warning);
    font-weight: 600;
  }

  .tabs {
    display: flex;
    border-bottom: 2px solid var(--border-color);
    margin: 16px 24px 0;
  }

  .tab {
    padding: 10px 20px;
    background: none;
    color: var(--text-secondary);
    border-radius: 0;
    font-size: 13px;
    font-weight: 500;
    border-bottom: 2px solid transparent;
    margin-bottom: -2px;
    transition: color 0.15s, border-color 0.15s;
  }

  .tab:hover {
    color: var(--text-primary);
  }

  .tab.active {
    color: var(--frost-blue);
    border-bottom-color: var(--frost-blue);
  }

  .tab-content {
    padding: 20px 24px 24px;
  }

  .status {
    text-align: center;
    padding: 32px;
    color: var(--text-secondary);
  }

  .status.error {
    color: var(--danger);
  }

  .course-actions {
    display: flex;
    gap: 8px;
    margin-top: 6px;
  }

  .btn-sm {
    padding: 4px 12px;
    font-size: 12px;
    font-weight: 500;
    background: var(--frost-blue);
    color: white;
    border: 1px solid transparent;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s, border-color 0.15s;
  }

  .btn-sm:hover {
    background: var(--frost-mid);
  }

  .btn-sm:active {
    transform: scale(0.96);
  }

  .btn-sm.secondary {
    background: var(--bg-primary);
    color: var(--frost-blue);
    border-color: var(--frost-blue);
  }

  .btn-sm.secondary:hover {
    background: var(--border-color);
  }
</style>
