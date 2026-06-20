import { streamPost } from './api.js'

// Generic streaming-operation overlay state, shared by long-running actions that
// emit live progress (colima lifecycle, docker compose). One overlay at a time.
export const op = $state({
  active: false,
  title: null,
  lines: [],
  error: null,
  done: false,
})

export async function runOp(title, path, onDone) {
  if (op.active) return
  op.active = true
  op.title = title
  op.lines = []
  op.error = null
  op.done = false
  try {
    await streamPost(path, {
      onEvent: (name, data) => {
        if (name === 'line') op.lines = [...op.lines, data.line]
        else if (name === 'done') {
          op.done = true
          if (!data.ok) op.error = data.error
        }
      },
    })
  } catch (e) {
    op.error = e.message
    op.done = true
  } finally {
    op.active = false
    onDone?.()
  }
}

export function dismissOp() {
  op.title = null
  op.lines = []
  op.error = null
  op.done = false
}
