// Normalize only on paste/proceed, never while the user is typing.
export function normalizeCSVPath(value) {
  let path = value.trim()
  if (path.length >= 2 && ((path[0] === '"' && path.at(-1) === '"') || (path[0] === "'" && path.at(-1) === "'"))) {
    path = path.slice(1, -1).trim()
  }
  return path
}

export function pasteCSVPath(event, value) {
  const text = event.clipboardData?.getData('text/plain')
  if (text === undefined) return value
  event.preventDefault()
  const input = event.target
  const pasted = normalizeCSVPath(text)
  return value.slice(0, input.selectionStart) + pasted + value.slice(input.selectionEnd)
}
