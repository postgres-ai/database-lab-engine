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
  TextField,
  Chip,
} from '@material-ui/core'
import { useState, useEffect } from 'react'
import { withStyles, makeStyles } from '@material-ui/core/styles'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { SectionTitle } from '@postgres.ai/shared/components/SectionTitle'
import { StubSpinner } from '@postgres.ai/shared/components/StubSpinner'
import { Button } from '@postgres.ai/shared/components/Button'
import { Header } from './Header'
import { observer } from 'mobx-react-lite'
import Editor from '@monaco-editor/react'
import { useStores } from '@postgres.ai/shared/pages/Instance/context'
import { FormValues, useForm } from './useForm'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import styles from './styles.module.scss'
import { SimpleModalControls } from '@postgres.ai/shared/components/SimpleModalControls'
import { ResponseMessage } from './ResponseMessage'
import { uniqueDatabases } from './utils'

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
  }: {
    switchActiveTab: (activeTab: number) => void
    activeTab: number
    reload: () => void
  }) => {
    const classes = useStyles()
    const stores = useStores()
    const {
      config,
      updateConfig,
      getFullConfig,
      fullConfig,
      testDbSource,
      updateConfigError,
    } = stores.main
    const configData = config && JSON.parse(JSON.stringify(config))
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
      switchActiveTab(0)
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
      Object.keys(connectionData).map(function (key) {
        if (key !== 'password') {
          formik.validateField(key)
        }
      })
      if (isConnectionDataValid) {
        setIsTestConnectionLoading(true)
        testDbSource(connectionData).then((response) => {
          if (response) {
            setConnectionStatus(response.status)
            setConnectionResponse(response.message)
            setIsTestConnectionLoading(false)
          }
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
        <Box>
          <Header retrievalMode="logical" setOpen={handleModalClick} />
          <Box>
            <Box mb={2}>
              <FormControlLabel
                control={
                  <Checkbox
                    name="debug"
                    checked={formik.values.debug}
                    onChange={(e) =>
                      formik.setFieldValue('debug', e.target.checked)
                    }
                    classes={{
                      root: classes.checkboxRoot,
                    }}
                  />
                }
                label={<GrayTextTypography>Debug mode</GrayTextTypography>}
                color="#8a8a8a"
              />
            </Box>
            <Box mb={2}>
              <SectionTitle
                level={2}
                tag="h2"
                text="Section databaseContainer"
              />
              <GrayTextTypography style={{ marginTop: '0.5rem' }}>
                Container settings that will be used by default for each
                Postgres container the DLE manages
              </GrayTextTypography>
              <Box mt={1} mb={2}>
                <TextField
                  className={styles.textField}
                  label="dockerImage"
                  variant="outlined"
                  size="small"
                  margin="normal"
                  value={formik.values.dockerImage}
                  error={Boolean(formik.errors.dockerImage)}
                  onChange={(e) =>
                    formik.setFieldValue('dockerImage', e.target.value)
                  }
                />
              </Box>
            </Box>
            <Box mb={3}>
              <SectionTitle level={2} tag="h2" text="Section databaseConfigs" />
              <GrayTextTypography style={{ marginTop: '0.5rem' }}>
                Default PostgreSQL configuration used by all Postgres instances
                managed by DLE. Each section have additional settings to
                override these defaults.
              </GrayTextTypography>
              <Box mt={2} mb={1}>
                <TextField
                  className={styles.textField}
                  label="configs.shared_buffers"
                  variant="outlined"
                  size="small"
                  value={formik.values.sharedBuffers}
                  onChange={(e) =>
                    formik.setFieldValue('sharedBuffers', e.target.value)
                  }
                />
              </Box>
              <Box mt={2} mb={2}>
                <TextField
                  className={styles.textField}
                  label="configs.shared_preload_libraries"
                  variant="outlined"
                  size="small"
                  value={formik.values.sharedPreloadLibraries}
                  onChange={(e) =>
                    formik.setFieldValue(
                      'sharedPreloadLibraries',
                      e.target.value,
                    )
                  }
                />
              </Box>
            </Box>
            <Box mb={3}>
              <SectionTitle level={2} tag="h2" text="Section retrieval" />
              <Box mt={2}>
                <Typography>Subsection retrieval.spec.logicalDump</Typography>
                <GrayTextTypography>
                  Source database credentials and dumping options.
                </GrayTextTypography>
                <Box mt={2} mb={2}>
                  <TextField
                    className={styles.textField}
                    label="source.connection.host"
                    variant="outlined"
                    size="small"
                    value={formik.values.host}
                    error={Boolean(formik.errors.host)}
                    onChange={(e) =>
                      formik.setFieldValue('host', e.target.value)
                    }
                  />
                </Box>
                <Box mt={2} mb={2}>
                  <TextField
                    className={styles.textField}
                    label="source.connection.port"
                    variant="outlined"
                    size="small"
                    value={formik.values.port}
                    error={Boolean(formik.errors.port)}
                    onChange={(e) =>
                      formik.setFieldValue('port', e.target.value)
                    }
                  />
                </Box>
                <Box mt={2} mb={2}>
                  <TextField
                    className={styles.textField}
                    label="source.connection.username"
                    variant="outlined"
                    size="small"
                    value={formik.values.username}
                    error={Boolean(formik.errors.username)}
                    onChange={(e) =>
                      formik.setFieldValue('username', e.target.value)
                    }
                  />
                </Box>
                <Box mt={2} mb={2}>
                  <TextField
                    className={styles.textField}
                    label="source.connection.password"
                    variant="outlined"
                    size="small"
                    type="password"
                    placeholder={formik.values.password}
                    onChange={(e) =>
                      formik.setFieldValue('password', e.target.value)
                    }
                  />
                </Box>
                <Box mt={2} mb={0}>
                  <TextField
                    className={styles.textField}
                    label="source.connection.dbname"
                    variant="outlined"
                    size="small"
                    value={formik.values.dbname}
                    error={Boolean(formik.errors.dbname)}
                    onChange={(e) =>
                      formik.setFieldValue('dbname', e.target.value)
                    }
                  />
                </Box>
                <Box mt={1} mb={2}>
                  <div className={styles.textField}>
                    <TextField
                      className={styles.textField}
                      variant="outlined"
                      helperText={
                        'Database(s) divided by space or end of string'
                      }
                      margin="normal"
                      onChange={(e) =>
                        formik.setFieldValue('databases', e.target.value)
                      }
                      value={formik.values.databases}
                      multiline
                      label="Databases"
                      inputProps={{
                        name: 'databases',
                        id: 'databases',
                      }}
                      InputLabelProps={{
                        shrink: true,
                      }}
                    />
                  </div>

                  <div>
                    {formik.values.databases &&
                      uniqueDatabases(formik.values.databases)
                        .split(' ')
                        .map((database, index) => {
                          if (database !== '') {
                            return (
                              <Chip
                                key={index}
                                className={styles.chip}
                                label={database}
                                onDelete={(event) =>
                                  handleDeleteDatabase(event, database)
                                }
                                color="primary"
                              />
                            )
                          }
                        })}
                  </div>
                </Box>
                <Box mt={2}>
                  <Button
                    variant="primary"
                    size="medium"
                    onClick={onTestConnectionClick}
                    isDisabled={isTestConnectionLoading}
                  >
                    Test connection
                    {isTestConnectionLoading && (
                      <Spinner size="sm" className={styles.spinner} />
                    )}
                  </Button>
                </Box>
                {connectionStatus && connectionResponse ? (
                  <ResponseMessage
                    type={connectionStatus}
                    message={connectionResponse}
                  />
                ) : null}
              </Box>
            </Box>
            <Box mt={2} mb={2}>
              <TextField
                className={styles.textField}
                label="pg_dump jobs"
                variant="outlined"
                size="small"
                value={formik.values.pg_dump}
                onChange={(e) =>
                  formik.setFieldValue('pg_dump', e.target.value)
                }
              />
            </Box>
            <Box mt={2} mb={2}>
              <TextField
                className={styles.textField}
                label="pg_restore jobs"
                variant="outlined"
                size="small"
                value={formik.values.pg_restore}
                onChange={(e) =>
                  formik.setFieldValue('pg_restore', e.target.value)
                }
              />
            </Box>
            <Box>
              <Typography>Subsection retrieval.refresh</Typography>
            </Box>
            <GrayTextTypography>
              Define full data refresh on schedule. The process requires at
              least one additional filesystem mount point. The schedule is to be
              specified using{' '}
              <a
                href="https://en.wikipedia.org/wiki/Cron#Overview"
                target="_blank"
              >
                crontab format.
              </a>
            </GrayTextTypography>
            <Box mt={2} mb={2}>
              <TextField
                className={styles.textField}
                label="timetable"
                variant="outlined"
                size="small"
                value={formik.values.timetable}
                onChange={(e) =>
                  formik.setFieldValue('timetable', e.target.value)
                }
              />
            </Box>
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
              isDisabled={formik.isSubmitting}
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
                onClick={() => switchActiveTab(0)}
              >
                Cancel
              </Button>
            </Box>
          </Box>
          {(submitStatus && submitMessage) || updateConfigError ? (
            <ResponseMessage
              type={updateConfigError ? 'error' : submitStatus}
              message={updateConfigError || submitMessage}
            />
          ) : null}
        </Box>
        <Modal
          title="Full configuration file (view only)"
          onClose={() => setIsOpen(false)}
          isOpen={isOpen}
          size="xl"
        >
          <Editor
            height="70vh"
            width="100%"
            defaultLanguage="yaml"
            value={fullConfig}
            loading={<StubSpinner />}
            theme="vs-light"
            options={{ domReadOnly: true, readOnly: true }}
          />
          <SimpleModalControls
            items={[
              {
                text: 'Cancel',
                onClick: () => setIsOpen(false),
              },
            ]}
          />
        </Modal>
      </div>
    )
  },
)
