/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState } from 'react'
import { makeStyles, IconButton } from '@material-ui/core'
import { ArrowLeft, ArrowRight } from '@material-ui/icons'

import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'
import { colors } from '@postgres.ai/shared/styles/colors'
import { formatUTC } from '@postgres.ai/shared/utils/date'

import { Day } from './Day'

import {
  getPrevMonthStartDate,
  getNextMonthStartDate,
  canGetNextMonthStartDate,

  getMonthStartDate,
  getCalendar,
} from './utils'

const useStyles = makeStyles({
  root: {
    marginTop: '8px',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
  },
  title: {
    fontWeight: 700,
    fontSize: '12px',
  },
  button: {
    padding: '8px',

    '&:disabled': {
      pointerEvents: 'auto',
      cursor: 'not-allowed',
    },
  },
  buttonIcon: {
    fontSize: '20px',
  },
  days: {
    display: 'flex',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
  },
  weekDay: {
    fontSize: '12px',
    flex: '0 0 24px',
    textAlign: 'center',
    color: colors.pgaiDarkGray,
    marginTop: '12px'
  }
})

type Props = {
  snapshots: Snapshot[]
  onSelectDate: (date: Date) => void
}

export const Calendar = (props: Props) => {
  const { snapshots, onSelectDate } = props

  const [monthStartDate, setMonthStartDate] = useState(getMonthStartDate)

  const classes = useStyles()

  const nextMonth = () => {
    const nextMonthStartDate = getNextMonthStartDate(monthStartDate)
    setMonthStartDate(nextMonthStartDate)
  }

  const prevMonth = () => {
    const prevMonthStartDate = getPrevMonthStartDate(monthStartDate)
    setMonthStartDate(prevMonthStartDate)
  }

  const calendar = getCalendar(monthStartDate, snapshots)

  return (
    <div className={classes.root}>
      <div className={classes.header}>
        <span className={classes.title}>{formatUTC(monthStartDate, 'MMMM y')} UTC</span>
        <div>
          <IconButton onClick={prevMonth} className={classes.button}>
            <ArrowLeft className={classes.buttonIcon} />
          </IconButton>
          <IconButton
            onClick={nextMonth}
            disabled={!canGetNextMonthStartDate(monthStartDate)}
            className={classes.button}
          >
            <ArrowRight className={classes.buttonIcon} />
          </IconButton>
        </div>
      </div>
      <div className={classes.days}>
        { calendar.weekDays.map(weekDay => {
          return <div key={weekDay} className={classes.weekDay}>
            { weekDay }
          </div>
        }) }
      </div>
      <div className={classes.days}>
        {calendar.days.map((day) => (
          <Day
            onClick={() => onSelectDate(day.date)}
            isBreak={day.isBreak}
            isDisabled={day.isDisabled}
            date={day.date}
            snapshots={day.snapshots}
            key={day.date.getTime()}
          />
        ))}
      </div>
    </div>
  )
}
