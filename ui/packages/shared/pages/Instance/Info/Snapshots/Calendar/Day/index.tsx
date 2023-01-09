/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core'
import clsx from 'clsx'

import { isSameDayUTC, formatUTC } from '@postgres.ai/shared/utils/date'
import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'
import { colors } from '@postgres.ai/shared/styles/colors'
import { Tooltip } from '@postgres.ai/shared/components/Tooltip'

const CELL_SIZE = 28

const useStyles = makeStyles(
  {
    root: {
      position: 'relative',
      cursor: 'default',
      flex: `0 0 ${CELL_SIZE}px`,
      background: '#f4f4f4',
      height: `${CELL_SIZE}px`,
      display: 'flex',
      borderRadius: `${CELL_SIZE / 2}px`,
      alignItems: 'center',
      justifyContent: 'center',
      marginTop: '16px',
      fontSize: '12px',
    },
    rootHasSnapshots: {
      background: colors.secondary2.lightLight,
      cursor: 'pointer',
    },
    rootCurrent: {
      border: `1px solid ${colors.secondary2.main}`,
    },
    rootDisabled: {
      opacity: '0.25',
      pointerEvents: 'none',
    },
    itemsCount: {
      top: '-8px',
      right: '-6px',
      position: 'absolute',
      fontSize: '8px',
      backgroundColor: colors.white,
      border: `1px solid ${colors.secondary2.lightLight}`,
      borderRadius: '8px',
      height: '16px',
      padding: '2px',
      minWidth: '16px',

      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
    },
    break: {
      flex: '0 0 100%',
    },
  },
  { index: 1 },
)

type Props = {
  date: Date
  snapshots: Snapshot[]
  isBreak: boolean
  isDisabled: boolean
  onClick: () => void
}

export const Day = (props: Props) => {
  const { date, isBreak, snapshots, isDisabled, onClick } = props
  const classes = useStyles()

  const today = new Date()

  const breakRendered = isBreak && <div className={classes.break} />

  const dayRendered = (
    <div
      className={clsx(
        classes.root,
        snapshots.length && classes.rootHasSnapshots,
        isDisabled && classes.rootDisabled,
        isSameDayUTC(today, date) && classes.rootCurrent,
      )}
      onClick={snapshots.length ? onClick : void 0}
    >
      {!!snapshots.length && (
        <div className={classes.itemsCount}>{snapshots.length}</div>
      )}
      {formatUTC(date, 'd')}
    </div>
  )

  if (snapshots.length) {
    const contentRendered = (
      <>
        {snapshots.map((snapshot) => {
          return <div key={snapshot.id}>{snapshot.dataStateAt}</div>
        })}
      </>
    )

    return (
      <>
        <Tooltip content={contentRendered} disableTouchListener>
          {dayRendered}
        </Tooltip>
        {breakRendered}
      </>
    )
  }

  return (
    <>
      {dayRendered}
      {breakRendered}
    </>
  )
}
