/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any small is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  Box,
  Checkbox,
  FormControlLabel,
  Typography,
  Snackbar,
} from '@material-ui/core'
import { useState, useEffect } from 'react'
import { withStyles, makeStyles } from '@material-ui/core/styles'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { Button } from '@postgres.ai/shared/components/Button'
import { ConfigSectionTitle, Header, ModalTitle } from './Header'
import { observer } from 'mobx-react-lite'
import Editor from '@monaco-editor/react'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { FormValues, useForm } from './useForm'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import styles from './styles.module.scss'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { ResponseMessage } from './ResponseMessage'
import { uniqueDatabases } from './utils'
import { ExternalIcon } from '@postgres.ai/shared/icons/External'
import { InputWithChip, InputWithTooltip } from './InputWithTooltip'
import { tooltipText } from './tooltipText'

const NON_LOGICAL_RETRIEVAL_MESSAGE =
  'Configuration editing is only available in logical mode'
const PREVENT_MODIFYING_MESSAGE = 'Editing is disabled by admin'

export const GrayTextTypography = withStyles({
  root: {
    color: '#8a8a8a',
    fontSize: '12px',
  },
})(Typography)

const useStyles = makeStyles({
  checkboxRoot: {
    padding: '9px 10px',
  },
})

export const Configuration = observer(
  ({
    switchActiveTab,
    activeTab,
    reload,
    isConfigurationActive,
    allowModifyingConfig,
  }: {
    switchActiveTab: (_: null, activeTab: number) => void
    activeTab: number
    reload: () => void
    isConfigurationActive: boolean
    allowModifyingConfig?: boolean
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
    } = stores.main
    const configData = config && JSON.parse(JSON.stringify(config))
    const isConfigurationDisabled =
      !isConfigurationActive || !allowModifyingConfig
    const [submitMessage, setSubmitMessage] = useState<
      string | React.ReactNode | null
    >('')
    const [connectionResponse, setConnectionResponse] = useState<string | null>(
      null,
    )
    const [submitStatus, setSubmitStatus] = useState('')
    const [connectionStatus, setConnectionStatus] = useState('')
    const [isTestConnectionLoading, setIsTestConnectionLoading] =
      useState<boolean>(false)
    const [isOpen, setIsOpen] = useState(false)

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
      setConnectionResponse(null)
      Object.keys(connectionData).map(function (key: string) {
        if (key !== 'password' && key !== 'db_list') {
          formik.validateField(key)
        }
      })
      if (isConnectionDataValid) {
        setIsTestConnectionLoading(true)
        testDbSource(connectionData)
          .then((response) => {
            if (response) {
              setConnectionStatus(response.status)
              setConnectionResponse(response.message)
              setIsTestConnectionLoading(false)
            }
          })
          .finally(() => {
            setIsTestConnectionLoading(false)
          })
      }
    }

    const handleModalClick = async () => {
      await getFullConfig()
      setIsOpen(true)
    }

    const handleDeleteDatabase = (
      _: React.FormEvent<HTMLInputElement>,
      database: string,
    ) => {
      if (formik.values.databases) {
        let currentDatabases = uniqueDatabases(formik.values.databases)
        let curDividers = formik.values.databases.match(
          /[,(\s)(\n)(\r)(\t)(\r\n)]/gm,
        )
        let splitDatabases = currentDatabases.split(' ')
        let newDatabases = ''

        for (let i in splitDatabases) {
          if (curDividers && splitDatabases[i] !== database) {
            newDatabases =
              newDatabases +
              splitDatabases[i] +
              (curDividers[i] ? curDividers[i] : '')
          }
        }

        formik.setFieldValue('databases', newDatabases)
      }
    }

    // Set initial data, empty string for password
    useEffect(() => {
      if (configData) {
        for (const [key, value] of Object.entries(configData)) {
          if (key !== 'password') {
            formik.setFieldValue(key, value)
          }
        }
      }
    }, [config])

    // Clear response message on tab change
    useEffect(() => {
      setConnectionResponse(null)
      setSubmitMessage(null)
    }, [activeTab])

    return (
      <div className={styles.root}>
        <Snackbar
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          open={isConfigurationDisabled && !isOpen}
          message={
            !allowModifyingConfig
              ? PREVENT_MODIFYING_MESSAGE
              : NON_LOGICAL_RETRIEVAL_MESSAGE
          }
          className={styles.snackbar}
        />
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
              <GrayTextTypography style={{ marginTop: '0.5rem' }}>
                DLE manages various database containers, such as clones. This
                section defines default container settings.
              </GrayTextTypography>
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
            </Box>
            <Box mb={3}>
              <ConfigSectionTitle tag="databaseConfigs" />
              <GrayTextTypography style={{ marginTop: '0.5rem' }}>
                Default Postgres configuration used for all Postgres instances
                running in containers managed by DLE.
              </GrayTextTypography>
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
                  formik.setFieldValue('sharedPreloadLibraries', e.target.value)
                }
              />
            </Box>
            <Box mb={3}>
              <ConfigSectionTitle tag="retrieval" />
              <Box mt={1}>
                <Typography className={styles.subsection}>
                  Subsection "retrieval.spec.logicalDump"
                </Typography>
                <GrayTextTypography>
                  Source database credentials and dumping options.
                </GrayTextTypography>
                <InputWithTooltip
                  label="source.connection.host"
                  value={formik.values.host}
                  error={formik.errors.host}
                  tooltipText={tooltipText.host}
                  disabled={isConfigurationDisabled}
                  onChange={(e) => formik.setFieldValue('host', e.target.value)}
                />
                <InputWithTooltip
                  label="source.connection.port"
                  value={formik.values.port}
                  error={formik.errors.port}
                  tooltipText={tooltipText.port}
                  disabled={isConfigurationDisabled}
                  onChange={(e) => formik.setFieldValue('port', e.target.value)}
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
                  handleDeleteDatabase={handleDeleteDatabase}
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
                      isTestConnectionLoading || isConfigurationDisabled
                    }
                  >
                    Test connection
                    {isTestConnectionLoading && (
                      <Spinner size="sm" className={styles.spinner} />
                    )}
                  </Button>
                </Box>
                {(connectionStatus && connectionResponse) || dbSourceError ? (
                  <ResponseMessage
                    type={dbSourceError ? 'error' : connectionStatus}
                    message={dbSourceError || connectionResponse}
                  />
                ) : null}
              </Box>
            </Box>
            <InputWithTooltip
              label="pg_dump jobs"
              value={formik.values.pg_dump}
              tooltipText={tooltipText.pg_dump}
              disabled={isConfigurationDisabled}
              onChange={(e) => formik.setFieldValue('pg_dump', e.target.value)}
            />
            <InputWithTooltip
              label="pg_restore jobs"
              value={formik.values.pg_restore}
              tooltipText={tooltipText.pg_restore}
              disabled={isConfigurationDisabled}
              onChange={(e) =>
                formik.setFieldValue('pg_restore', e.target.value)
              }
            />
            <Box>
              <Typography className={styles.subsection}>
                Subsection "retrieval.refresh"
              </Typography>
            </Box>
            <GrayTextTypography>
              Define full data refresh on schedule. The process requires at
              least one additional filesystem mount point. The schedule is to be
              specified using{' '}
              <a
                href="https://en.wikipedia.org/wiki/Cron#Overview"
                target="_blank"
                className={styles.externalLink}
              >
                crontab format
                <ExternalIcon className={styles.externalIcon} />
              </a>
              .
            </GrayTextTypography>
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
        <Modal
          title={<ModalTitle />}
          onClose={() => setIsOpen(false)}
          isOpen={isOpen}
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
          <SimpleModalControls
            items={[
              {
                text: 'Close',
                onClick: () => setIsOpen(false),
              },
            ]}
          />
        </Modal>
      </div>
    )
  },
)
