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
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'

import { Tabs } from './Tabs'
import { Clones } from './Clones'
import { Info } from './Info'
import { establishConnection } from './wsLogs'
import { Configuration } from '../Configuration'
import { ClonesModal } from './ClonesModal'
import { SnapshotsModal } from './SnapshotsModal'
import { Host, HostProvider, StoresProvider } from './context'

import PropTypes from "prop-types";
import Typography from "@material-ui/core/Typography";
import Box from "@material-ui/core/Box";
import Alert from '@material-ui/lab/Alert';
import AlertTitle from '@material-ui/lab/AlertTitle';

import { useCreatedStores } from './useCreatedStores'

import './styles.scss';

type Props = Host

const useStyles = makeStyles((theme) => ({
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

    [theme.breakpoints.down('sm')]: {
      flexDirection: 'column',
    },
  },
  spinnerContainer: {
    display: "flex",
    width: "100%",
    alignItems: "center",
    justifyContent: "center"
  }
}))

export const Instance = observer((props: Props) => {
  const classes = useStyles()

  const { instanceId, api } = props

  const stores = useCreatedStores(props)

  useEffect(() => {
    stores.main.load(instanceId)
  }, [instanceId])

  const { instance, instanceError, instanceRetrieval } = stores.main
  const isConfigurationActive = instanceRetrieval?.mode !== 'physical'

  useEffect(() => {
    if (
      instance &&
      instance?.state.retrieving?.status === 'pending' &&
      isConfigurationActive
    ) {
      setActiveTab(2)
    }
    if (instance && !instance?.state?.pools) {
      if (!props.callbacks) return

      props.callbacks.showDeprecatedApiBanner()
      return props.callbacks?.hideDeprecatedApiBanner
    }
  }, [instance])

  const [activeTab, setActiveTab] = React.useState(0);

  const [isLogConnectionEnabled, enableLogConnection] = React.useState(false);

  const switchTab = (_: React.ChangeEvent<{}>, tabID: number) => {
    if (tabID == 1 && api.initWS != undefined && !isLogConnectionEnabled) {
      establishConnection(api).then(() => {
        enableLogConnection(true)
      });
    }

    setActiveTab(tabID);
  };

  return (
    <HostProvider value={props}>
      <StoresProvider value={stores}>
        <>
          { props.elements.breadcrumbs }
          <SectionTitle
            text={props.title}
            level={1}
            tag="h1"
            className={classes.title}
            rightContent={
              <Button
                onClick={() => stores.main.load(props.instanceId)}
                isDisabled={!instance && !instanceError}
                className={classes.reloadButton}
              >
                Reload info
              </Button>
            }
          >
            <Tabs
                value={activeTab}
                handleChange={switchTab}
                hasLogs={api.initWS != undefined}
            />
          </SectionTitle>

          {instanceError && (
            <ErrorStub {...instanceError} className={classes.errorStub} />
          )}

          <TabPanel value={activeTab} index={0}>
          {!instanceError && (
            <div className={classes.content}>
              {!instance || !instance?.state.retrieving?.status && <StubSpinner />}

              {instance && (
                <>
                  <Clones />
                  <Info />
                </>
              )}
            </div>
          )}

          <ClonesModal />

          <SnapshotsModal />
          </TabPanel>

          <TabPanel value={activeTab} index={1}>
            <Alert severity="info">
              <AlertTitle>Sensitive data are masked.</AlertTitle>
              You can see the raw log data connecting to the machine and running the <strong>'docker logs'</strong> command.
            </Alert>
            <div id="logs-container">
              {!isLogConnectionEnabled && (
                <div className={classes.spinnerContainer}>
                  <Spinner />
                </div>
              )}
            </div>
          </TabPanel>
        </>

        <TabPanel value={activeTab} index={2}>
          <Configuration 
              isConfigurationActive={isConfigurationActive}
              allowModifyingConfig={instance?.state.engine.allowModifyingConfig}
              switchActiveTab={(id: number) => setActiveTab(id)} 
              activeTab={activeTab} 
              reload={() => stores.main.load(props.instanceId)} 
            />
        </TabPanel>

      </StoresProvider>
    </HostProvider>
  )
})

function TabPanel(props: PropTypes.InferProps<any>) {
  const { children, value, index, ...other } = props;

  return (
      <Typography
          component="div"
          role="tabpanel"
          hidden={value !== index}
          id={`scrollable-auto-tabpanel-${index}`}
          aria-labelledby={`scrollable-auto-tab-${index}`}
          {...other}
      >
        <Box p={3}>{children}</Box>
      </Typography>
  );
}

TabPanel.propTypes = {
  children: PropTypes.node,
  index: PropTypes.any.isRequired,
  value: PropTypes.any.isRequired
};
