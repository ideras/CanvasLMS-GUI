<script>
  import { EditAssignment, GetAssignmentGroups, CreateAssignmentGroup } from '../../bindings/canvaslms-gui/app.js'
  import { toast } from 'svelte-sonner'

  export let course
  export let assignment   // the existing assignment to edit
  export let onClose

  // Helper: convert a Canvas ISO date string ("2026-05-13T14:30:00Z") to the
  // value format expected by <input type="datetime-local"> ("2026-05-13T14:30").
  function toLocalInput(iso) {
    if (!iso) return ''
    // Strip seconds + timezone suffix so the browser input is happy.
    return iso.slice(0, 16)
  }

  function parseDueDate(d) {
    if (!d) return null
    if (d.includes('T')) return d + ':00Z'
    return d + 'T23:59:59Z'
  }

  // Initialise form state from the existing assignment.
  let name        = assignment.name        ?? ''
  let points      = assignment.points_possible != null ? String(assignment.points_possible) : ''
  let dueDate     = toLocalInput(assignment.due_at)
  let description = assignment.description ?? ''
  let published   = assignment.published   ?? true
  let assignmentGroupID = assignment.assignment_group_id ?? 0
  let newGroupName = ''
  let saving = false

  // Load existing groups for the group selector.
  let groups = []
  let groupsLoaded = false

  async function loadGroups() {
    try {
      groups = await GetAssignmentGroups(course.id)
      groupsLoaded = true
    } catch (e) {
      groups = []
      groupsLoaded = true
    }
  }
  loadGroups()

  async function handleSubmit() {
    if (!name.trim()) return
    saving = true

    try {
      // Create a new assignment group first if requested.
      let groupID = assignmentGroupID
      if (newGroupName.trim()) {
        const group = await CreateAssignmentGroup(course.id, newGroupName.trim())
        groupID = group.id
      }

      // Build the Canvas update payload — only include fields the user can change.
      const payload = { name: name.trim(), published }
      if (points !== '')       payload.points_possible     = parseFloat(points)
      if (dueDate)             payload.due_at              = parseDueDate(dueDate)
      else                     payload.due_at              = null   // clear the due date
      if (description.trim())  payload.description         = description.trim()
      else                     payload.description         = ''
      if (groupID > 0)         payload.assignment_group_id = groupID

      await EditAssignment(course.id, assignment.id, { assignment: payload })
      toast.success('Assignment updated')
      onClose()
    } catch (e) {
      toast.error(e.message || e.error || 'Failed to update assignment')
    } finally {
      saving = false
    }
  }
</script>

<div class="modal-overlay" on:click={onClose}>
  <div class="modal" on:click|stopPropagation>
    <h2>Edit Assignment</h2>

    <form on:submit|preventDefault={handleSubmit}>
      <div class="field">
        <label for="name">Name *</label>
        <input
          id="name"
          type="text"
          bind:value={name}
          placeholder="Assignment name"
          autofocus
          disabled={saving}
        />
      </div>

      <div class="field-row">
        <div class="field half">
          <label for="points">Points Possible</label>
          <input
            id="points"
            type="number"
            bind:value={points}
            placeholder="100"
            step="0.01"
            min="0"
            disabled={saving}
          />
        </div>
        <div class="field half">
          <label for="due">Due Date</label>
          <input id="due" type="datetime-local" bind:value={dueDate} disabled={saving} />
        </div>
      </div>

      <div class="field">
        <label for="desc">Description</label>
        <textarea
          id="desc"
          bind:value={description}
          rows="3"
          placeholder="Optional description..."
          disabled={saving}
        ></textarea>
      </div>

      <div class="field checkbox-field">
        <label>
          <input type="checkbox" bind:checked={published} disabled={saving} />
          Published
        </label>
      </div>

      <div class="field">
        <label for="group">Assignment Group</label>
        <select id="group" bind:value={assignmentGroupID} disabled={saving}>
          <option value="0">— None —</option>
          {#each groups as g}
            <option value={g.id}>{g.name}</option>
          {/each}
        </select>
      </div>

      <div class="field">
        <label>Or create a new group</label>
        <input
          type="text"
          bind:value={newGroupName}
          placeholder="New group name"
          disabled={saving || assignmentGroupID > 0}
        />
      </div>

      <div class="modal-footer">
        <button type="button" class="secondary" on:click={onClose} disabled={saving}>Cancel</button>
        <button type="submit" class="primary" disabled={saving || !name.trim()}>
          {saving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </form>
  </div>
</div>

<style>
  .modal {
    max-width: 520px;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .field-row {
    display: flex;
    gap: 14px;
  }

  .field.half {
    flex: 1;
  }

  .checkbox-field label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 400;
    cursor: pointer;
  }

  textarea {
    border: 1px solid var(--border-color);
    border-radius: 6px;
    padding: 8px 12px;
    font-size: 13px;
    font-family: inherit;
    resize: vertical;
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    margin-top: 8px;
  }
</style>
