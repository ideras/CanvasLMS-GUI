<script>
  import { GetStudentSubmissions, ExportStudentsCSV } from '../../bindings/canvaslms-gui/app.js'
  import { toast } from 'svelte-sonner'
  import DataTable from './DataTable.svelte'

  export let course
  export let students
  export let itemNameByID

  let selectedStudent = null
  let studentSubmissions = []
  let submissionsLoading = false

  function formatDate(d) {
    if (!d) return 'N/A'
    return d.slice(0, 10)
  }

  async function viewSubmissions(student) {
    selectedStudent = student
    submissionsLoading = true
    studentSubmissions = []
    try {
      studentSubmissions = await GetStudentSubmissions(course.id, student.id)
    } catch (e) {
      studentSubmissions = []
    } finally {
      submissionsLoading = false
    }
  }

  function closeSubmissions() {
    selectedStudent = null
    studentSubmissions = []
  }

  async function downloadStudentCSV() {
    try {
      const path = await ExportStudentsCSV(course.id)
      if (path) {
        toast.success('Student list saved: ' + path.split('/').pop())
      }
    } catch (e) {
      toast.error(e.message || 'Download failed')
    }
  }

  // ---- Roster columns (client-side sorting via DataTable) ----
  const rosterColumns = [
    { accessorKey: 'id', header: 'ID', meta: { cellClass: 'id-cell' } },
    { accessorKey: 'name', header: 'Name' },
    {
      accessorKey: 'email',
      header: 'Email',
      cell: (info) => info.getValue() || 'N/A'
    },
    {
      accessorKey: 'sis_user_id',
      header: 'SIS ID',
      meta: { cellClass: 'id-cell' },
      cell: (info) => info.getValue() || 'N/A'
    },
    {
      id: 'actions',
      header: 'Actions',
      enableSorting: false,
      snippet: true
    }
  ]

  // ---- Submissions drill-down columns ----
  const submissionColumns = [
    {
      id: 'assignment',
      header: 'Assignment',
      accessorFn: (row) => itemNameByID[row.assignment_id] || 'Assignment ' + row.assignment_id,
      snippet: true
    },
    {
      accessorKey: 'score',
      header: 'Score',
      cell: (info) => (info.getValue() != null ? info.getValue() : '-')
    },
    { accessorKey: 'workflow_state', header: 'Status' },
    {
      accessorKey: 'submitted_at',
      header: 'Submitted',
      cell: ({ row }) => formatDate(row.original.submitted_at)
    }
  ]
</script>

{#if selectedStudent}
  <div class="submissions-panel">
    <div class="submissions-header">
      <h3>Submissions — {selectedStudent.name}</h3>
      <button class="link" on:click={closeSubmissions}>Back to roster</button>
    </div>

    {#snippet assignmentCell(sub)}
      <span class="assignment-name">{itemNameByID[sub.assignment_id] || 'Assignment ' + sub.assignment_id}</span>
    {/snippet}

    <DataTable
      data={studentSubmissions}
      columns={submissionColumns}
      cells={{ assignment: assignmentCell }}
      rowKey="assignment_id"
      loading={submissionsLoading}
      emptyMessage="No submissions found."
      dense
    />
  </div>
{:else}
  <div class="roster-actions">
    <button class="secondary small" on:click={downloadStudentCSV}>
      Download CSV
    </button>
  </div>

  {#snippet actionsCell(student)}
    <button class="primary small" on:click={() => viewSubmissions(student)}>
      Submissions
    </button>
  {/snippet}

  <DataTable
    data={students}
    columns={rosterColumns}
    cells={{ actions: actionsCell }}
    rowKey="id"
    emptyMessage="No students found."
    dense
  />
{/if}

<style>
  .link {
    background: none;
    color: var(--frost-blue);
    font-size: 12px;
    padding: 0;
    text-decoration: underline;
  }

  .status {
    text-align: center;
    padding: 32px;
    color: var(--text-secondary);
  }

  button.small {
    padding: 5px 12px;
    font-size: 12px;
  }

  /* Roster */
  .roster-actions {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 12px;
  }

  /* Submissions drill-down */
  .submissions-panel {
    margin-top: 4px;
  }

  .submissions-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .submissions-header h3 {
    font-size: 15px;
    color: var(--frost-dark);
  }

  .assignment-name {
    font-weight: 500;
  }
</style>