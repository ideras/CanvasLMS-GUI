<script>
  import { GetStudentSubmissions, ExportStudentsCSV } from '../../bindings/canvaslms-gui/app.js'
  import { toast } from 'svelte-sonner'

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
</script>

{#if selectedStudent}
  <div class="submissions-panel">
    <div class="submissions-header">
      <h3>Submissions — {selectedStudent.name}</h3>
      <button class="link" on:click={closeSubmissions}>Back to roster</button>
    </div>
    {#if submissionsLoading}
      <p class="status">Loading submissions...</p>
    {:else if studentSubmissions.length === 0}
      <p class="status">No submissions found.</p>
    {:else}
      <table>
        <thead>
          <tr>
            <th>Assignment</th>
            <th>Score</th>
            <th>Status</th>
            <th>Submitted</th>
          </tr>
        </thead>
        <tbody>
          {#each studentSubmissions as sub}
            <tr>
              <td>
                <span class="assignment-name">
                  {itemNameByID[sub.assignment_id] || 'Assignment ' + sub.assignment_id}
                </span>
              </td>
              <td>{sub.score != null ? sub.score : '-'}</td>
              <td>{sub.workflow_state}</td>
              <td>{formatDate(sub.submitted_at)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
{:else}
  <div class="roster-actions">
    <button class="secondary small" on:click={downloadStudentCSV}>
      Download CSV
    </button>
  </div>

  {#if students.length === 0}
    <p class="status">No students found.</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>ID</th>
          <th>Name</th>
          <th>Email</th>
          <th>SIS ID</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        {#each students as s}
          <tr>
            <td class="id-cell">{s.id}</td>
            <td>{s.name}</td>
            <td>{s.email || 'N/A'}</td>
            <td class="id-cell">{s.sis_user_id || 'N/A'}</td>
            <td class="actions-cell">
              <button class="primary small" on:click={() => viewSubmissions(s)}>
                Submissions
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  {/if}
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

  .id-cell {
    font-family: monospace;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .actions-cell {
    text-align: right;
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
