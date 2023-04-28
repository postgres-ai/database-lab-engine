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
import { PostgresSQL } from '@postgres.ai/shared/icons/PostgresSQL'

export const TABS_INDEX = {
  OVERVIEW: 0,
  BRANCHES: 1,
  SNAPSHOTS: 2,
  CLONES: 3,
  LOGS: 4,
  CONFIGURATION: 5,
}

const useStyles = makeStyles(
  {
    tabsRoot: {
      minHeight: 0,
      marginTop: '-8px',

      '& .MuiTabs-fixed': {
        overflowX: 'auto!important',
      },

      '& .postgres-logo': {
        width: '18px',
        height: '18px',
      },
    },

    flexRow: {
      display: 'flex',
      flexDirection: 'row',
      gap: '5px',
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
  isConfigActive?: boolean
}

export const Tabs = (props: Props) => {
  const classes = useStyles()

    const { value, handleChange, hasLogs, isConfigActive, hideInstanceTabs } =
        props

  return (
    <TabsComponent
      value={value}
      onChange={handleChange}
      classes={{ root: classes.tabsRoot, indicator: classes.tabsIndicator }}
    >
      <TabComponent
        label="👁️ Overview"
        classes={{
          root: classes.tabRoot,
        }}
        value={TABS_INDEX.OVERVIEW}
      />
      <TabComponent
        label="🖖 Branches"
        classes={{
          root: props.hideInstanceTabs ? classes.tabHidden : classes.tabRoot,
        }}
        value={TABS_INDEX.BRANCHES}
      />
      <TabComponent
        label="⚡ Snapshots"
        classes={{
          root: props.hideInstanceTabs ? classes.tabHidden : classes.tabRoot,
        }}
        value={TABS_INDEX.SNAPSHOTS}
      />
      <TabComponent
        label={
          <div className={classes.flexRow}>
            <PostgresSQL /> Clones
          </div>
        }
        classes={{
            root: hideInstanceTabs ? classes.tabHidden : classes.tabRoot,
        }}
        value={TABS_INDEX.CLONES}
      />
      <TabComponent
        label="📓 Logs"
        disabled={!hasLogs}
        classes={{
            root:
                props.hideInstanceTabs || !isConfigActive
                    ? classes.tabHidden
                    : classes.tabRoot,
        }}
        value={TABS_INDEX.LOGS}
      />
      <TabComponent
        label="🛠️ Configuration"
        classes={{
            root: props.hideInstanceTabs ? classes.tabHidden : classes.tabRoot,
        }}
        value={TABS_INDEX.CONFIGURATION}
      />
    </TabsComponent>
  )
}
