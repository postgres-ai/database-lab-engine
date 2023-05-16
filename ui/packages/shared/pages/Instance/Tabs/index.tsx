/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import {
  makeStyles,
  Tab as TabComponent,
  Tabs as TabsComponent,
} from '@material-ui/core'
import { colors } from '@postgres.ai/shared/styles/colors'

const useStyles = makeStyles(
  {
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
    tabHidden: {
      display: 'none',
    },
  },
  { index: 1 },
)

type Props = {
  value: number
  handleChange: (event: React.ChangeEvent<{}>, newValue: number) => void
  hasLogs: boolean
  hideInstanceTabs?: boolean
}

export const Tabs = (props: Props) => {
  const classes = useStyles()

  const { value, handleChange, hasLogs, hideInstanceTabs } = props

  return (
    <TabsComponent
      value={value}
      onChange={handleChange}
      classes={{ root: classes.tabsRoot, indicator: classes.tabsIndicator }}
    >
      <TabComponent
        label="Overview"
        classes={{
          root: classes.tabRoot,
        }}
        value={0}
      />
      <TabComponent
        label="Logs"
        disabled={!hasLogs}
        classes={{
          root: hideInstanceTabs ? classes.tabHidden : classes.tabRoot,
        }}
        value={1}
      />
      <TabComponent
        label="Configuration"
        classes={{
          root: props.hideInstanceTabs ? classes.tabHidden : classes.tabRoot,
        }}
        value={2}
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
