import { test } from 'node:test'
import assert from 'node:assert/strict'
import { normalizeCSVPath, pasteCSVPath, jobLabel, formatElapsed, summaryText, summaryDetails } from './upload.js'

test('CSV paths trim whitespace and balanced Explorer quotes', () => {
  for (const value of ['  "C:\\My Documents\\grades.csv"  ', " 'C:\\My Documents\\grades.csv' ", ' C:\\My Documents\\grades.csv ']) {
    assert.equal(normalizeCSVPath(value), 'C:\\My Documents\\grades.csv')
  }
  assert.equal(normalizeCSVPath(' /tmp/my grades.csv '), '/tmp/my grades.csv')
  assert.equal(normalizeCSVPath(' "" '), '')
})

test('paste replaces selection with normalized path', () => {
  let prevented = false
  const event = {
    clipboardData: { getData: () => ' " /tmp/grades.csv " ' },
    target: { selectionStart: 0, selectionEnd: 3 },
    preventDefault() { prevented = true },
  }
  assert.equal(pasteCSVPath(event, 'old'), '/tmp/grades.csv')
  assert.equal(prevented, true)
})

test('job UI uses states and elapsed time, not completion percentages', () => {
  for (const state of ['queued', 'running', 'completed', 'failed']) {
    assert.equal(jobLabel(state), state[0].toUpperCase() + state.slice(1))
  }
  assert.equal(formatElapsed(65.9), '1m 5s')
  assert.equal(formatElapsed(-1), '0m 0s')
})

test('background notifications retain skipped and failed student details', () => {
  const summary = { total: 2, skipped: [{ student_id: '9', row: 3, reason: 'not in roster' }], failed: [{ student_id: '2', reason: 'rejected' }] }
  assert.match(summaryText(summary), /2 matched; 1 skipped; 1 Canvas/)
  assert.match(summaryDetails(summary), /Skipped 9 \(row 3\): not in roster/)
  assert.match(summaryDetails(summary), /Canvas 2: rejected/)
})
