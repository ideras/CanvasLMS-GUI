import { test } from 'node:test'
import assert from 'node:assert/strict'
import { normalizeCSVPath, pasteCSVPath } from './upload.js'

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
