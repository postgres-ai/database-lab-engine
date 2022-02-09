/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

type Props = {
  children: React.ReactNode
}

const useStyles = makeStyles({
  root: {
    fontWeight: 'bold',
    whiteSpace: 'nowrap',
  },
})

export const ImportantText = (props: Props) => {
  const classes = useStyles()

  return <span className={classes.root}>{props.children}</span>
}
