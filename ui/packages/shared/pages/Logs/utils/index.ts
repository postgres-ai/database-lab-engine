export const stringWithoutBrackets = (val: string | undefined) =>
  String(val).replace(/[\[\]]/g, '')

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
