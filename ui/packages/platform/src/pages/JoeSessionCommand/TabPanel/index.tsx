/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */
import Box from '@mui/material/Box'
import { Typography, makeStyles } from '@material-ui/core'
import { TabPanelProps } from '@postgres.ai/platform/src/components/types'

const useStyles = makeStyles(
  {
    root: {
      marginTop: 0,
    },
    content: {
      padding: '10px 0 0 0',
    },
  },
  { index: 1 },
)

export const TabPanel = (props: TabPanelProps) => {
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
