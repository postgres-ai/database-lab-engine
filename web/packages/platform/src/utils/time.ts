const pluralize = (single: string, multiple: string) => (val: number) =>
  val === 1 ? single : multiple

const MS_IN_SECOND = 1000
const MS_IN_MINUTE = MS_IN_SECOND * 60
const MS_IN_HOUR = MS_IN_MINUTE * 60
const MS_IN_DAY = MS_IN_HOUR * 24

const INTERVALS = [
  { duration: MS_IN_DAY, getName: pluralize('day', 'days') },
  { duration: MS_IN_HOUR, getName: pluralize('hour', 'hours') },
  { duration: MS_IN_MINUTE, getName: pluralize('minute', 'minutes') },
  { duration: MS_IN_SECOND, getName: pluralize('second', 'seconds') },
]

const splitDuration = (durationMS: number) => {
  let rest = durationMS

  return INTERVALS.map((interval) => {
    const value = Math.floor(rest / interval.duration)
    rest = rest % interval.duration
    return {
      value,
      name: interval.getName(value),
    }
  })
}

export const formatDuration = (durationMS: number) => {
  const parts = splitDuration(durationMS)

  let result = ''

  parts.forEach((part, i) => {
    const isLast = i === parts.length - 1

    const shouldDisplay = part.value || (!result.length && isLast)
    if (shouldDisplay) result += ` ${part.value} ${part.name}`
  })

  return result
}
