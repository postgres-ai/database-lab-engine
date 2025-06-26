/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any small is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState, useEffect, useMemo } from 'react'
import { observer } from 'mobx-react-lite'
import Editor from '@monaco-editor/react'
import {
  Checkbox,
  FormControlLabel,
  Typography,
  Snackbar,
  makeStyles,
  Button,
} from '@material-ui/core'
import Box from '@mui/material/Box'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { ExternalIcon } from '@postgres.ai/shared/icons/External'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { MainStore } from '@postgres.ai/shared/pages/Instance/stores/Main'

import { tooltipText } from './tooltipText'
import { FormValues, useForm } from './useForm'
import { ResponseMessage } from './ResponseMessage'
import { ConfigSectionTitle, Header, ModalTitle } from './Header'
import { dockerImageOptions, imagePgOptions } from './configOptions'
import {
  FormValuesKey,
  uniqueChipValue,
  customOrGenericImage,
  genericDockerImages,
  getImageMajorVersion,
  createFallbackDockerImage,
  createEnhancedDockerImages,
} from './utils'
import {
  SelectWithTooltip,
  InputWithChip,
  InputWithTooltip,
} from './InputWithTooltip'

import styles from './styles.module.scss'
import { SeImages } from '@postgres.ai/shared/types/api/endpoints/getSeImages'
import {
  formatTuningParams,
  formatTuningParamsToObj,
} from '@postgres.ai/shared/types/api/endpoints/testDbSource'

type PgOptionsType = {
  optionType: string
  pgDumpOptions: string[]
  pgRestoreOptions: string[]
}

const NON_LOGICAL_RETRIEVAL_MESSAGE =
  'Configuration editing is only available in logical mode'
const PREVENT_MODIFYING_MESSAGE = 'Editing is disabled by admin'

const useStyles = makeStyles(
  {
    checkboxRoot: {
      padding: '9px 10px',
    },
    grayText: {
      color: '#8a8a8a',
      fontSize: '12px',
    },
  },
  { index: 1 },
)

export const Configuration = observer(
  ({
    instanceId,
    switchActiveTab,
    reload,
    isConfigurationActive,
    disableConfigModification,
  }: {
    instanceId: string
    switchActiveTab: (_: null, activeTab: number) => void
    reload: () => void
    isConfigurationActive: boolean
    disableConfigModification?: boolean
  }) => {
    const classes = useStyles()
    const stores = useStores()
    const {
      config,
      isConfigurationLoading,
      updateConfig,
      getSeImages,
      fullConfig,
      testDbSource,
      configError,
      getFullConfig,
      getFullConfigError,
      getEngine,
    } = stores.main

    const configData: MainStore['config'] =
      config && JSON.parse(JSON.stringify(config))
    const isConfigurationDisabled =
      !isConfigurationActive || disableConfigModification

    const [dleEdition, setDledition] = useState('')
    const isCeEdition = dleEdition === 'community'
    const filteredDockerImageOptions = isCeEdition
      ? dockerImageOptions.filter(
          (option) =>
            option.type === 'custom' || option.type === 'Generic Postgres',
        )
      : dockerImageOptions

    const [isModalOpen, setIsModalOpen] = useState(false)
    const [submitState, setSubmitState] = useState({
      status: '',
      response: '' as string | React.ReactNode,
    })
    const [dockerState, setDockerState] = useState({
      loading: false,
      error: '',
      tags: [] as string[],
      locations: [] as string[],
      images: [] as string[],
      preloadLibraries: '' as string | undefined,
      data: [] as SeImages[],
    })
    const [testConnectionState, setTestConnectionState] = useState({
      default: {
        loading: false,
        error: '',
        message: {
          status: '',
          message: '',
        },
      },
      dockerImage: {
        loading: false,
        error: '',
        message: {
          status: '',
          message: '',
        },
      },
      fetchTuning: {
        loading: false,
        error: '',
        message: {
          status: '',
          message: '',
        },
      },
    })

    const switchTab = async () => {
      reload()
      switchActiveTab(null, 0)
    }

    const onSubmit = async (values: FormValues) => {
      setSubmitState({
        ...submitState,
        response: '',
      })
      await updateConfig(
        {
          ...values,
          tuningParams: formatTuningParamsToObj(
            values.tuningParams,
          ) as unknown as string,
        },
        instanceId,
      ).then((response) => {
        if (response?.ok) {
          setSubmitState({
            status: 'success',
            response: (
              <p>
                Changes applied.{' '}
                <span className={styles.underline} onClick={switchTab}>
                  Switch to Overview
                </span>{' '}
                to see details and to work with clones
              </p>
            ),
          })
        }
      })
    }
    const [{ formik, connectionData, isConnectionDataValid }] =
      useForm(onSubmit)

    // Memoized enhanced Docker images to avoid recreation on every render
    // This combines predefined images with any custom image from configuration
    const enhancedDockerImages = useMemo(() => {
      return createEnhancedDockerImages(
        configData?.dockerImageType === 'Generic Postgres' ? configData?.dockerPath : undefined,
        configData?.dockerImageType === 'Generic Postgres' ? configData?.dockerTag : undefined
      )
    }, [configData?.dockerPath, configData?.dockerTag, configData?.dockerImageType])

    // Memoized computed values from enhanced images
    const dockerImageVersions = useMemo(() => {
      return enhancedDockerImages
        .map((image) => image.pg_major_version)
        .filter((value, index, self) => self.indexOf(value) === index)
        .sort((a, b) => Number(a) - Number(b))
    }, [enhancedDockerImages])

    // Memoized tags and locations for performance
    const dockerTags = useMemo(() => {
      return enhancedDockerImages.map((image) => image.tag)
    }, [enhancedDockerImages])

    const dockerLocations = useMemo(() => {
      return enhancedDockerImages.map((image) => image.location)
    }, [enhancedDockerImages])

    const scrollToField = () => {
      const errorElement = document.querySelector('.Mui-error')
      if (errorElement) {
        errorElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
        const inputElement = errorElement.querySelector('input')
        if (inputElement) {
          setTimeout(() => {
            inputElement.focus()
          }, 1000)
        }
      }
    }

    const onTestConnectionClick = async ({
      type,
    }: {
      type: 'default' | 'dockerImage' | 'fetchTuning'
    }) => {
      Object.keys(connectionData).map(function (key: string) {
        if (key !== 'password' && key !== 'db_list') {
          formik.validateField(key).then(() => {
            scrollToField()
          })
        }
      })
      if (isConnectionDataValid) {
        setTestConnectionState({
          ...testConnectionState,
          [type]: {
            ...testConnectionState[type as keyof typeof testConnectionState],
            loading: true,
            error: '',
            message: {
              status: '',
              message: '',
            },
          },
        })
        testDbSource({
          ...connectionData,
          instanceId,
        })
          .then((res) => {
            if (res?.response) {
              setTestConnectionState({
                ...testConnectionState,
                [type]: {
                  ...testConnectionState[
                    type as keyof typeof testConnectionState
                  ],
                  message: {
                    status: res.response.status,
                    message: res.response.message,
                  },
                },
              })

              if (type === 'fetchTuning') {
                formik.setFieldValue(
                  'tuningParams',
                  formatTuningParams(res.response.tuningParams),
                )
              }

              if (type === 'dockerImage' && res.response?.dbVersion) {
                const currentDockerImage = dockerState.data.find(
                  (image) =>
                    Number(image.pg_major_version) === res.response?.dbVersion,
                )

                if (currentDockerImage) {
                  formik.setValues({
                    ...formik.values,
                    dockerImage: currentDockerImage.pg_major_version,
                    dockerPath: currentDockerImage.location,
                    dockerTag: currentDockerImage.tag,
                  })

                  setDockerState({
                    ...dockerState,
                    tags: dockerState.data
                      .map((image) => image.tag)
                      .filter((tag) =>
                        tag.startsWith(currentDockerImage.pg_major_version),
                      ),
                  })
                }
              }
            } else if (res?.error) {
              setTestConnectionState({
                ...testConnectionState,
                [type]: {
                  ...testConnectionState[
                    type as keyof typeof testConnectionState
                  ],
                  message: {
                    status: 'error',
                    message: res.error.message,
                  },
                },
              })
            }
          })
          .catch((err) => {
            setTestConnectionState({
              ...testConnectionState,
              [type]: {
                ...testConnectionState[
                  type as keyof typeof testConnectionState
                ],
                error: err.message,
                loading: false,
              },
            })
          })
      }
    }

    const handleModalClick = async () => {
      await getFullConfig(instanceId)
      setIsModalOpen(true)
    }

    const handleDeleteChip = (
      _: React.FormEvent<HTMLInputElement>,
      uniqueValue: string,
      id: string,
    ) => {
      if (formik.values[id as FormValuesKey]) {
        let newValues = ''
        const currentValues = uniqueChipValue(
          String(formik.values[id as FormValuesKey]),
        )
        const splitValues = currentValues.split(' ')
        const curDividers = String(formik.values[id as FormValuesKey]).match(
          /[,(\s)(\n)(\r)(\t)(\r\n)]/gm,
        )
        for (let i in splitValues) {
          if (curDividers && splitValues[i] !== uniqueValue) {
            newValues =
              newValues +
              splitValues[i] +
              (curDividers[i] ? curDividers[i] : '')
          }
        }
        formik.setFieldValue(id, newValues)
      }
    }

    const handleSelectPgOptions = (
      e: React.ChangeEvent<HTMLInputElement>,
      formikName: string,
    ) => {
      let pgValue = formik.values[formikName as FormValuesKey]
      formik.setFieldValue(
        formikName,
        configData && configData[formikName as FormValuesKey],
      )
      const selectedPgOptions = imagePgOptions.filter(
        (pg) => e.target.value === pg.optionType,
      )

      const setFormikPgValue = (name: string) => {
        if (selectedPgOptions.length === 0) {
          formik.setFieldValue(formikName, '')
        }

        selectedPgOptions.forEach((pg: PgOptionsType) => {
          return (pg[name as keyof PgOptionsType] as string[]).forEach(
            (addOption) => {
              if (!String(pgValue)?.includes(addOption)) {
                const addOptionWithSpace = addOption + ' '
                formik.setFieldValue(
                  formikName,
                  (pgValue += addOptionWithSpace),
                )
              }
            },
          )
        })
      }

      if (formikName === 'pgRestoreCustomOptions') {
        setFormikPgValue('pgRestoreOptions')
      } else {
        setFormikPgValue('pgDumpOptions')
      }
    }

    const fetchSeImages = async ({
      dockerTag,
      packageGroup,
      initialRender,
    }: {
      dockerTag?: string
      packageGroup: string
      initialRender?: boolean
    }) => {
      setDockerState({
        ...dockerState,
        loading: true,
      })
      await getSeImages({
        packageGroup,
      }).then((data) => {
        if (data) {
          const seImagesMajorVersions = data
            .map((image) => image.pg_major_version)
            .filter((value, index, self) => self.indexOf(value) === index)
            .sort((a, b) => Number(a) - Number(b))
          const currentDockerImage = initialRender
            ? formik.values.dockerImage
            : seImagesMajorVersions.slice(-1)[0]

          const currentPreloadLibraries =
            data.find((image) => image.tag === dockerTag)?.pg_config_presets
              ?.shared_preload_libraries ||
            data[0]?.pg_config_presets?.shared_preload_libraries

          setDockerState({
            ...(initialRender
              ? { images: seImagesMajorVersions }
              : {
                  ...dockerState,
                }),
            error: '',
            tags: data
              .map((image) => image.tag)
              .filter((tag) => tag.startsWith(currentDockerImage)),
            locations: data
              .map((image) => image.location)
              .filter((location) => location?.includes(currentDockerImage)),
            loading: false,
            preloadLibraries: currentPreloadLibraries,
            images: seImagesMajorVersions,
            data,
          })

          formik.setValues({
            ...formik.values,
            dockerImage: currentDockerImage,
            dockerImageType: packageGroup,
            dockerTag: dockerTag
              ? dockerTag
              : data.map((image) => image.tag)[0],
            dockerPath: initialRender
              ? formik.values.dockerPath
              : data.map((image) => image.location)[0],
            sharedPreloadLibraries: currentPreloadLibraries || '',
          })
        } else {
          setDockerState({
            ...dockerState,
            loading: false,
          })
        }
      })
    }

    const handleDockerImageSelect = (
      e: React.ChangeEvent<HTMLInputElement>,
    ) => {
      if (e.target.value === 'Generic Postgres') {
        // Use memoized enhanced list for better performance
        const currentDockerImage = dockerImageVersions.slice(-1)[0]

        setDockerState({
          ...dockerState,
          tags: dockerTags.filter((tag) => tag.startsWith(currentDockerImage)),
          locations: dockerLocations.filter((location) => location?.includes(currentDockerImage)),
          images: dockerImageVersions,
          data: enhancedDockerImages,
        })

        formik.setValues({
          ...formik.values,
          dockerImage: currentDockerImage,
          dockerImageType: e.target.value,
          dockerTag: dockerTags[0],
          dockerPath: dockerLocations[0],
          sharedPreloadLibraries:
            'pg_stat_statements,pg_stat_kcache,pg_cron,pgaudit,anon',
        })
      } else if (e.target.value === 'custom') {
        formik.setValues({
          ...formik.values,
          dockerImage: '',
          dockerPath: '',
          dockerTag: '',
          sharedPreloadLibraries: '',
          dockerImageType: e.target.value,
        })
      } else {
        formik.setValues({
          ...formik.values,
          dockerImageType: e.target.value,
        })
        fetchSeImages({
          packageGroup: e.target.value,
        })
      }

      handleSelectPgOptions(e, 'pgDumpCustomOptions')
      handleSelectPgOptions(e, 'pgRestoreCustomOptions')
    }

    const handleDockerVersionSelect = (
      e: React.ChangeEvent<HTMLInputElement>,
    ) => {
      if (formik.values.dockerImageType !== 'custom') {
        const updatedDockerTags = dockerState.data
          .map((image) => image.tag)
          .filter((tag) => tag.startsWith(e.target.value))

        setDockerState({
          ...dockerState,
          tags: updatedDockerTags,
        })

        // Add safety check for empty array
        const firstTag = updatedDockerTags[0]
        if (firstTag) {
          const currentLocation = dockerState.data.find(
            (image) => image.tag === firstTag,
          )?.location

          formik.setValues({
            ...formik.values,
            dockerTag: firstTag,
            dockerImage: e.target.value,
            dockerPath: currentLocation || '',
          })
        } else {
          // Fallback when no matching tags found
          formik.setValues({
            ...formik.values,
            dockerImage: e.target.value,
            dockerTag: '',
            dockerPath: '',
          })
        }
      } else {
        formik.setValues({
          ...formik.values,
          dockerImage: e.target.value,
          dockerPath: e.target.value,
        })
      }
    }

    // Set initial data, empty string for password
    useEffect(() => {
      if (configData) {
        for (const [key, value] of Object.entries(configData)) {
          if (key !== 'password') {
            formik.setFieldValue(key, value)
          }

          if (key === 'tuningParams') {
            formik.setFieldValue(key, value)
          }

          if (customOrGenericImage(configData?.dockerImageType)) {
            if (configData?.dockerImageType === 'Generic Postgres') {
              // Use memoized enhanced list for better performance
              const currentDockerImage = enhancedDockerImages.find(
                (image) => image.location === configData?.dockerPath || image.tag === configData?.dockerTag
              )

              if (currentDockerImage) {
                setDockerState({
                  ...dockerState,
                  tags: dockerTags.filter((tag) =>
                    tag.startsWith(currentDockerImage.pg_major_version),
                  ),
                  images: dockerImageVersions,
                  data: enhancedDockerImages,
                })

                formik.setFieldValue('dockerTag', currentDockerImage.tag)
                formik.setFieldValue('dockerImage', currentDockerImage.pg_major_version)
                formik.setFieldValue('dockerPath', currentDockerImage.location)
              } else {
                // Fallback: shouldn't happen with enhancedDockerImages, but keep for safety
                const fallbackVersion = dockerImageVersions.slice(-1)[0]
                
                setDockerState({
                  ...dockerState,
                  tags: dockerTags.filter((tag) => tag.startsWith(fallbackVersion)),
                  images: dockerImageVersions,
                  data: enhancedDockerImages,
                })

                formik.setFieldValue('dockerTag', configData?.dockerTag || '')
                formik.setFieldValue('dockerImage', fallbackVersion)
                formik.setFieldValue('dockerPath', configData?.dockerPath || '')
              }
            } else {
              formik.setFieldValue('dockerImage', configData?.dockerPath)
            }
          }
        }
      }
    }, [config, configData?.dockerPath, configData?.dockerTag, configData?.dockerImageType])

    useEffect(() => {
      getEngine(instanceId).then((res) => {
        setDledition(String(res?.edition))
      })
    }, [])

    useEffect(() => {
      const initialFetch = async () => {
        if (
          formik.dirty &&
          !isCeEdition &&
          !customOrGenericImage(configData?.dockerImageType)
        ) {
          await getFullConfig(instanceId).then(async (data) => {
            if (data) {
              await fetchSeImages({
                packageGroup: configData?.dockerImageType as string,
                dockerTag: configData?.dockerTag,
                initialRender: true,
              })
            }
          })
        }
      }
      initialFetch()
    }, [
      formik.dirty,
      configData?.dockerImageType,
      configData?.dockerTag,
      isCeEdition,
    ])

    return (
      <div className={styles.root}>
        <Snackbar
          onClick={() => {
            Boolean(dockerState.error)
              ? setDockerState({
                  ...dockerState,
                  error: '',
                })
              : undefined
          }}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          open={
            (isConfigurationDisabled || Boolean(dockerState.error)) &&
            !isModalOpen
          }
          message={
            Boolean(dockerState.error)
              ? dockerState.error
              : disableConfigModification
              ? PREVENT_MODIFYING_MESSAGE
              : NON_LOGICAL_RETRIEVAL_MESSAGE
          }
          className={styles.snackbar}
        />
        {!config && isConfigurationLoading ? (
          <div className={styles.spinnerContainer}>
            <Spinner size="lg" className={styles.spinner} />
          </div>
        ) : (
          <Box>
            <Header retrievalMode="logical" setOpen={handleModalClick} />
            <Box>
              <Box>
                <FormControlLabel
                  control={
                    <Checkbox
                      name="debug"
                      checked={formik.values.debug}
                      disabled={isConfigurationDisabled}
                      onChange={(e) =>
                        formik.setFieldValue('debug', e.target.checked)
                      }
                      classes={{
                        root: classes.checkboxRoot,
                      }}
                    />
                  }
                  label={'Debug mode'}
                />
              </Box>
              <Box mb={1} mt={1}>
                <ConfigSectionTitle tag="retrieval" />
                <Box mt={1}>
                  <Typography className={styles.subsection}>
                    Subsection "retrieval.spec.logicalDump"
                  </Typography>
                  <span className={classes.grayText}>
                    Source database credentials and dumping options.
                  </span>
                  <InputWithTooltip
                    label="source.connection.host *"
                    value={formik.values.host}
                    error={formik.errors.host}
                    tooltipText={tooltipText.host}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('host', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.port *"
                    value={formik.values.port}
                    error={formik.errors.port}
                    tooltipText={tooltipText.port}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('port', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.username *"
                    value={formik.values.username}
                    error={formik.errors.username}
                    tooltipText={tooltipText.username}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('username', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    type="password"
                    value={formik.values.password}
                    label="source.connection.password"
                    tooltipText={tooltipText.password}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('password', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.dbname *"
                    value={formik.values.dbname}
                    error={formik.errors.dbname}
                    tooltipText={tooltipText.dbname}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('dbname', e.target.value)
                    }
                  />
                  <InputWithChip
                    id="databases"
                    value={formik.values.databases}
                    label="Databases to copy"
                    tooltipText={tooltipText.databases}
                    handleDeleteChip={handleDeleteChip}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('databases', e.target.value)
                    }
                  />
                  <Box mt={3} mb={3}>
                    <Button
                      variant="outlined"
                      color="secondary"
                      onClick={() => {
                        onTestConnectionClick({
                          type: 'default',
                        })
                      }}
                      disabled={
                        testConnectionState.default.loading ||
                        isConfigurationDisabled
                      }
                    >
                      Test connection
                      {testConnectionState.default.loading && (
                        <Spinner size="sm" className={styles.spinner} />
                      )}
                    </Button>
                    {testConnectionState.default.message.status ||
                    testConnectionState.default.error ? (
                      <ResponseMessage
                        type={
                          testConnectionState.default.error
                            ? 'error'
                            : testConnectionState.default.message.status
                            ? testConnectionState.default.message.status
                            : ''
                        }
                        message={
                          testConnectionState.default.error ||
                          testConnectionState.default.message.message
                        }
                      />
                    ) : null}
                  </Box>
                  <InputWithTooltip
                    label="pg_dump jobs"
                    value={formik.values.dumpParallelJobs}
                    tooltipText={tooltipText.dumpParallelJobs}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('dumpParallelJobs', e.target.value)
                    }
                  />
                  <InputWithChip
                    value={formik.values.pgDumpCustomOptions}
                    label="pg_dump customOptions"
                    id="pgDumpCustomOptions"
                    tooltipText={tooltipText.pgDumpCustomOptions}
                    handleDeleteChip={handleDeleteChip}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue(
                        'pgDumpCustomOptions',
                        e.target.value,
                      )
                    }
                  />
                  <FormControlLabel
                    style={{ maxWidth: 'max-content' }}
                    control={
                      <Checkbox
                        name="dumpIgnoreErrors"
                        checked={formik.values.dumpIgnoreErrors}
                        disabled={isConfigurationDisabled}
                        onChange={(e) =>
                          formik.setFieldValue(
                            'dumpIgnoreErrors',
                            e.target.checked,
                          )
                        }
                        classes={{
                          root: classes.checkboxRoot,
                        }}
                      />
                    }
                    label={'Ignore errors during logical data dump'}
                  />
                </Box>
              </Box>
              <Box mb={2} mt={1}>
                <ConfigSectionTitle tag="databaseContainer" />
                <span
                  className={classes.grayText}
                  style={{ margin: '0.5rem 0 1rem 0', display: 'block' }}
                >
                  DBLab manages various database containers, such as clones.
                  This section defines default container settings.
                </span>
                <div>
                  <SelectWithTooltip
                    label="dockerImage - choose from the list *"
                    value={formik.values.dockerImageType}
                    error={Boolean(formik.errors.dockerImageType)}
                    tooltipText={tooltipText.dockerImageType}
                    disabled={isConfigurationDisabled || dockerState.loading}
                    items={filteredDockerImageOptions.map((image) => {
                      return {
                        value: image.type,
                        children: image.name,
                      }
                    })}
                    onChange={handleDockerImageSelect}
                  />
                  {formik.values.dockerImageType === 'custom' ? (
                    <InputWithTooltip
                      label="dockerImage *"
                      value={formik.values.dockerImage}
                      error={formik.errors.dockerImage}
                      tooltipText={tooltipText.dockerImage}
                      disabled={isConfigurationDisabled}
                      onChange={(e) => {
                        formik.setValues({
                          ...formik.values,
                          dockerImage: e.target.value,
                          dockerPath: e.target.value,
                        })
                      }}
                    />
                  ) : (
                    <>
                      <SelectWithTooltip
                        label="dockerImage - Postgres major version *"
                        value={formik.values.dockerImage}
                        error={Boolean(formik.errors.dockerImage)}
                        tooltipText={tooltipText.dockerImage}
                        disabled={
                          isConfigurationDisabled ||
                          dockerState.loading ||
                          !dockerState.images.length
                        }
                        loading={dockerState.loading}
                        items={dockerState.images
                          .slice()
                          .reverse()
                          .map((image) => {
                            return {
                              value: image,
                              children: image,
                            }
                          })}
                        onChange={handleDockerVersionSelect}
                      />
                      <Box mt={0.5} mb={2}>
                        <Button
                          variant="outlined"
                          color="secondary"
                          onClick={() => {
                            onTestConnectionClick({
                              type: 'dockerImage',
                            })
                          }}
                          disabled={
                            testConnectionState.dockerImage.loading ||
                            isConfigurationDisabled
                          }
                        >
                          Get version from source
                          {testConnectionState.dockerImage.loading && (
                            <Spinner size="sm" className={styles.spinner} />
                          )}
                        </Button>
                        {testConnectionState.dockerImage.message.status ===
                          'error' || testConnectionState.dockerImage.error ? (
                          <ResponseMessage
                            type={
                              testConnectionState.dockerImage.error ||
                              testConnectionState.dockerImage.message
                                ? 'error'
                                : ''
                            }
                            message={
                              testConnectionState.dockerImage.error ||
                              testConnectionState.dockerImage.message.message
                            }
                          />
                        ) : null}
                      </Box>
                      <SelectWithTooltip
                        label="dockerImage - tag *"
                        value={formik.values.dockerTag}
                        error={Boolean(formik.errors.dockerTag)}
                        tooltipText={tooltipText.dockerTag}
                        disabled={
                          isConfigurationDisabled ||
                          dockerState.loading ||
                          !dockerState.tags.length
                        }
                        loading={dockerState.loading}
                        onChange={(e) => {
                          const currentLocation = dockerState.data.find(
                            (image) => image.tag === e.target.value,
                          )?.location as string

                          formik.setValues({
                            ...formik.values,
                            dockerTag: e.target.value,
                            dockerPath: currentLocation,
                          })
                        }}
                        items={dockerState.tags.map((image) => {
                          return {
                            value: image,
                            children: image,
                          }
                        })}
                      />
                    </>
                  )}
                  <Typography paragraph>
                    Cannot find your image? Reach out to support:{' '}
                    <a
                      href={'https://postgres.ai/contact'}
                      target="_blank"
                      className={styles.externalLink}
                    >
                      https://postgres.ai/contact
                      <ExternalIcon className={styles.externalIcon} />
                    </a>
                  </Typography>
                </div>
              </Box>
              <Box mb={3}>
                <ConfigSectionTitle tag="databaseConfigs" />
                <span
                  className={classes.grayText}
                  style={{ marginTop: '0.5rem', display: 'block' }}
                >
                  Default PostgreSQL configuration used for all PostgreSQL instances
                  running in containers managed by DBLab.
                </span>
                <InputWithTooltip
                  type="textarea"
                  label="shared_buffers parameter"
                  value={formik.values.sharedBuffers}
                  tooltipText={tooltipText.sharedBuffers}
                  disabled={isConfigurationDisabled}
                  onChange={(e) =>
                    formik.setFieldValue('sharedBuffers', e.target.value)
                  }
                />
                <InputWithTooltip
                  type="textarea"
                  label="shared_preload_libraries"
                  value={formik.values.sharedPreloadLibraries}
                  tooltipText={tooltipText.sharedPreloadLibraries}
                  disabled={isConfigurationDisabled}
                  onChange={(e) =>
                    formik.setFieldValue(
                      'sharedPreloadLibraries',
                      e.target.value,
                    )
                  }
                />
                <InputWithTooltip
                  type="textarea"
                  label="Query tuning parameters"
                  value={
                    typeof formik.values.tuningParams === 'object'
                      ? Object.entries(
                          formik.values.tuningParams as Record<string, string>,
                        )
                          .map(([key, value]) => `${key}=${value}`)
                          .join('\n')
                      : formik.values.tuningParams
                  }
                  tooltipText={tooltipText.tuningParams}
                  disabled={isConfigurationDisabled}
                  onChange={(e) =>
                    formik.setFieldValue('tuningParams', e.target.value)
                  }
                />
                <Button
                  variant="outlined"
                  color="secondary"
                  onClick={() => {
                    onTestConnectionClick({
                      type: 'fetchTuning',
                    })
                  }}
                  disabled={
                    testConnectionState.fetchTuning.loading ||
                    isConfigurationDisabled
                  }
                >
                  Get from source database
                  {testConnectionState.fetchTuning.loading && (
                    <Spinner size="sm" className={styles.spinner} />
                  )}
                </Button>
                {testConnectionState.fetchTuning.message.status === 'error' ||
                testConnectionState.fetchTuning.error ? (
                  <ResponseMessage
                    type={
                      testConnectionState.fetchTuning.error ||
                      testConnectionState.fetchTuning.message
                        ? 'error'
                        : ''
                    }
                    message={
                      testConnectionState.fetchTuning.error ||
                      testConnectionState.fetchTuning.message.message
                    }
                  />
                ) : null}
              </Box>
              <Box>
                <Box>
                  <Typography className={styles.subsection}>
                    Subsection "retrieval.spec.logicalRestore"
                  </Typography>
                  <span className={classes.grayText}>Restoring options.</span>
                </Box>
                <InputWithTooltip
                  label="pg_restore jobs"
                  value={formik.values.restoreParallelJobs}
                  tooltipText={tooltipText.restoreParallelJobs}
                  disabled={isConfigurationDisabled}
                  onChange={(e) =>
                    formik.setFieldValue('restoreParallelJobs', e.target.value)
                  }
                />
                <InputWithChip
                  value={formik.values.pgRestoreCustomOptions}
                  label="pg_restore customOptions"
                  id="pgRestoreCustomOptions"
                  tooltipText={tooltipText.pgRestoreCustomOptions}
                  handleDeleteChip={handleDeleteChip}
                  disabled={isConfigurationDisabled}
                  onChange={(e) =>
                    formik.setFieldValue(
                      'pgRestoreCustomOptions',
                      e.target.value,
                    )
                  }
                />
                <FormControlLabel
                  style={{ maxWidth: 'max-content' }}
                  control={
                    <Checkbox
                      name="restoreIgnoreErrors"
                      checked={formik.values.restoreIgnoreErrors}
                      disabled={isConfigurationDisabled}
                      onChange={(e) =>
                        formik.setFieldValue(
                          'restoreIgnoreErrors',
                          e.target.checked,
                        )
                      }
                      classes={{
                        root: classes.checkboxRoot,
                      }}
                    />
                  }
                  label={'Ignore errors during logical data restore'}
                />
              </Box>
              <Box mt={1}>
                <Typography className={styles.subsection}>
                  Subsection "retrieval.refresh"
                </Typography>
              </Box>
              <span className={classes.grayText}>
                Define full data refresh on schedule. The process requires at
                least one additional filesystem mount point. The schedule is to
                be specified using{' '}
                <a
                  href="https://en.wikipedia.org/wiki/Cron#Overview"
                  target="_blank"
                  className={styles.externalLink}
                >
                  crontab format
                  <ExternalIcon className={styles.externalIcon} />
                </a>
                .
              </span>
              <InputWithTooltip
                label="timetable"
                value={formik.values.timetable}
                tooltipText={tooltipText.timetable}
                disabled={isConfigurationDisabled}
                onChange={(e) =>
                  formik.setFieldValue('timetable', e.target.value)
                }
              />
            </Box>
            <Box
              mt={2}
              mb={2}
              sx={{
                display: 'flex',
                alignItems: 'center',
              }}
            >
              <Button
                variant="contained"
                color="secondary"
                onClick={() => {
                  formik.submitForm().then(() => {
                    scrollToField()
                  })
                }}
                disabled={formik.isSubmitting || isConfigurationDisabled}
              >
                Apply changes
                {formik.isSubmitting && (
                  <Spinner size="sm" className={styles.spinner} />
                )}
              </Button>
              <Box sx={{ px: 2 }}>
                <Button
                  variant="outlined"
                  color="secondary"
                  onClick={switchTab}
                >
                  Cancel
                </Button>
              </Box>
            </Box>
            {(submitState.status && submitState.response) || configError ? (
              <ResponseMessage
                type={configError ? 'error' : submitState.status}
                message={configError || submitState.response}
              />
            ) : null}
          </Box>
        )}
        <Modal
          title={<ModalTitle />}
          onClose={() => setIsModalOpen(false)}
          isOpen={isModalOpen}
          size="xl"
        >
          <Editor
            height="70vh"
            width="100%"
            defaultLanguage="yaml"
            value={getFullConfigError ? getFullConfigError : fullConfig}
            loading={<StubSpinner />}
            theme="vs-light"
            options={{ domReadOnly: true, readOnly: true }}
          />
        </Modal>
      </div>
    )
  },
)
