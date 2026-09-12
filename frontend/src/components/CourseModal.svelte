<script>
  import { SaveCourseSet } from '../../wailsjs/go/main/App.js'

  export let courses = []
  export let courseSet = []
  export let onSelect
  export let onClose

  let search = ''
  // Start showing all when no course set is configured yet, so the user
  // can star multiple courses without the list collapsing on first toggle.
  let showAll = courseSet.length === 0

  // Local mutable copy of the set for instant toggle feedback
  let mySet = [...courseSet]

  $: useSet = mySet.length > 0

  $: baseList = (useSet && !showAll)
    ? courses.filter(c => mySet.includes(c.id))
    : courses

  $: filtered = baseList.filter(c => {
    const q = search.toLowerCase()
    return c.name.toLowerCase().includes(q) ||
           (c.course_code && c.course_code.toLowerCase().includes(q))
  })

  function toggleInSet(id) {
    if (mySet.includes(id)) {
      mySet = mySet.filter(i => i !== id)
    } else {
      mySet = [...mySet, id]
    }
    SaveCourseSet(mySet)
  }
</script>

<div class="modal-overlay" on:click={onClose}>
  <div class="modal" on:click|stopPropagation>
    <h2>Select Course</h2>
    <input
      type="text"
      placeholder="Search courses..."
      bind:value={search}
      class="search-input"
      autofocus
    />
    {#if useSet}
      <div class="set-bar">
        <span class="set-info">
          Showing {baseList.length} of {courses.length} courses
        </span>
        <button class="link" on:click={() => showAll = !showAll}>
          {showAll ? 'Show my set' : 'Show all'}
        </button>
      </div>
    {/if}
    <div class="course-list">
      {#if filtered.length === 0}
        <p class="empty">No courses found.</p>
      {:else}
        {#each filtered as course (course.id)}
          <div class="course-row">
            <button class="course-info" on:click={() => onSelect(course)}>
              <span class="course-id">{course.id}</span>
              <span class="course-name">{course.name}</span>
            </button>
            <button class="star-btn" title={mySet.includes(course.id) ? 'Remove from my set' : 'Add to my set'}
                    on:click|stopPropagation={() => toggleInSet(course.id)}>
              {mySet.includes(course.id) ? '★' : '☆'}
            </button>
          </div>
        {/each}
      {/if}
    </div>
    <div class="modal-footer">
      <button class="secondary" on:click={onClose}>Cancel</button>
    </div>
  </div>
</div>

<style>
  .search-input {
    width: 100%;
    margin-bottom: 12px;
  }

  .course-list {
    max-height: 400px;
    overflow-y: auto;
    overflow-x: hidden;
  }

  .empty {
    text-align: center;
    color: var(--text-secondary);
    padding: 24px;
  }

  .set-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
    padding: 6px 10px;
    background: var(--bg-primary);
    border-radius: 6px;
    font-size: 12px;
  }

  .set-info {
    color: var(--text-secondary);
  }

  .link {
    background: none;
    color: var(--frost-blue);
    font-size: 12px;
    padding: 0;
    text-decoration: underline;
  }

  .course-row {
    display: flex;
    align-items: center;
    border-bottom: 1px solid var(--border-color);
  }

  .course-row:hover {
    background: var(--bg-primary);
    border-radius: 6px;
  }

  .course-info {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    padding: 12px 14px;
    background: transparent;
    border: none;
    text-align: left;
    cursor: pointer;
    border-radius: 6px;
    color: inherit;
    font: inherit;
  }

  .course-name {
    font-weight: 600;
  }

  .course-id {
    font-size: 12px;
    color: var(--text-secondary);
  }

  .star-btn {
    padding: 8px 10px 8px 0;
    background: transparent;
    border: none;
    font-size: 16px;
    cursor: pointer;
    color: var(--frost-blue);
    transition: transform 0.15s;
    flex-shrink: 0;
  }

  .star-btn:hover {
    transform: scale(1.2);
  }

  .modal-footer {
    display: flex;
    justify-content: flex-end;
    margin-top: 16px;
  }
</style>
