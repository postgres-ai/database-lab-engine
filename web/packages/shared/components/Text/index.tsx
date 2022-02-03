/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'

type Props = {
  children: React.ReactNode,
}

const useStyles = makeStyles({
  root: {
    margin: 0,
  }
})

export const Text = (props: Props) => {
  const classes = useStyles()

  return <p className={classes.root}>
    { props.children }
  </p>
}
