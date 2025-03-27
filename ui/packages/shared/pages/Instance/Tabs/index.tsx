/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { Link } from 'react-router-dom'
import { Tab as TabComponent, Tabs as TabsComponent } from '@material-ui/core'

import { PostgresSQLIcon } from '@postgres.ai/shared/icons/PostgresSQL'
import { useTabsStyles } from './styles'

export const TABS_INDEX = {
  OVERVIEW: 0,
  BRANCHES: 1,
  SNAPSHOTS: 2,
  CLONES: 3,
  LOGS: 4,
  CONFIGURATION: 5,
}
export interface TabsProps {
  value: number
  handleChange: (event: React.ChangeEvent<{}>, newValue: number) => void
  hasLogs: boolean
  isPlatform?: boolean
  hideInstanceTabs?: boolean
}

export const Tabs = ({
  value,
  handleChange,
  hasLogs,
  hideInstanceTabs,
}: TabsProps) => {
  const classes = useTabsStyles()

  const tabData = [
    { label: '👁️ Overview', to: '/instance', value: TABS_INDEX.OVERVIEW },
    {
      label: '🖖 Branches',
      to: '/instance/branches',
      value: TABS_INDEX.BRANCHES,
      hide: hideInstanceTabs,
    },
    {
      label: '⚡ Snapshots',
      to: '/instance/snapshots',
      value: TABS_INDEX.SNAPSHOTS,
      hide: hideInstanceTabs,
    },
    {
      label: (
        <div className={classes.flexRow}>
          <PostgresSQLIcon /> Clones
        </div>
      ),
      to: '/instance/clones',
      value: TABS_INDEX.CLONES,
      hide: hideInstanceTabs,
    },
    {
      label: '📓 Logs',
      to: '/instance/logs',
      value: TABS_INDEX.LOGS,
      disabled: !hasLogs,
    },
    {
      label: '🛠️ Configuration',
      to: '/instance/configuration',
      value: TABS_INDEX.CONFIGURATION,
      hide: hideInstanceTabs,
    },
  ]

  return (
    <TabsComponent
      value={value}
      onChange={handleChange}
      classes={{ root: classes.tabsRoot, indicator: classes.tabsIndicator }}
    >
      {tabData.map(({ label, to, value, hide, disabled }) => (
        <Link key={value} to={to}>
          <TabComponent
            label={label}
            value={value}
            disabled={disabled}
            classes={{
              root: hide ? classes.tabHidden : classes.tabRoot,
            }}
          />
        </Link>
      ))}
    </TabsComponent>
  )
}
