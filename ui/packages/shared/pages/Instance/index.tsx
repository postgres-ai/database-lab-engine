/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect } from 'react'
import { makeStyles } from '@material-ui/core'
import { observer } from 'mobx-react-lite'

import { Button } from '@postgres.ai/shared/components/Button2'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'

import { TABS_INDEX, Tabs, TabsProps } from './Tabs'
import { PlatformTabs } from './Tabs/PlatformTabs'
import { Logs } from '../Logs'
import { Clones } from './Clones'
import { Info } from './Info'
import { Snapshots } from './Snapshots'
import { Branches } from '../Branches'
import { Configuration } from '../Configuration'
import { ClonesModal } from './Clones/ClonesModal'
import { SnapshotsModal } from './Snapshots/components/SnapshotsModal'
import { InactiveInstance } from './InactiveInstance'
import { Host, HostProvider, StoresProvider } from './context'

import Typography from '@material-ui/core/Typography'
import Box from '@mui/material/Box'

import { useCreatedStores } from './useCreatedStores'

import './styles.scss'

type Props = Host

const useStyles = makeStyles(
  (theme) => ({
    title: {
      marginTop: '8px',
    },
    reloadButton: {
      flex: '0 0 auto',
      alignSelf: 'flex-start',
    },
    errorStub: {
      marginTop: '16px',
    },
    content: {
      display: 'flex',
      marginTop: '16px',
      position: 'relative',
      flex: '1 1 100%',
      height: '100%',

      [theme.breakpoints.down('sm')]: {
        flexDirection: 'column',
      },
    },
  }),
  { index: 1 },
)

export const Instance = observer((props: Props) => {
  const classes = useStyles()

  const { instanceId, api } = props
  const [activeTab, setActiveTab] = React.useState(
    props?.renderCurrentTab || TABS_INDEX.OVERVIEW,
  )
  const [hasBeenRedirected, setHasBeenRedirected] = React.useState(false);

  const stores = useCreatedStores(props)
  const {
    instance,
    instanceError,
    instanceRetrieval,
    isLoadingInstance,
    load,
  } = stores.main

  const switchTab = (_: React.ChangeEvent<{}> | null, tabID: number) => {
    const contentElement = document.getElementById('content-container')
    setActiveTab(tabID)

    if (tabID === 0) {
      load(props.instanceId)
    }
    contentElement?.scroll(0, 0)
  }

  const isInstanceIntegrated =
    instanceRetrieval ||
    (!isLoadingInstance && instance && instance?.url && !instanceError)

  const isConfigurationActive = instanceRetrieval?.mode !== 'physical'

  const InstanceTab = (props: TabsProps) =>
    !props.isPlatform ? <Tabs {...props} /> : <PlatformTabs {...props} />

  useEffect(() => {
    load(instanceId)
  }, [instanceId])

  useEffect(() => {
    if (
      instance &&
      instance?.state.retrieving?.status === 'pending' &&
      isConfigurationActive &&
      !props.isPlatform && !hasBeenRedirected
    ) {
      setActiveTab(TABS_INDEX.CONFIGURATION)
      setHasBeenRedirected(true)
    }
  }, [instance, hasBeenRedirected])

  return (
    <HostProvider value={props}>
      <StoresProvider value={stores}>
        {props.elements.breadcrumbs}
        <SectionTitle
          text={props.title}
          level={1}
          tag="h1"
          className={classes.title}
          rightContent={
            <Button
              onClick={() => load(props.instanceId)}
              isDisabled={!instance && !instanceError}
              className={classes.reloadButton}
            >
              Reload info
            </Button>
          }
        >
          {isInstanceIntegrated && (
            <InstanceTab
              value={activeTab}
              handleChange={switchTab}
              isPlatform={props.isPlatform}
              hasLogs={api.initWS != undefined}
            />
          )}
        </SectionTitle>

        {instanceError && (
          <ErrorStub {...instanceError} className={classes.errorStub} />
        )}

        {isInstanceIntegrated ? (
          <>
            <TabPanel value={activeTab} index={TABS_INDEX.OVERVIEW}>
              {!instanceError && (
                <div className={classes.content}>
                  {instance && instance?.state.retrieving?.status ? (
                    <>
                      <Clones />
                      <Info />
                    </>
                  ) : (
                    <StubSpinner />
                  )}
                </div>
              )}

              <ClonesModal />

              <SnapshotsModal />
            </TabPanel>

            {!props.isPlatform && (
              <>
                <TabPanel value={activeTab} index={TABS_INDEX.CLONES}>
                  {activeTab === TABS_INDEX.CLONES && (
                    <div className={classes.content}>
                      {!instanceError &&
                        (instance ? (
                          <Clones onlyRenderList />
                        ) : (
                          <StubSpinner />
                        ))}
                    </div>
                  )}
                </TabPanel>
                <TabPanel value={activeTab} index={TABS_INDEX.LOGS}>
                  {activeTab === TABS_INDEX.LOGS && <Logs api={api} />}
                </TabPanel>
                <TabPanel value={activeTab} index={TABS_INDEX.CONFIGURATION}>
                  {activeTab === TABS_INDEX.CONFIGURATION && (
                    <Configuration
                      switchActiveTab={switchTab}
                      isConfigurationActive={isConfigurationActive}
                      reload={() => load(props.instanceId)}
                      disableConfigModification={
                        instance?.state.engine.disableConfigModification
                      }
                    />
                  )}
                </TabPanel>
                <TabPanel value={activeTab} index={TABS_INDEX.SNAPSHOTS}>
                  {activeTab === TABS_INDEX.SNAPSHOTS && <Snapshots />}
                </TabPanel>
                <TabPanel value={activeTab} index={TABS_INDEX.BRANCHES}>
                  {activeTab === TABS_INDEX.BRANCHES && <Branches />}
                </TabPanel>
              </>
            )}
          </>
        ) : !isLoadingInstance && !instanceError ? (
          <TabPanel value={activeTab} index={activeTab}>
            <InactiveInstance
              instance={instance}
              org={(props.elements.breadcrumbs as any)?.props.org}
            />
          </TabPanel>
        ) : (
          !instanceError && (
            <TabPanel value={activeTab} index={activeTab}>
              <div className={classes.content}>
                <StubSpinner />
              </div>
            </TabPanel>
          )
        )}
      </StoresProvider>
    </HostProvider>
  )
})

function TabPanel(props: {
  children?: React.ReactNode
  index: number
  value: number
}) {
  const { children, value, index, ...other } = props

  return (
    <Typography
      component="div"
      role="tabpanel"
      hidden={value !== index}
      id={`scrollable-auto-tabpanel-${index}`}
      aria-labelledby={`scrollable-auto-tab-${index}`}
      {...other}
    >
      <Box p={3} sx={{ height: '100%' }}>
        {children}
      </Box>
    </Typography>
  )
}
