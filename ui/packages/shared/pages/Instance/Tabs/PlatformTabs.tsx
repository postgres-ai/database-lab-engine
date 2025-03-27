/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import { Link, useParams } from 'react-router-dom'
import { Tab as TabComponent, Tabs as TabsComponent } from '@material-ui/core'

import { TABS_INDEX } from '.'
import { useTabsStyles } from './styles'
import { PostgresSQLIcon } from '@postgres.ai/shared/icons/PostgresSQL'

type Props = {
  value: number
  handleChange: (event: React.ChangeEvent<{}>, newValue: number) => void
  hasLogs: boolean
  isPlatform?: boolean
  hideInstanceTabs?: boolean
}

export const PlatformTabs = ({
  value,
  handleChange,
  hasLogs,
  hideInstanceTabs,
}: Props) => {
  const classes = useTabsStyles()
  const { org, instanceId } = useParams<{ org: string; instanceId: string }>()

  const tabs = [
    {
      label: '👁️ Overview',
      to: 'overview',
      value: TABS_INDEX.OVERVIEW,
    },
    {
      label: '🖖 Branches',
      to: 'branches',
      value: TABS_INDEX.BRANCHES,
      hide: hideInstanceTabs,
    },
    {
      label: '⚡ Snapshots',
      to: 'snapshots',
      value: TABS_INDEX.SNAPSHOTS,
      hide: hideInstanceTabs,
    },
    {
      label: (
        <div className={classes.flexRow}>
          <PostgresSQLIcon /> Clones
        </div>
      ),
      to: 'clones',
      value: TABS_INDEX.CLONES,
      hide: hideInstanceTabs,
    },
    {
      label: '📓 Logs',
      to: 'logs',
      value: TABS_INDEX.LOGS,
      disabled: !hasLogs,
      hide: hideInstanceTabs,
    },
    {
      label: '🛠️ Configuration',
      to: 'configuration',
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
      {tabs.map(({ label, to, value, hide, disabled }) => (
        <Link key={value} to={`/${org}/instances/${instanceId}/${to}`}>
          <TabComponent
            label={label}
            value={value}
            disabled={disabled}
            classes={{
              root: hide ? classes.tabHidden : classes.tabRoot,
            }}
            onClick={(event) => handleChange(event, value)}
          />
        </Link>
      ))}
    </TabsComponent>
  )
}
