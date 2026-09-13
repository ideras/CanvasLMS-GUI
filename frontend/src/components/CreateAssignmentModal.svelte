<script>
  import { CreateAssignment, GetAssignmentGroups, CreateAssignmentGroup } from '../../bindings/canvaslms-gui/app.js'
  import { toast } from 'svelte-sonner'

  export let course
  export let onClose

  let name = ''
  let points = ''
  let dueDate = ''
  let description = ''
  let published = true
  let assignmentGroupID = 0
  let newGroupName = ''
  let creating = false

  // Load existing groups
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

  function parseDueDate(d) {
    if (!d) return null
    // datetime-local returns "2026-05-13T14:30" — missing seconds and timezone.
    // Canvas requires full ISO 8601.
    if (d.includes('T')) return d + ':00Z'
    return d + 'T23:59:59Z'
  }

  async function handleSubmit() {
    if (!name.trim()) return
    creating = true

    try {
      // Create assignment group first if requested
      let groupID = assignmentGroupID
      if (newGroupName.trim()) {
        const group = await CreateAssignmentGroup(course.id, newGroupName.trim())
        groupID = group.id
      }

      // Build payload like Python: only include fields that have values
      const assignment = { name: name.trim(), published }
      if (points) assignment.points_possible = parseFloat(points)
      if (dueDate) assignment.due_at = parseDueDate(dueDate)
      if (description.trim()) assignment.description = description.trim()
      if (groupID > 0) assignment.assignment_group_id = groupID

      await CreateAssignment(course.id, { assignment })
      toast.success('Assignment created')
      onClose()
    } catch (e) {
      toast.error(e.message || e.error || 'Failed to create assignment')
    } finally {
      creating = false
    }
  }
</script>

<div class="modal-overlay" on:click={onClose}>
  <div class="modal" on:click|stopPropagation>
    <h2>Create Assignment</h2>

    <form on:submit|preventDefault={handleSubmit}>
      <div class="field">
        <label for="name">Name *</label>
        <input id="name" type="text" bind:value={name} placeholder="Assignment name" autofocus disabled={creating} />
      </div>

      <div class="field-row">
        <div class="field half">
          <label for="points">Points Possible</label>
          <input id="points" type="number" bind:value={points} placeholder="100" step="0.01" min="0" disabled={creating} />
        </div>
        <div class="field half">
          <label for="due">Due Date</label>
          <input id="due" type="datetime-local" bind:value={dueDate} disabled={creating} />
        </div>
      </div>

      <div class="field">
        <label for="desc">Description</label>
        <textarea id="desc" bind:value={description} rows="3" placeholder="Optional description..." disabled={creating}></textarea>
      </div>

      <div class="field checkbox-field">
        <label>
          <input type="checkbox" bind:checked={published} disabled={creating} />
          Publish immediately
        </label>
      </div>

      <div class="field">
        <label for="group">Assignment Group</label>
        <select id="group" bind:value={assignmentGroupID} disabled={creating}>
          <option value="0">— None —</option>
          {#each groups as g}
            <option value={g.id}>{g.name}</option>
          {/each}
        </select>
      </div>

      <div class="field">
        <label>Or create a new group</label>
        <input type="text" bind:value={newGroupName} placeholder="New group name" disabled={creating || assignmentGroupID > 0} />
      </div>

      <div class="modal-footer">
        <button type="button" class="secondary" on:click={onClose} disabled={creating}>Cancel</button>
        <button type="submit" class="primary" disabled={creating || !name.trim()}>
          {creating ? 'Creating...' : 'Create Assignment'}
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
