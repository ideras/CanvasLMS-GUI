<script>
  import { onMount, onDestroy } from 'svelte'
  import Quill from 'quill'
  import 'quill/dist/quill.snow.css'

  export let value = ''
  export let placeholder = 'Write your announcement…'
  /** If true, editor grabs focus on mount (for edit mode). Default true. */
  export let autofocus = true

  let editorEl
  let quill
  let internalChange = false

  onMount(() => {
    quill = new Quill(editorEl, {
      theme: 'snow',
      placeholder,
      modules: {
        toolbar: [
          ['bold', 'italic', 'underline'],
          [{ list: 'ordered' }, { list: 'bullet' }],
          ['blockquote', 'code-block'],
          [{ header: [2, 3, false] }],
          ['clean']
        ]
      }
    })

    if (value) quill.clipboard.dangerouslyPasteHTML(value)

    // Modal transitions can steal focus — only grab it when requested.
    if (autofocus) setTimeout(() => quill?.focus(), 80)

    quill.on('text-change', () => {
      internalChange = true
      value = quill.root.innerHTML
    })
  })

  onDestroy(() => {
    quill = null
  })

  /** Called by parent to get current HTML at save time. */
  export function getHTML() {
    return quill?.root?.innerHTML ?? ''
  }

  /** Called by parent to append HTML (e.g. file attachment link). */
  export function appendHTML(html) {
    if (!quill) return
    const range = quill.getSelection() || { index: quill.getLength(), length: 0 }
    quill.clipboard.dangerouslyPasteHTML(range.index, html)
    quill.setSelection(range.index + quill.getLength() - range.index, 0)
  }
</script>

<div bind:this={editorEl} class="quill-editor"></div>

<style>
  .quill-editor {
    border-radius: 6px;
    overflow: hidden;
  }

  :global(.quill-editor .ql-toolbar) {
    border-color: var(--border) !important;
    background: var(--bg-primary);
    border-radius: 6px 6px 0 0;
  }
  :global(.quill-editor .ql-toolbar button) { color: var(--text-secondary); }
  :global(.quill-editor .ql-toolbar button:hover),
  :global(.quill-editor .ql-toolbar button.ql-active) { color: var(--frost-blue); }
  :global(.quill-editor .ql-toolbar .ql-stroke) { stroke: var(--text-secondary); }
  :global(.quill-editor .ql-toolbar button:hover .ql-stroke),
  :global(.quill-editor .ql-toolbar button.ql-active .ql-stroke) { stroke: var(--frost-blue); }
  :global(.quill-editor .ql-toolbar .ql-fill) { fill: var(--text-secondary); }
  :global(.quill-editor .ql-toolbar button:hover .ql-fill),
  :global(.quill-editor .ql-toolbar button.ql-active .ql-fill) { fill: var(--frost-blue); }

  :global(.quill-editor .ql-container) {
    border-color: var(--border) !important;
    border-radius: 0 0 6px 6px;
    background: var(--input-bg);
    color: var(--text-primary);
    font-family: inherit;
    font-size: 13px;
  }

  :global(.quill-editor .ql-editor) {
    min-height: 200px;
    max-height: 360px;
    overflow-y: auto;
  }

  :global(.quill-editor .ql-editor.ql-blank::before) {
    color: var(--text-secondary);
    font-style: italic;
  }
</style>
