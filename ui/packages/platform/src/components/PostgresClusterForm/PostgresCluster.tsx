/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import cn from 'classnames'
import { useEffect, useReducer } from 'react'
import { Box } from '@mui/material'
import {
  Tab,
  Tabs,
  TextField,
  MenuItem,
  InputAdornment,
  FormControlLabel,
  Checkbox,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@material-ui/core'

import ConsolePageTitle from '../ConsolePageTitle'
import { TabPanel } from 'pages/JoeSessionCommand/TabPanel'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DbLabInstanceFormProps } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'
import { StorageSlider } from 'components/DbLabInstanceForm/DbLabInstanceFormSlider'
import { CloudProvider, getCloudProviders } from 'api/cloud/getCloudProviders'
import { CloudVolumes, getCloudVolumes } from 'api/cloud/getCloudVolumes'
import { initialState, reducer } from 'components/PostgresClusterForm/reducer'
import { initialState as dbLabInitialState } from 'components/DbLabInstanceForm/reducer'
import { DbLabInstanceFormSidebar } from 'components/DbLabInstanceForm/DbLabInstanceFormSidebar'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinnerFlex'
import { Select } from '@postgres.ai/shared/components/Select'

import { validateDLEName } from 'utils/utils'
import urls from 'utils/urls'

import { AnsibleInstance } from 'components/DbLabInstanceForm/DbLabFormSteps/AnsibleInstance'
import { CloudRegion, getCloudRegions } from 'api/cloud/getCloudRegions'
import { CloudInstance, getCloudInstances } from 'api/cloud/getCloudInstances'
import { DockerInstance } from 'components/DbLabInstanceForm/DbLabFormSteps/DockerInstance'
import { SimpleInstance } from 'components/DbLabInstanceForm/DbLabFormSteps/SimpleInstance'
import { icons } from '@postgres.ai/shared/styles/icons'

interface PostgresClusterProps extends DbLabInstanceFormProps {
  classes: ClassesType
  auth?: {
    userId: number
  }
}

const PostgresCluster = (props: PostgresClusterProps) => {
  const { classes, orgPermissions } = props
  const [state, dispatch] = useReducer(reducer, initialState)

  const permitted = !orgPermissions || orgPermissions.dblabInstanceCreate
  const urlParams = new URLSearchParams(window.location.search)
  const urlTaskID = urlParams.get('taskID')
  const urlProvider = urlParams.get('provider')
  const requirePublicKeys =
    !state.publicKeys && (state.provider === 'aws' || state.provider === 'gcp')

  useEffect(() => {
    if (urlTaskID && urlProvider) {
      dispatch({
        type: 'set_form_step',
        formStep: 'simple',
        taskID: urlTaskID,
        provider: urlProvider,
      })
    } else {
      dispatch({
        type: 'set_form_step',
        formStep: initialState.formStep,
        provider: initialState.provider,
      })
    }
  }, [urlTaskID, urlProvider])

  useEffect(() => {
    const fetchCloudDetails = async () => {
      dispatch({ type: 'set_is_loading', isLoading: true })
      try {
        const cloudRegions = await getCloudRegions(initialState.provider)
        const cloudVolumes = await getCloudVolumes(initialState.provider)
        const serviceProviders = await getCloudProviders()
        const ssdCloudVolumes = cloudVolumes.response.filter(
          (volume: CloudVolumes) => volume.api_name === initialState?.api_name,
        )[0]

        dispatch({
          type: 'set_initial_state',
          cloudRegions: cloudRegions.response,
          volumes: cloudVolumes.response,
          volumeType: `${ssdCloudVolumes.api_name} (${ssdCloudVolumes.cloud_provider}: ${ssdCloudVolumes.native_name})`,
          volumeCurrency: ssdCloudVolumes.native_reference_price_currency,
          volumePricePerHour:
            ssdCloudVolumes.native_reference_price_per_1000gib_per_hour,
          volumePrice:
            (initialState.storage *
              ssdCloudVolumes.native_reference_price_per_1000gib_per_hour) /
            1000,
          serviceProviders: serviceProviders.response,
          isLoading: false,
        })
      } catch (error) {
        console.log(error)
      }
    }
    fetchCloudDetails()
  }, [])

  useEffect(() => {
    const fetchUpdatedDetails = async () => {
      try {
        const cloudRegions = await getCloudRegions(state.provider)
        const cloudVolumes = await getCloudVolumes(state.provider)
        const ssdCloudVolumes = cloudVolumes.response.filter(
          (volume: CloudVolumes) => volume.api_name === initialState?.api_name,
        )[0]

        dispatch({
          type: 'update_initial_state',
          volumes: cloudVolumes.response,
          volumeType: `${ssdCloudVolumes.api_name} (${ssdCloudVolumes.cloud_provider}: ${ssdCloudVolumes.native_name})`,
          volumeCurrency: ssdCloudVolumes.native_reference_price_currency,
          volumePricePerHour:
            ssdCloudVolumes.native_reference_price_per_1000gib_per_hour,
          volumePrice:
            (initialState.storage *
              ssdCloudVolumes.native_reference_price_per_1000gib_per_hour) /
            1000,
          cloudRegions: cloudRegions.response,
        })
      } catch (error) {
        console.log(error)
      }
    }
    fetchUpdatedDetails()
  }, [state.api_name, state.provider])

  useEffect(() => {
    const fetchUpdatedDetails = async () => {
      dispatch({ type: 'set_is_reloading', isReloading: true })
      try {
        const cloudInstances = await getCloudInstances({
          provider: state.provider,
          region: state.location.native_code,
        })

        dispatch({
          type: 'update_instance_type',
          cloudInstances: cloudInstances.response,
          instanceType: cloudInstances.response[0],
          isReloading: false,
        })
      } catch (error) {
        console.log(error)
      }
    }
    fetchUpdatedDetails()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.location.native_code])

  const uniqueRegionsByProvider = state.cloudRegions
    .map((region: CloudRegion) => region.world_part)
    .filter(
      (value: string, index: number, self: string) =>
        self.indexOf(value) === index,
    )

  const filteredRegions = state.cloudRegions.filter(
    (region: CloudRegion) => region.world_part === state.region,
  )

  const pageTitle = <ConsolePageTitle title="Create Postgres Cluster" />
  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      {...props}
      breadcrumbs={[
        { name: 'Postgres Clusters', url: 'pg' },
        { name: 'Create Postgres Cluster' },
      ]}
    />
  )

  const handleChangeVolume = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const volumeApiName = event.target.value.split(' ')[0]
    const selectedVolume = state.volumes.filter(
      (volume: CloudVolumes) => volume.api_name === volumeApiName,
    )[0]

    dispatch({
      type: 'change_volume_type',
      volumeType: event.target.value,
      volumePricePerHour:
        selectedVolume.native_reference_price_per_1000gib_per_hour,
      volumePrice:
        (state.storage *
          selectedVolume.native_reference_price_per_1000gib_per_hour) /
        1000,
    })
  }

  const handleSetFormStep = (step: string) => {
    dispatch({ type: 'set_form_step', formStep: step })
  }

  const handleReturnToList = () => {
    props.history.push(urls.linkClusters(props))
  }

  const handleReturnToForm = () => {
    dispatch({ type: 'set_form_step', formStep: initialState.formStep })
  }

  const checkSyncStandbyCount = () => {
    if (state.synchronous_mode) {
      if (Number(state.numberOfInstances) === 1) {
        return state.synchronous_node_count > state.numberOfInstances
      } else {
        return state.synchronous_node_count > state.numberOfInstances - 1
      }
    }
  }

  const disableSubmitButton =
    validateDLEName(state.name) ||
    requirePublicKeys ||
    state.numberOfInstances > 32 ||
    checkSyncStandbyCount() ||
    (state.publicKeys && state.publicKeys.length < 30)

  if (state.isLoading) return <StubSpinner />

  return (
    <div className={classes.root}>
      {breadcrumbs}

      {pageTitle}

      {!permitted && (
        <WarningWrapper>
          You do not have permission to add Database Lab instances.
        </WarningWrapper>
      )}

      <div
        className={cn(
          classes.container,
          state.isReloading && classes.backgroundOverlay,
        )}
      >
        {state.formStep === initialState.formStep && permitted ? (
          <>
            {state.isReloading && (
              <Spinner className={classes.absoluteSpinner} />
            )}
            <div className={classes.form}>
              <p className={classes.sectionTitle}>
                1. Select your cloud provider
              </p>
              <div className={classes.providerFlex}>
                {state.serviceProviders.map(
                  (provider: CloudProvider, index: number) => (
                    <div
                      className={cn(
                        classes.provider,
                        state.provider === provider.api_name &&
                          classes.activeBorder,
                      )}
                      key={index}
                      onClick={() =>
                        dispatch({
                          type: 'change_provider',
                          provider: provider.api_name,
                        })
                      }
                    >
                      <img
                        src={`/images/service-providers/${provider.api_name}.png`}
                        width={85}
                        height="auto"
                        alt={provider.label}
                      />
                    </div>
                  ),
                )}
              </div>
              <p className={classes.sectionTitle}>
                2. Select your cloud region
              </p>
              <div className={classes.sectionContainer}>
                <Tabs
                  value={state.region}
                  onChange={(_: React.ChangeEvent<{}> | null, value: string) =>
                    dispatch({
                      type: 'change_region',
                      region: value,
                      location: state.cloudRegions.find(
                        (region: CloudRegion) =>
                          region.world_part === value &&
                          region.cloud_provider === state.provider,
                      ),
                    })
                  }
                >
                  {uniqueRegionsByProvider.map(
                    (region: string, index: number) => (
                      <Tab
                        key={index}
                        label={region}
                        value={region}
                        className={classes.tab}
                      />
                    ),
                  )}
                </Tabs>
              </div>
              <TabPanel value={state.region} index={state.region}>
                {filteredRegions.map((region: CloudRegion, index: number) => (
                  <div
                    key={index}
                    className={cn(
                      classes.serviceLocation,
                      state.location?.api_name === region?.api_name &&
                        classes.activeBorder,
                    )}
                    onClick={() =>
                      dispatch({
                        type: 'change_location',
                        location: region,
                      })
                    }
                  >
                    <p className={classes.serviceTitle}>{region.api_name}</p>
                    <p className={classes.serviceTitle}>🏴 {region.label}</p>
                  </div>
                ))}
              </TabPanel>
              {state.instanceType ? (
                <>
                  <p className={classes.sectionTitle}>
                    3. Choose instance type
                  </p>
                  <TabPanel
                    value={state.cloudInstances}
                    index={state.cloudInstances}
                  >
                    {state.cloudInstances.map(
                      (instance: CloudInstance, index: number) => (
                        <div
                          key={index}
                          className={cn(
                            classes.instanceSize,
                            state.instanceType === instance &&
                              classes.activeBorder,
                          )}
                          onClick={() =>
                            dispatch({
                              type: 'change_instance_type',
                              instanceType: instance,
                            })
                          }
                        >
                          <p>
                            {instance.api_name} (
                            {state.instanceType.cloud_provider}:{' '}
                            {instance.native_name})
                          </p>
                          <div>
                            <span>🔳 {instance.native_vcpus} CPU</span>
                            <span>🧠 {instance.native_ram_gib} GiB RAM</span>
                          </div>
                        </div>
                      ),
                    )}
                  </TabPanel>
                  <p className={classes.sectionTitle}>4. Number of instances</p>
                  <p className={classes.instanceParagraph}>
                    Number of servers in the Postgres cluster
                  </p>
                  <Box className={classes.sliderContainer}>
                    <Box className={classes.sliderInputContainer}>
                      <Box className={classes.databaseSize}>
                        <TextField
                          variant="outlined"
                          fullWidth
                          type="number"
                          label="Number of instances"
                          InputLabelProps={{
                            shrink: true,
                          }}
                          helperText={
                            state.numberOfInstances > 32 &&
                            'Maximum 32 instances'
                          }
                          error={state.numberOfInstances > 32}
                          value={state.numberOfInstances}
                          className={classes.filterSelect}
                          onChange={(
                            event: React.ChangeEvent<
                              HTMLTextAreaElement | HTMLInputElement
                            >,
                          ) => {
                            dispatch({
                              type: 'change_number_of_instances',
                              number: event.target.value,
                            })
                          }}
                        />
                      </Box>
                    </Box>
                    <StorageSlider
                      value={state.numberOfInstances}
                      sliderOptions={{
                        min: 1,
                        max: 32,
                        step: 1,
                      }}
                      customMarks={[
                        {
                          value: 1,
                          label: '1',
                          scaledValue: 1,
                        },
                        {
                          value: 5,
                          label: '5',
                          scaledValue: 5,
                        },
                        {
                          value: 9,
                          label: '9',
                          scaledValue: 9,
                        },
                        {
                          value: 13,
                          label: '13',
                          scaledValue: 13,
                        },
                        {
                          value: 17,
                          label: '17',
                          scaledValue: 17,
                        },
                        {
                          value: 21,
                          label: '21',
                          scaledValue: 21,
                        },
                        {
                          value: 25,
                          label: '25',
                          scaledValue: 25,
                        },
                        {
                          value: 29,
                          label: '29',
                          scaledValue: 29,
                        },
                        {
                          value: 32,
                          label: '32',
                          scaledValue: 32,
                        },
                      ]}
                      onChange={(_: React.ChangeEvent<{}>, value: unknown) => {
                        dispatch({
                          type: 'change_number_of_instances',
                          number: value,
                        })
                      }}
                    />
                  </Box>
                  <p className={classes.sectionTitle}>5. Database volume</p>
                  <Box className={classes.sliderContainer}>
                    <Box className={classes.sliderInputContainer}>
                      <Box className={classes.sliderVolume}>
                        <TextField
                          value={state.volumeType}
                          onChange={handleChangeVolume}
                          select
                          label="Volume type"
                          InputLabelProps={{
                            shrink: true,
                          }}
                          variant="outlined"
                          className={classes.filterSelect}
                        >
                          {(state.volumes as CloudVolumes[]).map((p, id) => {
                            const volumeName = `${p.api_name} (${p.cloud_provider}: ${p.native_name})`
                            return (
                              <MenuItem value={volumeName} key={id}>
                                {volumeName}
                              </MenuItem>
                            )
                          })}
                        </TextField>
                      </Box>
                      <Box className={classes.databaseSize}>
                        <TextField
                          variant="outlined"
                          fullWidth
                          type="number"
                          label="Volume size"
                          InputLabelProps={{
                            shrink: true,
                          }}
                          InputProps={{
                            inputProps: {
                              min: 0,
                            },
                            endAdornment: (
                              <InputAdornment position="end">
                                GiB
                              </InputAdornment>
                            ),
                          }}
                          value={state.storage}
                          className={classes.filterSelect}
                          onChange={(
                            event: React.ChangeEvent<
                              HTMLTextAreaElement | HTMLInputElement
                            >,
                          ) => {
                            dispatch({
                              type: 'change_volume_price',
                              volumeSize: event.target.value,
                              volumePrice: event.target.value,
                            })
                          }}
                        />
                      </Box>
                    </Box>
                    <StorageSlider
                      value={state.storage}
                      sliderOptions={{
                        min: 0,
                        max: 10000,
                        step: 10,
                      }}
                      customMarks={[
                        {
                          value: 0,
                          label: '0',
                          scaledValue: 0,
                        },
                        {
                          value: 1000,
                          label: '1000 GiB',
                          scaledValue: 1000,
                        },
                        {
                          value: 2000,
                          label: '2000 GiB',
                          scaledValue: 2000,
                        },
                        {
                          value: 3000,
                          label: '3000 GiB',
                          scaledValue: 3000,
                        },
                        {
                          value: 4000,
                          label: '4000 GiB',
                          scaledValue: 4000,
                        },
                        {
                          value: 5000,
                          label: '5000 GiB',
                          scaledValue: 5000,
                        },
                        {
                          value: 6000,
                          label: '6000 GiB',
                          scaledValue: 6000,
                        },
                        {
                          value: 7000,
                          label: '7000 GiB',
                          scaledValue: 7000,
                        },
                        {
                          value: 8000,
                          label: '8000 GiB',
                          scaledValue: 8000,
                        },
                        {
                          value: 9000,
                          label: '9000 GiB',
                          scaledValue: 9000,
                        },
                        {
                          value: 10000,
                          label: '10 TB',
                          scaledValue: 10000,
                        },
                      ]}
                      onChange={(_: React.ChangeEvent<{}>, value: unknown) => {
                        dispatch({
                          type: 'change_volume_price',
                          volumeSize: value,
                          volumePrice:
                            (Number(value) * state.volumePricePerHour) / 1000,
                        })
                      }}
                    />
                  </Box>
                  <p className={classes.sectionTitle}>
                    6. Provide cluster name
                  </p>
                  <TextField
                    required
                    label="Cluster Name"
                    variant="outlined"
                    fullWidth
                    value={state.name}
                    className={classes.marginTop}
                    InputLabelProps={{
                      shrink: true,
                    }}
                    helperText={
                      validateDLEName(state.name)
                        ? 'Name must be lowercase and contain only letters and numbers.'
                        : ''
                    }
                    error={validateDLEName(state.name)}
                    onChange={(
                      event: React.ChangeEvent<
                        HTMLTextAreaElement | HTMLInputElement
                      >,
                    ) =>
                      dispatch({
                        type: 'change_name',
                        name: event.target.value,
                      })
                    }
                  />
                  <p className={classes.sectionTitle}>
                    7. Choose Postgres version
                  </p>
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
                      e: React.ChangeEvent<
                        HTMLTextAreaElement | HTMLInputElement
                      >,
                    ) =>
                      dispatch({
                        type: 'set_version',
                        version: e.target.value,
                      })
                    }
                  />
                  <p className={classes.sectionTitle}>
                    8. Provide SSH public keys (one per line)
                  </p>
                  <p className={classes.instanceParagraph}>
                    These SSH public keys will be added to the DBLab server's
                    &nbsp;
                    <code className={classes.code}>~/.ssh/authorized_keys</code>
                    &nbsp; file. Providing at least one public key is
                    recommended to ensure access to the server after deployment.
                  </p>
                  <TextField
                    label="SSH public keys"
                    variant="outlined"
                    fullWidth
                    multiline
                    helperText={
                      state.publicKeys && state.publicKeys.length < 30
                        ? 'Public key is too short'
                        : ''
                    }
                    error={state.publicKeys && state.publicKeys.length < 30}
                    value={state.publicKeys}
                    required={requirePublicKeys}
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
                        type: 'change_public_keys',
                        publicKeys: event.target.value,
                      })
                    }
                  />
                  <Accordion className={classes.sectionTitle}>
                    <AccordionSummary
                      aria-controls="panel1a-content"
                      id="panel1a-header"
                      expandIcon={icons.sortArrowDown}
                    >
                      9. Advanced options
                    </AccordionSummary>
                    <AccordionDetails>
                      <Box
                        sx={{
                          display: 'flex',
                          flexDirection: 'column',
                        }}
                      >
                        <FormControlLabel
                          control={
                            <Checkbox
                              name="database_public_access"
                              checked={state.database_public_access}
                              onChange={(e) =>
                                dispatch({
                                  type: 'change_database_public_access',
                                  database_public_access: e.target.checked,
                                })
                              }
                              classes={{
                                root: classes.checkboxRoot,
                              }}
                            />
                          }
                          label={'Database public access'}
                        />
                        <FormControlLabel
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
                          helperText={
                            checkSyncStandbyCount() &&
                            `Maximum ${
                              Number(state.numberOfInstances) === 1
                                ? state.numberOfInstances
                                : state.numberOfInstances - 1
                            } synchronous standbys`
                          }
                          error={checkSyncStandbyCount()}
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
                </>
              ) : (
                <p className={classes.errorMessage}>
                  No instance types available for this cloud region. Please try
                  another region.
                </p>
              )}
            </div>
            <DbLabInstanceFormSidebar
              cluster
              state={state as unknown as typeof dbLabInitialState}
              disabled={disableSubmitButton}
              handleCreate={() => handleSetFormStep('simple')}
            />
          </>
        ) : state.formStep === 'ansible' && permitted ? (
          <AnsibleInstance
            cluster
            state={state as unknown as typeof dbLabInitialState}
            orgId={props.orgId}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBack={handleReturnToList}
            goBackToForm={handleReturnToForm}
          />
        ) : state.formStep === 'docker' && permitted ? (
          <DockerInstance
            cluster
            state={state as unknown as typeof dbLabInitialState}
            orgId={props.orgId}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBack={handleReturnToList}
            goBackToForm={handleReturnToForm}
          />
        ) : state.formStep === 'simple' && permitted ? (
          <SimpleInstance
            cluster
            state={state as unknown as typeof dbLabInitialState}
            userID={props.auth?.userId}
            orgId={props.orgId}
            formStep={state.formStep}
            setFormStep={handleSetFormStep}
            goBackToForm={() => {
              window.history.pushState({}, '', `${window.location.pathname}`)
              handleReturnToForm()
            }}
          />
        ) : null}
      </div>
    </div>
  )
}

export default PostgresCluster
