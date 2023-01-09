/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { makeStyles } from '@material-ui/core'
import Box from '@mui/material/Box'
import clsx from 'clsx'

const useStyles = makeStyles(
  {
    root: {
      padding: '20px 0',
      flex: '1 1 100%',
    },
  },
  { index: 1 },
)

type Props = {
  className?: string
  children: React.ReactNode
}

export const StubContainer = (props: Props) => {
  const classes = useStyles()

  return (
    <Box
      display="flex"
      justifyContent="center"
      alignItems="center"
      className={clsx(classes.root, props.className)}
    >
      {props.children}
    </Box>
  )
}
