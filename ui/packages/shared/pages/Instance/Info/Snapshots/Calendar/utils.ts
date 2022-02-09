/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { daysInWeek } from 'date-fns'

import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'
import { config } from '@postgres.ai/shared/config'
import {
  startOfMonthUTC,
  isSameDayUTC,
  startOfWeekUTC,
  endOfMonthUTC,
  endOfWeekUTC,
  startOfDayUTC,
  differenceInDaysUTC,
  addDaysUTC,
  addMonthsUTC,
  isSameMonthUTC,
  formatUTC
} from '@postgres.ai/shared/utils/date'

export const getMonthStartDate = () => startOfMonthUTC(new Date())

const getEdgeDates = (monthStartDate: Date) => {
  return {
    firstDate: startOfWeekUTC(monthStartDate, { locale: config.dateFnsLocale }),
    lastDate: startOfDayUTC(
      endOfWeekUTC(endOfMonthUTC(monthStartDate), {
        locale: config.dateFnsLocale,
      }),
    ),
  }
}

export const getCalendar = (monthStartDate: Date, snapshots: Snapshot[]) => {
  const { firstDate, lastDate } = getEdgeDates(monthStartDate)

  const countDays = differenceInDaysUTC(lastDate, firstDate)

  const days = Array.from({ length: countDays + 1 }, (_, i) => {
    const date = addDaysUTC(firstDate, i)

    return {
      date,
      snapshots: snapshots.filter((snapshot) =>
        isSameDayUTC(date, snapshot.dataStateAtDate),
      ),
      isBreak: i > 0 && (i + 1) % 7 === 0,
      isDisabled: !isSameMonthUTC(monthStartDate, date),
    }
  })

  const weekDays = days
    .slice(0, daysInWeek)
    .map((day) => formatUTC(day.date, 'E'))

  return {
    weekDays,
    days,
  }
}

export const getPrevMonthStartDate = (monthStartDate: Date) =>
  addMonthsUTC(monthStartDate, -1)

export const getNextMonthStartDate = (monthStartDate: Date) =>
  addMonthsUTC(monthStartDate, 1)

export const canGetNextMonthStartDate = (monthStartDate: Date) =>
  monthStartDate < getMonthStartDate()
