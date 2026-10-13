// Focus helper for use:autofocus — Svelte's a11y_autofocus warning targets the
// attribute, but modal inputs still need focus on open. Mounts run after the
// modal is in the DOM, so a rAF defers until layout settles.
export function autofocus(node, enabled = true) {
  if (!enabled) return
  requestAnimationFrame(() => node.focus())
}
