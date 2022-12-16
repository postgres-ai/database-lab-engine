/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any small is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useState, useEffect } from 'react'
import { observer } from 'mobx-react-lite'
import Editor from '@monaco-editor/react'
import {
  Checkbox,
  FormControlLabel,
  Typography,
  Snackbar,
  makeStyles,
} from '@material-ui/core'
import Box from '@mui/material/Box'

import { Modal } from '@postgres.ai/shared/components/Modal'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { Button } from '@postgres.ai/shared/components/Button'
import { ExternalIcon } from '@postgres.ai/shared/icons/External'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { MainStore } from '@postgres.ai/shared/pages/Instance/stores/Main'

import { tooltipText } from './tooltipText'
import { FormValues, useForm } from './useForm'
import { ResponseMessage } from './ResponseMessage'
import { ConfigSectionTitle, Header, ModalTitle } from './Header'
import {
  dockerImageOptions,
  defaultPgDumpOptions,
  defaultPgRestoreOptions,
} from './configOptions'
import { formatDockerImageArray, FormValuesKey, uniqueChipValue } from './utils'
import {
  SelectWithTooltip,
  InputWithChip,
  InputWithTooltip,
} from './InputWithTooltip'

import styles from './styles.module.scss'

type PgOptionsType = {
  optionType: string
  addDefaultOptions: string[]
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
    switchActiveTab,
    reload,
    isConfigurationActive,
    disableConfigModification,
  }: {
    switchActiveTab: (_: null, activeTab: number) => void
    reload: () => void
    isConfigurationActive: boolean
    disableConfigModification?: boolean
  }) => {
    const classes = useStyles()
    const stores = useStores()
    const {
      config,
      updateConfig,
      getFullConfig,
      fullConfig,
      testDbSource,
      configError,
      dbSourceError,
      getFullConfigError,
      getEngine,
    } = stores.main
    const configData: MainStore['config'] =
      config && JSON.parse(JSON.stringify(config))
    const isConfigurationDisabled =
      !isConfigurationActive || disableConfigModification
    const [submitMessage, setSubmitMessage] = useState<
      string | React.ReactNode | null
    >('')
    const [dleEdition, setDledition] = useState('')
    const [submitStatus, setSubmitStatus] = useState('')
    const [connectionStatus, setConnectionStatus] = useState('')
    const [isModalOpen, setIsModalOpen] = useState(false)
    const [isConnectionLoading, setIsConnectionLoading] = useState(false)
    const [connectionRes, setConnectionRes] = useState<string | null>(null)
    const [dockerImages, setDockerImages] = useState<string[]>([])

    const switchTab = async () => {
      reload()
      switchActiveTab(null, 0)
    }

    const onSubmit = async (values: FormValues) => {
      setSubmitMessage(null)
      await updateConfig(values).then((response) => {
        if (response?.ok) {
          setSubmitStatus('success')
          setSubmitMessage(
            <p>
              Changes applied.{' '}
              <span className={styles.underline} onClick={switchTab}>
                Switch to Overview
              </span>{' '}
              to see details and to work with clones
            </p>,
          )
        }
      })
    }
    const [{ formik, connectionData, isConnectionDataValid }] =
      useForm(onSubmit)

    const onTestConnectionClick = async () => {
      setConnectionRes(null)
      Object.keys(connectionData).map(function (key: string) {
        if (key !== 'password' && key !== 'db_list') {
          formik.validateField(key)
        }
      })
      if (isConnectionDataValid) {
        setIsConnectionLoading(true)
        testDbSource(connectionData)
          .then((response) => {
            if (response) {
              setConnectionStatus(response.status)
              setConnectionRes(response.message)
              setIsConnectionLoading(false)
            }
          })
          .finally(() => {
            setIsConnectionLoading(false)
          })
      }
    }

    const handleModalClick = async () => {
      await getFullConfig()
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
      formikValue: string,
      initialValue: string | undefined,
      pgOptions: PgOptionsType[],
    ) => {
      let pgValue = formikValue
      // set initial value on change
      formik.setFieldValue(formikName, initialValue)

      const selectedPgOptions = pgOptions.filter(
        (pg) => e.target.value === pg.optionType,
      )

      // add options to formik field
      selectedPgOptions.forEach((pg) => {
        pg.addDefaultOptions.forEach((addOption) => {
          if (!pgValue.includes(addOption)) {
            const addOptionWithSpace = addOption + ' '
            formik.setFieldValue(formikName, (pgValue += addOptionWithSpace))
          }
        })
      })
    }

    const handleDockerImageSelect = (
      e: React.ChangeEvent<HTMLInputElement>,
    ) => {
      const newDockerImages = formatDockerImageArray(e.target.value)
      setDockerImages(newDockerImages)
      handleSelectPgOptions(
        e,
        'pgDumpCustomOptions',
        formik.values.pgDumpCustomOptions,
        configData?.pgDumpCustomOptions,
        defaultPgDumpOptions,
      )
      handleSelectPgOptions(
        e,
        'pgRestoreCustomOptions',
        formik.values.pgRestoreCustomOptions,
        configData?.pgRestoreCustomOptions,
        defaultPgRestoreOptions,
      )
      formik.setFieldValue('dockerImageType', e.target.value)

      // select latest Postgres version on dockerImage change
      if (configData?.dockerImageType !== e.target.value) {
        formik.setFieldValue('dockerImage', newDockerImages.slice(-1)[0])
      } else {
        formik.setFieldValue('dockerImage', configData?.dockerImage)
      }
    }

    // Set initial data, empty string for password
    useEffect(() => {
      if (configData) {
        for (const [key, value] of Object.entries(configData)) {
          if (key !== 'password') {
            formik.setFieldValue(key, value)
          }
          setDockerImages(
            formatDockerImageArray(configData?.dockerImageType || ''),
          )
        }
      }
    }, [config])

    useEffect(() => {
      // Clear response message on tab change and set dockerImageType
      setConnectionRes(null)
      setSubmitMessage(null)
      getEngine().then((res) => {
        setDledition(String(res?.edition))
      })
    }, [])

    return (
      <div className={styles.root}>
        <Snackbar
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          open={isConfigurationDisabled && !isModalOpen}
          message={
            disableConfigModification
              ? PREVENT_MODIFYING_MESSAGE
              : NON_LOGICAL_RETRIEVAL_MESSAGE
          }
          className={styles.snackbar}
        />
        {!config || !dleEdition ? (
          <div className={styles.spinnerContainer}>
            <Spinner size="lg" className={styles.spinner} />
          </div>
        ) : (
          <Box>
            <Header retrievalMode="logical" setOpen={handleModalClick} />
            <Box>
              <Box mb={2}>
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
              <Box mb={2}>
                <ConfigSectionTitle tag="databaseContainer" />
                <span
                  className={classes.grayText}
                  style={{ marginTop: '0.5rem', display: 'block' }}
                >
                  DLE manages various database containers, such as clones. This
                  section defines default container settings.
                </span>
                {dleEdition !== 'community' ? (
                  <div>
                    <SelectWithTooltip
                      label="dockerImage - choose from the list"
                      value={formik.values.dockerImageType}
                      error={Boolean(formik.errors.dockerImageType)}
                      tooltipText={tooltipText.dockerImageType}
                      disabled={isConfigurationDisabled}
                      items={dockerImageOptions.map((image) => {
                        return {
                          value: image.type,
                          children: image.name,
                        }
                      })}
                      onChange={handleDockerImageSelect}
                    />
                    {formik.values.dockerImageType === 'custom' ? (
                      <InputWithTooltip
                        label="dockerImage"
                        value={formik.values.dockerImage}
                        error={formik.errors.dockerImage}
                        tooltipText={tooltipText.dockerImage}
                        disabled={isConfigurationDisabled}
                        onChange={(e) =>
                          formik.setFieldValue('dockerImage', e.target.value)
                        }
                      />
                    ) : (
                      <SelectWithTooltip
                        label="dockerImage - Postgres major version"
                        value={formik.values.dockerImage}
                        error={Boolean(formik.errors.dockerImage)}
                        tooltipText={tooltipText.dockerImage}
                        disabled={isConfigurationDisabled}
                        items={dockerImages.map((image) => {
                          return {
                            value: image,
                            children: image.split(':')[1],
                          }
                        })}
                        onChange={(e) =>
                          formik.setFieldValue('dockerImage', e.target.value)
                        }
                      />
                    )}
                    <Typography paragraph>
                      Haven't found the image you need? Contact support:{' '}
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
                ) : (
                  <InputWithTooltip
                    label="dockerImage"
                    value={formik.values.dockerImage}
                    error={formik.errors.dockerImage}
                    tooltipText={tooltipText.dockerImage}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('dockerImage', e.target.value)
                    }
                  />
                )}
              </Box>
              <Box mb={3}>
                <ConfigSectionTitle tag="databaseConfigs" />
                <span
                  className={classes.grayText}
                  style={{ marginTop: '0.5rem', display: 'block' }}
                >
                  Default Postgres configuration used for all Postgres instances
                  running in containers managed by DLE.
                </span>
                <InputWithTooltip
                  label="configs.shared_buffers"
                  value={formik.values.sharedBuffers}
                  tooltipText={tooltipText.sharedBuffers}
                  disabled={isConfigurationDisabled}
                  onChange={(e) =>
                    formik.setFieldValue('sharedBuffers', e.target.value)
                  }
                />
                <InputWithTooltip
                  label="configs.shared_preload_libraries"
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
              </Box>
              <Box mb={3}>
                <ConfigSectionTitle tag="retrieval" />
                <Box mt={1}>
                  <Typography className={styles.subsection}>
                    Subsection "retrieval.spec.logicalDump"
                  </Typography>
                  <span className={classes.grayText}>
                    Source database credentials and dumping options.
                  </span>
                  <InputWithTooltip
                    label="source.connection.host"
                    value={formik.values.host}
                    error={formik.errors.host}
                    tooltipText={tooltipText.host}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('host', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.port"
                    value={formik.values.port}
                    error={formik.errors.port}
                    tooltipText={tooltipText.port}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('port', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.username"
                    value={formik.values.username}
                    error={formik.errors.username}
                    tooltipText={tooltipText.username}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('username', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.password"
                    tooltipText={tooltipText.password}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('password', e.target.value)
                    }
                  />
                  <InputWithTooltip
                    label="source.connection.dbname"
                    value={formik.values.dbname}
                    error={formik.errors.dbname}
                    tooltipText={tooltipText.dbname}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('dbname', e.target.value)
                    }
                  />
                  <InputWithChip
                    value={formik.values.databases}
                    label="Databases"
                    id="databases"
                    tooltipText={tooltipText.databases}
                    handleDeleteChip={handleDeleteChip}
                    disabled={isConfigurationDisabled}
                    onChange={(e) =>
                      formik.setFieldValue('databases', e.target.value)
                    }
                  />
                  <Box mt={2}>
                    <Button
                      variant="primary"
                      size="medium"
                      onClick={onTestConnectionClick}
                      isDisabled={
                        isConnectionLoading || isConfigurationDisabled
                      }
                    >
                      Test connection
                      {isConnectionLoading && (
                        <Spinner size="sm" className={styles.spinner} />
                      )}
                    </Button>
                  </Box>
                  {(connectionStatus && connectionRes) || dbSourceError ? (
                    <ResponseMessage
                      type={dbSourceError ? 'error' : connectionStatus}
                      message={dbSourceError || connectionRes}
                    />
                  ) : null}
                </Box>
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
              <InputWithTooltip
                label="pg_restore jobs"
                value={formik.values.restoreParallelJobs}
                tooltipText={tooltipText.restoreParallelJobs}
                disabled={isConfigurationDisabled}
                onChange={(e) =>
                  formik.setFieldValue('restoreParallelJobs', e.target.value)
                }
              />
              {dleEdition !== 'community' && (
                <>
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
                </>
              )}
              <Box>
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
                variant="primary"
                size="medium"
                onClick={formik.submitForm}
                isDisabled={formik.isSubmitting || isConfigurationDisabled}
              >
                Apply changes
                {formik.isSubmitting && (
                  <Spinner size="sm" className={styles.spinner} />
                )}
              </Button>
              <Box sx={{ px: 2 }}>
                <Button
                  variant="secondary"
                  size="medium"
                  onClick={() => switchActiveTab(null, 0)}
                >
                  Cancel
                </Button>
              </Box>
            </Box>
            {(submitStatus && submitMessage) || configError ? (
              <ResponseMessage
                type={configError ? 'error' : submitStatus}
                message={configError || submitMessage}
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
