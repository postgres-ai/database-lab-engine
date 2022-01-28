/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { Paper, makeStyles } from '@material-ui/core'
import clsx from 'clsx'

type Props = {
  title?: string
  message: string
  className?: string
  size?: 'big' | 'normal'
}

const useStyles = makeStyles({
  '*': {
    margin: 0,
  },
  root: {
    color: '#c00111',
    overflowWrap: 'break-word',
  },
  rootBig: {
    padding: '16px 24px',

    '& $title': {
      fontSize: '16px',
    },
    '& $message': {
      marginTop: '16px',
      fontSize: '14px',
    },
  },
  rootNormal: {
    padding: '8px 16px',

    '& $title': {
      fontSize: '14px',
    },
    '& $message': {
      marginTop: '12px',
      fontSize: '12px',
    },
  },
  title: {
    fontWeight: 700,
    textTransform: 'uppercase',
  },
  message: {},
})

export const ErrorStub = (props: Props) => {
  const { title = 'Unknown error', message, className, size = 'big' } = props

  const classes = useStyles()

  return (
    <Paper className={clsx(
      classes.root,
      size === 'big' && classes.rootBig,
      size === 'normal' && classes.rootNormal,
      className,
    )}>
      <h2 className={classes.title}>{title}</h2>
      <p className={classes.message}>{message}</p>
    </Paper>
  )
}
