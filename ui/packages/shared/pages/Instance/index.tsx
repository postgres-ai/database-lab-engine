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

import { Tabs } from './Tabs'
import { Clones } from './Clones'
import { Info } from './Info'
import { establishConnection } from './wsLogs'
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
  }
}))

export const Instance = observer((props: Props) => {
  const classes = useStyles()

  const { instanceId, api } = props

  const stores = useCreatedStores(props)

  useEffect(() => {
    stores.main.load(instanceId)
  }, [instanceId])

  const { instance, instanceError } = stores.main

  useEffect(() => {
    if (instance && !instance?.state?.pools) {
      if (!props.callbacks) return

      props.callbacks.showDeprecatedApiBanner()
      return props.callbacks?.hideDeprecatedApiBanner
    }
  }, [instance])

  const [activeTab, setActiveTab] = React.useState(0);

  const [isLogConnectionEnabled, enableLogConnection] = React.useState(false);

  const switchTab = (event: React.ChangeEvent<{}>, tabID: number) => {
    if (tabID == 1 && api.initWS != undefined && !isLogConnectionEnabled) {
      establishConnection(api);
      enableLogConnection(true)
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
              {!instance && <StubSpinner />}

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
            Instance
          </TabPanel>

          <TabPanel value={activeTab} index={1}>
            <Alert severity="info">
              <AlertTitle>Sensitive data are masked.</AlertTitle>
              You can see the raw log data connecting to the machine and running the <strong>'docker logs'</strong> command.
            </Alert>
            <div id="logs-container"></div>
          </TabPanel>
        </>

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
