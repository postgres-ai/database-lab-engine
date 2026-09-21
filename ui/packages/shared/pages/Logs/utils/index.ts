export const LOGS_FILTER_KEY = 'logsFilter'

// The filter state is persisted across reloads, so anything may be sitting under the key by
// the time the page reads it back. An absent or unusable entry falls back to the defaults the
// page seeds instead of throwing out of a render or out of a socket message handler.
export const readLogsFilterState = (): Record<string, boolean> => {
  const stored = localStorage.getItem(LOGS_FILTER_KEY)

  if (!stored) return {}

  try {
    const parsed: unknown = JSON.parse(stored)

    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
      return {}
    }

    return parsed as Record<string, boolean>
  } catch {
    return {}
  }
}

export const stringWithoutBrackets = (val: string | undefined) =>
  String(val).replace(/[[\]]/g, '')

export const stringContainsPattern = (
  target: string,
  pattern = [
    'base.go',
    'runners.go',
    'snapshots.go',
    'util.go',
    'logging.go',
    'ws.go',
  ],
) => {
  let value: number = 0
  pattern.forEach(function (word) {
    value = value + Number(target?.includes(word))
  })
  return value === 1
}
