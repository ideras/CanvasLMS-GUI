<script>
  import { createEventDispatcher, onMount } from 'svelte'
  import { fly } from 'svelte/transition'
  import RichTextEditor from './RichTextEditor.svelte'
  import { CreateAnnouncement, UpdateAnnouncement, UploadAnnouncementAttachment } from '../../bindings/canvaslms-gui/app.js'
  import { toast } from 'svelte-sonner'

  export let course
  /** Pass null for create mode, or an Announcement object for edit mode. */
  export let announcement = null

  const dispatch = createEventDispatcher()

  function onKeyDown(e) { if (e.key === 'Escape') close() }
  onMount(() => {
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  })

  let title = ''
  let initialHTML = ''
  let saving = false
  let attaching = false
  let editorRef = null

  $: isEdit = announcement != null
  $: modalTitle = isEdit ? 'Edit Announcement' : 'New Announcement'

  // Set initial values when announcement prop changes
  $: if (announcement) {
    title = announcement.title || ''
    initialHTML = announcement.message || ''
  } else {
    title = ''
    initialHTML = ''
  }

  function close() {
    dispatch('close')
  }

  async function attachFile() {
    attaching = true
    try {
      const info = await UploadAnnouncementAttachment(course.id)
      if (info && editorRef) {
        editorRef.appendHTML(`<p><a href="${info.download_url}">📎 ${info.name}</a></p>`)
        toast.success('File attached: ' + info.name)
      }
    } catch (e) {
      toast.error(e.message || 'Failed to attach file')
    } finally {
      attaching = false
    }
  }

  async function save() {
    if (!title.trim()) {
      toast.error('Title is required')
      return
    }
    const message = editorRef?.getHTML() ?? ''
    saving = true
    try {
      if (isEdit) {
        await UpdateAnnouncement(course.id, announcement.id, title, message)
        toast.success('Announcement updated')
      } else {
        await CreateAnnouncement(course.id, title, message)
        toast.success('Announcement created')
      }
      dispatch('saved')
    } catch (e) {
      toast.error(e.message || 'Failed to save announcement')
    } finally {
      saving = false
    }
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-backdrop" on:click={close}>
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="modal-content" on:click|stopPropagation in:fly|global={{ y: 20, duration: 200 }}>
    <div class="modal-header">
      <h2>{modalTitle}</h2>
      <button class="btn-close" on:click={close}>✕</button>
    </div>

    <div class="modal-body">
      <label>
        Title
        <input type="text" bind:value={title} placeholder="Announcement title" autofocus={!isEdit} />
      </label>

      <div class="field">
        <div class="field-label">Message</div>
        {#key initialHTML}
          <RichTextEditor
            bind:this={editorRef}
            value={initialHTML}
            placeholder="Write your announcement…"
            autofocus={isEdit}
          />
        {/key}
      </div>
    </div>

    <div class="modal-footer">
      <button class="btn-attach" on:click={attachFile} disabled={attaching}>
        {attaching ? 'Uploading…' : '📎 Attach File'}
      </button>
      <div class="spacer"></div>
      <button class="btn-secondary" on:click={close}>Cancel</button>
      <button class="btn-primary" on:click={save} disabled={saving}>
        {saving ? 'Saving…' : isEdit ? 'Update' : 'Create'}
      </button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
    background: rgba(0, 0, 0, 0.5);
  }

  .modal-content {
    background: var(--bg-primary);
    border-radius: 12px;
    width: 680px;
    max-width: 95vw;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
    box-shadow: 0 20px 60px rgba(0,0,0,0.3);
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--border);
  }

  .modal-header h2 {
    margin: 0;
    font-size: 16px;
    color: var(--frost-dark);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-label {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .btn-close {
    background: none;
    border: none;
    font-size: 18px;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 4px 8px;
    border-radius: 4px;
  }
  .btn-close:hover { background: var(--bg-hover); }

  .modal-body {
    padding: 20px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 16px;
    flex: 1;
  }

  .modal-body label {
    display: flex;
    flex-direction: column;
    gap: 4px;
    font-size: 12px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .modal-body input {
    padding: 8px 10px;
    font-size: 13px;
    border: 1px solid var(--border);
    border-radius: 6px;
    background: var(--input-bg);
    color: var(--text-primary);
    font-family: inherit;
  }

  .modal-footer {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 20px;
    border-top: 1px solid var(--border);
  }

  .spacer { flex: 1; }

  .btn-primary {
    padding: 7px 18px;
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

  .btn-secondary {
    padding: 7px 16px;
    font-size: 13px;
    font-weight: 500;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px solid var(--border);
    border-radius: 6px;
    cursor: pointer;
  }
  .btn-secondary:hover { background: var(--bg-hover); }

  .btn-attach {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 6px 14px;
    font-size: 12px;
    font-weight: 500;
    background: var(--bg-primary);
    color: var(--text-primary);
    border: 1px dashed var(--border);
    border-radius: 6px;
    cursor: pointer;
  }
  .btn-attach:hover:not(:disabled) { border-color: var(--frost-blue); color: var(--frost-blue); }
  .btn-attach:disabled { opacity: 0.6; cursor: default; }
</style>
