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

export function jobLabel(state) {
  return ({ preparing: 'Preparing', submitting: 'Submitting', queued: 'Queued', running: 'Running', completed: 'Completed', failed: 'Failed', cancelled: 'Cancelled', stopped: 'Monitoring stopped', unknown: 'Submission outcome unknown' })[state] || 'Waiting for Canvas'
}

export function formatElapsed(seconds) {
  const elapsed = Math.max(0, Math.floor(seconds))
  return `${Math.floor(elapsed / 60)}m ${elapsed % 60}s`
}

export function summaryText(summary) {
  return `${summary.total || 0} matched; ${summary.skipped?.length || 0} skipped; ${summary.failed?.length || 0} Canvas failures/result warnings`
}

export function summaryDetails(summary) {
  return [
    ...(summary.skipped || []).map(s => `Skipped ${s.student_id} (row ${s.row}): ${s.reason}`),
    ...(summary.failed || []).map(s => `Canvas ${s.student_id || 'result'}: ${s.reason}`),
    summary.message || '',
  ].filter(Boolean).join('\n')
}
