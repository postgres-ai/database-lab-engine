/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { makeStyles } from '@material-ui/core'

import { Status } from './Status'
import { Retrieval } from './Retrieval'
import { Connection } from './Connection'
import { Disks } from './Disks'
import { Snapshots } from './Snapshots'

const useStyles = makeStyles(
  (theme) => ({
    root: {
      flex: '0 0 437px',
      minWidth: 0,

      [theme.breakpoints.down('md')]: {
        flexBasis: '300px',
      },

      [theme.breakpoints.down('sm')]: {
        flex: '1 1 100%',
        marginTop: '20px',
      },
    },
  }),
  { index: 1 },
)

export const Info = () => {
  const classes = useStyles()

  return (
    <div className={classes.root}>
      <Status />
      <Retrieval />
      <Connection />
      <Disks />
      <Snapshots />
    </div>
  )
}
