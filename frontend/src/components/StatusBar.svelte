<script>
  export let course = null
  export let uploadStatus = ''
  export let onUploadStatus = () => {}
</script>

<footer>
  <div class="status-left">
    <span class="dot connected"></span>
    Connected
  </div>
  <div class="status-center">
    {#if course}
      <span class="course-label">Course:</span> {course.name}
    {:else}
      No course selected
    {/if}
  </div>
  <div class="status-right">
    {#if uploadStatus}
      <button class="job-indicator" on:click={onUploadStatus} title="Open grade upload status and student summary">{uploadStatus}</button>
    {:else}CanvasLMS v0.1.0{/if}
  </div>
</footer>

<style>
  .job-indicator { color: inherit; font-size: 11px; background: transparent; padding: 2px 6px; max-width: 480px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 24px;
    height: 32px;
    background: var(--frost-dark);
    color: rgba(255,255,255,0.8);
    font-size: 11px;
    flex-shrink: 0;
  }

  .dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    margin-right: 6px;
  }

  .dot.connected {
    background: #2ecc71;
  }

  .course-label {
    color: rgba(255,255,255,0.5);
  }
</style>
