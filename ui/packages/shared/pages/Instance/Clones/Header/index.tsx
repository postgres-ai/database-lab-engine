/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core'

import { colors } from '@postgres.ai/shared/styles/colors'
import { formatBytesIEC } from '@postgres.ai/shared/utils/units'

type Props = {
  expectedCloningTimeS: number
  logicalSize: number | null
  clonesCount: number
  clonesCountLastMonth?: number
}

const useStyles = makeStyles((theme) => ({
  root: {
    display: 'flex',
    justifyContent: 'space-between',
    padding: '20px',

    [theme.breakpoints.down('xs')]: {
      padding: '20px 0',
    },
  },
  item: {
    fontSize: '12px',
    textAlign: 'center',
    color: colors.pgaiDarkGray,
  },
  accent: {
    fontWeight: 'bold',
    fontSize: '14px',
    color: colors.black,
  },
}))

export const Header = (props: Props) => {
  const {
    expectedCloningTimeS,
    logicalSize,
    clonesCount,
    clonesCountLastMonth,
  } = props
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <div className={classes.item}>
        <span className={classes.accent}>
          {expectedCloningTimeS ? `${expectedCloningTimeS} s` : '-'}
        </span>
        <br />
        average
        <br />
        cloning time
      </div>
      <div className={classes.item}>
        <span className={classes.accent}>
          {logicalSize ? formatBytesIEC(logicalSize) : '-'}
        </span>
        <br />
        logical
        <br />
        data size
      </div>
      <div className={classes.item}>
        <span className={classes.accent}>{clonesCount}</span>
        <br />
        clones
        <br />
        now
      </div>
      {clonesCountLastMonth && (
        <div className={classes.item}>
          <span className={classes.accent}>{clonesCountLastMonth}</span>
          <br />
          clones
          <br />
          in last month
        </div>
      )}
    </div>
  )
}
