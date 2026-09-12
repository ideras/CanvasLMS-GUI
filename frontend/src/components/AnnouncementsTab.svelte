<script>
  import { DeleteAnnouncement, GetAnnouncements, DownloadAnnouncementFile } from '../../wailsjs/go/main/App.js'
  import { BrowserOpenURL } from '../../wailsjs/runtime/runtime.js'
  import { toasts } from 'svelte-toasts'
  import { onMount } from 'svelte'
  import { fly } from 'svelte/transition'
  import AnnouncementComposerModal from './AnnouncementComposerModal.svelte'

  export let course

  let announcements = []
  let loading = true
  let error = null
  let showComposer = false
  let composerAnnouncement = null  // null = create, object = edit

  onMount(loadAnnouncements)

  async function loadAnnouncements() {
    loading = true
    error = null
    try {
      announcements = await GetAnnouncements(course.id) || []
    } catch (e) {
      error = e.message || 'Failed to load announcements'
    } finally {
      loading = false
    }
  }

  function openNew() { composerAnnouncement = null; showComposer = true }
  function openEdit(ann) { composerAnnouncement = ann; showComposer = true }
  function closeComposer() { showComposer = false }

  async function removeAnnouncement(ann) {
    if (!confirm(`Delete "${ann.title}"?`)) return
    try {
      await DeleteAnnouncement(course.id, ann.id)
      toasts.success('Announcement deleted')
      await loadAnnouncements()
    } catch (e) {
      toasts.error(e.message || 'Failed to delete announcement')
    }
  }

  function handleLinkClick(e) {
    const anchor = e.target.closest('a')
    if (!anchor) return
    const href = anchor.getAttribute('href')
    if (!href) return
    e.preventDefault()

    if (href.includes('/files/') && href.match(/\/courses\/\d+/)) {
      let dl = href
      if (!dl.includes('/download')) dl = dl.replace(/\/$/, '') + '/download'
      const name = (anchor.textContent || '').trim()
      downloadFile(dl, name)
      return
    }
    BrowserOpenURL(href)
  }

  async function downloadFile(url, name) {
    try {
      await DownloadAnnouncementFile(url, name)
    } catch (e) {
      toasts.error(e.message || 'Download failed')
    }
  }

  function formatDate(d) {
    if (!d) return ''
    const dt = new Date(d)
    return dt.toLocaleDateString('en-US', {
      year: 'numeric', month: 'short', day: 'numeric',
      hour: 'numeric', minute: '2-digit'
    })
  }
</script>

<div class="announcements-tab">
  <div class="toolbar">
    <h3>Anuncios</h3>
    <button class="btn-primary" on:click={openNew}>+ New Announcement</button>
  </div>

  {#if loading}
    <p class="status">Loading…</p>
  {:else if error}
    <p class="status error">{error}</p>
  {:else if announcements.length === 0}
    <p class="status empty">No announcements yet.</p>
  {:else}
    <div class="announcement-list">
      {#each announcements as ann (ann.id)}
        <div class="announcement-card" in:fly={{ y: 10, duration: 150 }}>
          <div class="card-header">
            <div class="card-title-row">
              <h4>{ann.title}</h4>
              <div class="card-actions">
                <button class="icon-btn-sm" on:click={() => openEdit(ann)} title="Edit">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                </button>
                <button class="icon-btn-sm danger" on:click={() => removeAnnouncement(ann)} title="Delete">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <polyline points="3 6 5 6 21 6"/>
                    <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                </button>
              </div>
            </div>
            <div class="card-meta">
              <span class="author">{ann.author?.display_name || 'Unknown'}</span>
              <span class="date">{formatDate(ann.posted_at)}</span>
              {#if ann.discussion_subentry_count > 0}
                <span class="replies">{ann.discussion_subentry_count} replies</span>
              {/if}
            </div>
          </div>
          <!-- svelte-ignore a11y-click-events-have-key-events -->
          <div class="card-body" on:click={handleLinkClick}>
            {@html ann.message}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if showComposer}
  <AnnouncementComposerModal
    {course}
    announcement={composerAnnouncement}
    on:close={closeComposer}
    on:saved={async () => { closeComposer(); await loadAnnouncements() }}
  />
{/if}

<style>
  .announcements-tab {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .toolbar h3 {
    font-size: 15px;
    color: var(--frost-dark);
    margin: 0;
  }

  .btn-primary {
    padding: 6px 16px;
    font-size: 13px;
    font-weight: 600;
    background: var(--frost-blue);
    color: white;
    border-radius: 6px;
    border: none;
    cursor: pointer;
    transition: background 0.15s;
  }
  .btn-primary:hover:not(:disabled) { background: #2e86c1; }
  .btn-primary:disabled { opacity: 0.6; cursor: default; }

  .announcement-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .announcement-card {
    background: var(--bg-primary);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }

  .card-header {
    padding: 14px 16px 10px;
    border-bottom: 1px solid var(--border);
  }

  .card-title-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 8px;
  }

  .card-title-row h4 { margin: 0; font-size: 14px; color: var(--frost-dark); }

  .card-actions { display: flex; gap: 4px; flex-shrink: 0; }

  .icon-btn-sm {
    background: none;
    border: none;
    padding: 4px;
    border-radius: 4px;
    cursor: pointer;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
  }
  .icon-btn-sm:hover { background: var(--bg-hover); color: var(--text-primary); }
  .icon-btn-sm.danger:hover { color: #e74c3c; }

  .card-meta {
    display: flex; gap: 12px; margin-top: 6px;
    font-size: 11px; color: var(--text-secondary);
  }
  .card-meta .replies { color: var(--frost-blue); font-weight: 500; }

  .card-body {
    padding: 14px 16px;
    font-size: 13px; line-height: 1.6;
    color: var(--text-primary); cursor: default;
  }
  .card-body :global(a) { color: var(--frost-blue); cursor: pointer; text-decoration: underline; }
  .card-body :global(a:hover) { color: #2e86c1; }

  .status { text-align: center; padding: 32px; color: var(--text-secondary); }
  .status.error { color: #e74c3c; }
  .status.empty { font-style: italic; }
</style>
