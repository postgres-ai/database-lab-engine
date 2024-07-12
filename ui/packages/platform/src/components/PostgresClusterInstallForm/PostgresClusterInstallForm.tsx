/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Box } from '@mui/material'
import { useReducer } from 'react'
import {
  TextField,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  FormControlLabel,
  Checkbox,
} from '@material-ui/core'

import ConsolePageTitle from '../ConsolePageTitle'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DbLabInstanceFormProps } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'
import {
  initialState,
  reducer,
} from 'components/PostgresClusterInstallForm/reducer'
import { PostgresClusterInstallFormSidebar } from 'components/PostgresClusterInstallForm/PostgresClusterInstallFormSidebar'
import { AnsibleInstance } from 'components/PostgresClusterInstallForm/PostgresClusterSteps/AnsibleInstance'
import { DockerInstance } from 'components/PostgresClusterInstallForm/PostgresClusterSteps/DockerInstance'
import { ClusterExtensionAccordion } from 'components/PostgresClusterForm/PostgresClusterSteps'
import { Select } from '@postgres.ai/shared/components/Select'

import { validateDLEName } from 'utils/utils'
import urls from 'utils/urls'
import { isIPAddress } from './utils'
import { icons } from '@postgres.ai/shared/styles/icons'

interface PostgresClusterInstallFormProps extends DbLabInstanceFormProps {
  classes: ClassesType
}

const PostgresClusterInstallForm = (props: PostgresClusterInstallFormProps) => {
  const { classes, orgPermissions } = props
  const [state, dispatch] = useReducer(reducer, initialState)

  const permitted = !orgPermissions || orgPermissions.dblabInstanceCreate

  const pageTitle = <ConsolePageTitle title="Install Postgres Cluster" />
  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      {...props}
      breadcrumbs={[
        { name: 'Postgres Clusters', url: 'pg' },
        { name: 'Install Postgres Cluster' },
      ]}
    />
  )

  const handleReturnToList = () => {
    props.history.push(urls.linkClusters(props))
  }

  const handleSetFormStep = (step: string) => {
    dispatch({ type: 'set_form_step', formStep: step })
  }

  const handleReturnToForm = () => {
    dispatch({ type: 'set_form_step', formStep: initialState.formStep })
  }

  return (
    <div className={classes.root}>
      {breadcrumbs}

      {pageTitle}

      {!permitted && (
        <WarningWrapper>
          You do not have permission to add clusters to this organization.
        </WarningWrapper>
      )}

      <div className={classes.container}>
        {state.formStep === initialState.formStep && permitted ? (
          <>
            <div className={classes.form}>
              <p className={classes.sectionTitle}>1. Provide cluster name</p>
              <TextField
                required
                label="Cluster Name"
                variant="outlined"
                fullWidth
                value={state.patroni_cluster_name}
                className={classes.marginTop}
                InputLabelProps={{
                  shrink: true,
                }}
                helperText={
                  validateDLEName(state.patroni_cluster_name)
                    ? 'Name must be lowercase and contain only letters and numbers.'
                    : ''
                }
                error={validateDLEName(state.patroni_cluster_name)}
                onChange={(
                  event: React.ChangeEvent<
                    HTMLTextAreaElement | HTMLInputElement
                  >,
                ) =>
                  dispatch({
                    type: 'change_patroni_cluster_name',
                    patroni_cluster_name: event.target.value,
                  })
                }
              />
              <p className={classes.sectionTitle}>2. Choose Postgres version</p>
              <Select
                label="Select version"
                items={Array.from({ length: 7 }, (_, i) => i + 10).map(
                  (version) => {
                    return {
                      value: version,
                      children: version,
                    }
                  },
                )}
                value={state.version}
                onChange={(
                  e: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>,
                ) =>
                  dispatch({
                    type: 'change_version',
                    version: e.target.value,
                  })
                }
              />
              <p className={classes.sectionTitle}>3. Data directory</p>
              <p className={classes.instanceParagraph}>
                If you want to place the data on a separate disk, you can
                specify an alternative path to the data directory.
              </p>
              <TextField
                required
                label="PGDATA"
                variant="outlined"
                fullWidth
                value={state.postgresql_data_dir}
                className={classes.marginTop}
                InputLabelProps={{
                  shrink: true,
                }}
                onChange={(
                  event: React.ChangeEvent<
                    HTMLTextAreaElement | HTMLInputElement
                  >,
                ) =>
                  dispatch({
                    type: 'change_postgresql_data_dir',
                    postgresql_data_dir: event.target.value,
                  })
                }
              />
              <ClusterExtensionAccordion
                step={8}
                state={Object.fromEntries(
                  Object.entries(state).filter(
                    ([key]) => key !== 'postgresql_data_dir',
                  ),
                )}
                classes={classes}
                dispatch={dispatch}
              />
              <Accordion className={classes.sectionTitle}>
                <AccordionSummary
                  aria-controls="panel1a-content"
                  id="panel1a-header"
                  expandIcon={icons.sortArrowDown}
                >
                  5. Advanced options
                </AccordionSummary>
                <AccordionDetails>
                  <Box
                    sx={{
                      display: 'flex',
                      flexDirection: 'column',
                      width: '100%',
                      fontWeight: 'normal',
                    }}
                  >
                    <p className={classes.instanceParagraph}>
                      Optional. Specify here the IP address that will be used as
                      a single entry point for client access to databases in the
                      cluster (not for cloud environments).
                    </p>
                    <TextField
                      label="Cluster VIP address"
                      variant="outlined"
                      fullWidth
                      error={!isIPAddress(state.cluster_vip)}
                      helperText={
                        isIPAddress(state.cluster_vip)
                          ? ''
                          : 'IP address is invalid'
                      }
                      value={state.cluster_vip}
                      className={classes.marginTop}
                      InputLabelProps={{
                        shrink: true,
                      }}
                      onChange={(
                        event: React.ChangeEvent<
                          HTMLTextAreaElement | HTMLInputElement
                        >,
                      ) =>
                        dispatch({
                          type: 'change_cluster_vip',
                          cluster_vip: event.target.value,
                        })
                      }
                    />
                    <FormControlLabel
                      className={classes.marginTop}
                      control={
                        <Checkbox
                          name="with_haproxy_load_balancing"
                          checked={state.with_haproxy_load_balancing}
                          onChange={(e) =>
                            dispatch({
                              type: 'change_with_haproxy_load_balancing',
                              with_haproxy_load_balancing: e.target.checked,
                            })
                          }
                          classes={{
                            root: classes.checkboxRoot,
                          }}
                        />
                      }
                      label={'Haproxy load balancing'}
                    />
                    <FormControlLabel
                      control={
                        <Checkbox
                          name="pgbouncer_install"
                          checked={state.pgbouncer_install}
                          onChange={(e) =>
                            dispatch({
                              type: 'change_pgbouncer_install',
                              pgbouncer_install: e.target.checked,
                            })
                          }
                          classes={{
                            root: classes.checkboxRoot,
                          }}
                        />
                      }
                      label={'PgBouncer connection pooler'}
                    />
                    <FormControlLabel
                      control={
                        <Checkbox
                          name="synchronous_mode"
                          checked={state.synchronous_mode}
                          onChange={(e) =>
                            dispatch({
                              type: 'change_synchronous_mode',
                              synchronous_mode: e.target.checked,
                            })
                          }
                          classes={{
                            root: classes.checkboxRoot,
                          }}
                        />
                      }
                      label={'Enable synchronous replication'}
                    />
                    <TextField
                      label="Number of synchronous standbys"
                      variant="outlined"
                      fullWidth
                      type="number"
                      InputProps={{
                        inputProps: {
                          min: 1,
                        },
                      }}
                      value={state.synchronous_node_count}
                      disabled={!state.synchronous_mode}
                      className={classes.marginTop}
                      InputLabelProps={{
                        shrink: true,
                      }}
                      onChange={(
                        e: React.ChangeEvent<
                          HTMLTextAreaElement | HTMLInputElement
                        >,
                      ) =>
                        dispatch({
                          type: 'change_synchronous_node_count',
                          synchronous_node_count: e.target.value,
                        })
                      }
                    />
                    <FormControlLabel
                      className={classes.marginTop}
                      control={
                        <Checkbox
                          name="netdata_install"
                          checked={state.netdata_install}
                          onChange={(e) =>
                            dispatch({
                              type: 'change_netdata_install',
                              netdata_install: e.target.checked,
                            })
                          }
                          classes={{
                            root: classes.checkboxRoot,
                          }}
                        />
                      }
                      label={'Netdata monitoring'}
                    />
                  </Box>
                </AccordionDetails>
              </Accordion>
            </div>
            <PostgresClusterInstallFormSidebar
              state={state}
              disabled={
                validateDLEName(state.patroni_cluster_name) ||
                !isIPAddress(state.cluster_vip)
              }
              handleCreate={() =>
                !validateDLEName(state.patroni_cluster_name) &&
                handleSetFormStep('docker')
              }
            />
          </>
        ) : state.formStep === 'ansible' && permitted ? (
          <AnsibleInstance
            state={state}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBack={handleReturnToList}
            goBackToForm={handleReturnToForm}
          />
        ) : state.formStep === 'docker' && permitted ? (
          <DockerInstance
            state={state}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBack={handleReturnToList}
            goBackToForm={handleReturnToForm}
          />
        ) : null}
      </div>
    </div>
  )
}

export default PostgresClusterInstallForm
