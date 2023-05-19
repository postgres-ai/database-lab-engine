/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  parse,
  startOfDay,
  isSameDay,
  startOfMonth,
  startOfWeek,
  endOfMonth,
  endOfWeek,
  differenceInDays,
  addDays,
  addMonths,
  isSameMonth,
  format,
  formatDistanceToNowStrict,
} from 'date-fns'

// parseDate parses date of both format: '2006-01-02 15:04:05 UTC' and `2006-01-02T15:04:05Z` (RFC3339).
export const parseDate = (dateStr: string) =>
  parse(
    dateStr.replace(' UTC', 'Z').replace('T', ' '),
    'yyyy-MM-dd HH:mm:ssX',
    new Date(),
  )

// UTCf - UTC formatted, but not actually UTC.
// date-fns using this approach because browser don't have an opportunity to switch single date
// object in different timezones.
// Example: 15:00:00 <some date> Moscow Standard Time -> 12:00:00 <same date> Moscow Standard Time.
// In example above just ignore real timezone "Moscow Standard Time" and imagine that it is UTC.
const toUTCf = (date: Date) =>
  new Date(date.getTime() + date.getTimezoneOffset() * 60 * 1000)

const toLocal = (date: Date) =>
  new Date(date.getTime() - date.getTimezoneOffset() * 60 * 1000)

const inUTC =
  // eslint-disable-next-line @typescript-eslint/no-explicit-any


    <T extends (date: Date, ...otherArgs: any[]) => Date>(func: T) =>
    (date: Date, ...otherArgs: unknown[]) =>
      toLocal(func(toUTCf(date), ...otherArgs))

export const startOfMonthUTC = inUTC(startOfMonth)

export const startOfWeekUTC = inUTC(startOfWeek)

export const endOfMonthUTC = inUTC(endOfMonth)

export const endOfWeekUTC = inUTC(endOfWeek)

export const startOfDayUTC = inUTC(startOfDay)

export const addDaysUTC = inUTC(addDays)

export const addMonthsUTC = inUTC(addMonths)

export const differenceInDaysUTC = (date1: Date, date2: Date) =>
  differenceInDays(toUTCf(date1), toUTCf(date2))

export const isSameMonthUTC = (date1: Date, date2: Date) =>
  isSameMonth(toUTCf(date1), toUTCf(date2))

export const formatUTC = (date: Date, formatStr: string) =>
  format(toUTCf(date), formatStr)

export const isSameDayUTC = (date1: Date, date2: Date) =>
  isSameDay(toUTCf(date1), toUTCf(date2))

// Std date utils.
export const formatDistanceStd = (date: Date) =>
  formatDistanceToNowStrict(date, { addSuffix: true })

export const formatDateStd = (
  date: Date,
  options?: { withDistance?: boolean },
) =>
  `${formatUTC(date, 'yyyy-MM-dd HH:mm:ss')} UTC ${
    options?.withDistance ? `(${formatDistanceStd(date)})` : ''
  }`
