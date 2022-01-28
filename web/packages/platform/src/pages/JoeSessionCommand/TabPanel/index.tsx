/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { Typography, Box, makeStyles } from '@material-ui/core'

type Props = {
  children: React.ReactNode
  index: number
  value: number
}

const useStyles = makeStyles({
  root: {
    marginTop: 0,
  },
  content: {
    padding: '10px 0 0 0',
  },
})

export const TabPanel = (props: Props) => {
  const { children, value, index } = props

  const classes = useStyles()

  return (
    <Typography
      component="div"
      role="tabpanel"
      hidden={value !== index}
      id={`plan-tabpanel-${index}`}
      aria-labelledby={`plan-tab-${index}`}
      className={classes.root}
    >
      <Box p={3} className={classes.content}>
        {children}
      </Box>
    </Typography>
  )
}
