/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  makeStyles,
  Tabs as TabsComponent,
  Tab as TabComponent,
} from '@material-ui/core'

import { colors } from '@postgres.ai/shared/styles/colors'

const useStyles = makeStyles({
  tabsRoot: {
    minHeight: 0,
    marginTop: '-8px',
  },
  tabsIndicator: {
    height: '3px',
  },
  tabRoot: {
    fontWeight: 400,
    minWidth: 0,
    minHeight: 0,
    padding: '6px 16px',
    borderBottom: `3px solid ${colors.consoleStroke}`,

    '& + $tabRoot': {
      marginLeft: '10px',
    },

    '&.Mui-disabled': {
      opacity: 1,
      color: colors.pgaiDarkGray,
    },
  },
})

export const Tabs = () => {
  const classes = useStyles()

  return (
    <TabsComponent
      value={0}
      classes={{ root: classes.tabsRoot, indicator: classes.tabsIndicator }}
    >
      <TabComponent
        label="Overview"
        classes={{
          root: classes.tabRoot,
        }}
      />
      {/* // TODO(Anton): Probably will be later. */}
      {/* <TabComponent
        label='Snapshots'
        disabled
        classes={{
          root: classes.tabRoot
        }}
      /> */}
    </TabsComponent>
  )
}
