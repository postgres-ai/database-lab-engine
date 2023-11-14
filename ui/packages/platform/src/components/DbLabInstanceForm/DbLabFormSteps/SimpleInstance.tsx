import { Box } from '@mui/material'
import { Button, TextField } from '@material-ui/core'
import { useCallback, useEffect, useState } from 'react'

import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { formStyles } from 'components/DbLabInstanceForm/DbLabFormSteps/AnsibleInstance'
import { ResponseMessage } from '@postgres.ai/shared/pages/Configuration/ResponseMessage'
import { InstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/InstanceFormCreation'

import { initialState } from '../reducer'
import { cloudProviderName } from '../utils'
import { getOrgKeys } from 'api/cloud/getOrgKeys'
import { establishConnection } from './streamLogs'
import { launchDeploy } from 'api/configs/launchDeploy'
import { getCloudImages } from 'api/cloud/getCloudImages'
import { regenerateCode } from 'api/configs/regenerateCode'
import { useWsScroll } from '@postgres.ai/shared/pages/Logs/hooks/useWsScroll'
import { getTaskState } from 'api/configs/getTaskState'

const SimpleInstanceDocumentation = ({
  state,
  isLoading,
  secondStep,
  documentation,
  deployingState,
  handleDeploy,
}: {
  isLoading: boolean
  documentation: string
  secondStep: JSX.Element
  state: typeof initialState
  handleDeploy: (e: React.FormEvent<HTMLFormElement>) => void
  deployingState: {
    status: string
    error: string
  }
}) => {
  const classes = formStyles()

  useEffect(() => {
    const textFields = document.querySelectorAll('input[type="text"]')
    textFields?.forEach((textField) => {
      textField.addEventListener('blur', () => {
        textField.setAttribute('type', 'password')
      })
      textField.addEventListener('focus', () => {
        textField.setAttribute('type', 'text')
      })
    })
  }, [])

  return (
    <form onSubmit={handleDeploy} className={classes.maxContentWidth}>
      <h1 className={classes.mainTitle}>{cloudProviderName(state.provider)}</h1>
      <p>
        {state.provider === 'aws' ? (
          <>
            {`Create a ${cloudProviderName(state.provider)} access key per`}{' '}
            <a href={documentation} target="_blank" rel="noreferrer">
              the official documentation.
            </a>{' '}
            These secrets will be used securely in a
          </>
        ) : state.provider === 'gcp' ? (
          <>
            {`Create a ${cloudProviderName(
              state.provider,
            )} service account per`}{' '}
            <a href={documentation} target="_blank" rel="noreferrer">
              the official documentation.
            </a>{' '}
            The service account content will be used securely in a
          </>
        ) : (
          <>
            {`Generate a ${cloudProviderName(state.provider)} API token per`}{' '}
            <a href={documentation} target="_blank" rel="noreferrer">
              the official documentation.
            </a>{' '}
            This token will be used securely in a
          </>
        )}{' '}
        <a href="https://postgres.ai/" target="_blank" rel="noreferrer">
          Postgres.ai
        </a>{' '}
        temporary container and will not be stored.
      </p>
      {secondStep}
      <div className={classes.marginTop}>
        <Button
          type="submit"
          color="primary"
          variant="contained"
          disabled={isLoading || deployingState.status === 'finished'}
        >
          {isLoading && <Spinner size="sm" className={classes.buttonSpinner} />}
          {isLoading
            ? 'Deploying...'
            : deployingState.status === 'finished'
            ? 'Deployed'
            : 'Deploy'}
        </Button>
      </div>

      {deployingState.error && (
        <ResponseMessage type="error" message={deployingState.error} />
      )}
    </form>
  )
}

export const SimpleInstance = ({
  cluster,
  state,
  orgId,
  userID,
  goBackToForm,
  formStep,
  setFormStep,
}: {
  cluster?: boolean
  state: typeof initialState
  orgId: number
  userID?: number
  goBackToForm: () => void
  formStep: string
  setFormStep: (step: string) => void
}) => {
  const classes = formStyles()
  const hasTaskID =
    new URLSearchParams(window.location.search).get('taskID') === state.taskID
  const logElement = document.getElementById('logs-container')
  const [orgKey, setOrgKey] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [taskStatus, setTaskStatus] = useState('')
  const [isConnected, setIsConnected] = useState(false)
  const [deployingState, setDeployingState] = useState({
    status: 'stale',
    error: '',
  })
  useWsScroll(deployingState.status === 'loading', true)
  const [cloudImages, setCloudImages] = useState([
    {
      native_os_image: '',
    },
  ])
  const [orgKeyError, setOrgKeyError] = useState(false)

  const [extraEnvs, setExtraEnvs] = useState({
    DO_API_TOKEN: '',
    HCLOUD_API_TOKEN: '',
    AWS_ACCESS_KEY_ID: '',
    AWS_SECRET_ACCESS_KEY: '',
    GCP_SERVICE_ACCOUNT_CONTENTS: '',
  })

  useEffect(() => {
    if (
      state.provider &&
      state.location.native_code &&
      state.instanceType?.arch
    ) {
      setIsLoading(true)
      getOrgKeys(orgId).then((data) => {
        if (data.error !== null || !Array.isArray(data.response)) {
          setIsLoading(false)
          setOrgKeyError(true)
        } else {
          setOrgKeyError(false)
          setOrgKey(data.response[0].value)
        }
      })
      getCloudImages({
        os_name: 'Ubuntu',
        os_version: '22.04%20LTS',
        arch: state.instanceType.arch,
        cloud_provider: state.provider,
        region: state.provider === 'aws' ? state.location.native_code : 'all',
      }).then((data) => {
        setIsLoading(false)
        setOrgKeyError(false)
        setCloudImages(data.response)
      })
    }
  }, [
    orgId,
    state.instanceType?.arch,
    state.location.native_code,
    state.provider,
  ])

  useEffect(() => {
    const handleHeightChange = () => {
      if (logElement) {
        logElement.scrollIntoView({
          behavior: 'smooth',
          block: 'end',
        })
      }
    }

    const observer = new ResizeObserver(handleHeightChange)
    if (logElement) {
      observer.observe(logElement)
    }

    return () => {
      if (logElement) {
        observer.unobserve(logElement)
      }
    }
  }, [logElement])

  const establishWebsocketConnection = useCallback(
    ({ taskId, otCode }: { taskId: string; otCode: string }) => {
      establishConnection({
        taskId: taskId,
        otCode: otCode,
        userID,
        isConnected,
        setIsConnected,
      }).then(() => {
        getTaskState({ taskID: taskId, userID }).then((status) => {
          if (status.response) {
            const responseStatus =
              status.response?.state === 'error' ||
              status.response?.state === 'finished'
                ? 'finished'
                : 'loading'
            setTaskStatus(responseStatus)
            setDeployingState({
              status: 'finished',
              error: '',
            })
          } else if (status.error) {
            setDeployingState({
              status: 'finished',
              error: status.error?.Error,
            })
          }
        })
      })
    },
    [isConnected, userID],
  )

  useEffect(() => {
    if (
      hasTaskID &&
      userID &&
      Object.values(extraEnvs).every((x) => x === null || x === '') &&
      taskStatus !== 'error' &&
      taskStatus !== 'finished'
    ) {
      setDeployingState({
        status: 'loading',
        error: '',
      })
      getTaskState({ taskID: state.taskID, userID }).then((data) => {
        if (data.response?.state) {
          regenerateCode({ taskID: state.taskID, userID }).then((res) => {
            if (res.response) {
              establishWebsocketConnection({
                taskId: state.taskID,
                otCode: res.response.otCode,
              })
            } else if (res.error) {
              setDeployingState({
                status: 'finished',
                error: res.error?.Error,
              })
            }
          })
        } else if (data.error) {
          setDeployingState({
            status: 'finished',
            error: data.error?.Error,
          })
        }
      })
    }
  }, [
    hasTaskID,
    state.taskID,
    userID,
    isConnected,
    extraEnvs,
    taskStatus,
    establishWebsocketConnection,
  ])

  const handleDeploy = useCallback(
    async (e: React.FormEvent<HTMLFormElement>) => {
      e.preventDefault()
      if (logElement) {
        logElement.innerHTML = ''
      }

      setDeployingState({
        status: 'loading',
        error: '',
      })
      await launchDeploy({
        launchType: cluster ? 'cluster' : 'instance',
        state: state,
        userID: userID,
        extraEnvs: extraEnvs,
        orgKey: orgKey,
        cloudImage: cloudImages[0]?.native_os_image,
      })
        .then(async (data) => {
          if (data.response) {
            window.history.pushState(
              {},
              '',
              `${window.location.pathname}?taskID=${data.response.taskID}&provider=${state.provider}`,
            )
            establishWebsocketConnection({
              taskId: data.response.taskID,
              otCode: data.response.otCode,
            })
            setDeployingState({
              status: 'finished',
              error: '',
            })
          } else if (data.error) {
            const error =
              data.error.Error ||
              data.error.Errors[0] ||
              data.error.FieldErrors.playbook

            setDeployingState({
              status: 'stale',
              error: error,
            })
            if (logElement) {
              logElement.innerHTML = error
            }
          }
        })
        .catch(() => {
          setDeployingState({
            ...deployingState,
            status: 'stale',
          })
        })
    },
    [
      state,
      extraEnvs,
      orgKey,
      cloudImages,
      userID,
      logElement,
      deployingState,
      establishWebsocketConnection,
    ],
  )

  const isFormDisabled =
    deployingState.status === 'loading' ||
    deployingState.status === 'finished' ||
    isConnected ||
    (hasTaskID && Object.values(extraEnvs).every((x) => x === null || x === ''))

  return (
    <InstanceFormCreation
      formStep={formStep}
      setFormStep={setFormStep}
      fullWidth
    >
      {isLoading ? (
        <span className={classes.spinner}>
          <Spinner />
        </span>
      ) : (
        <>
          {orgKeyError ? (
            <ErrorStub title="Error 404" message="orgKey not found" />
          ) : state.provider === 'digitalocean' ? (
            <SimpleInstanceDocumentation
              state={state}
              handleDeploy={handleDeploy}
              deployingState={deployingState}
              isLoading={deployingState.status === 'loading' || isConnected}
              documentation="https://docs.digitalocean.com/reference/api/create-personal-access-token"
              secondStep={
                <TextField
                  type="text"
                  required
                  label="DO_API_TOKEN"
                  variant="outlined"
                  disabled={isFormDisabled}
                  fullWidth
                  value={
                    isFormDisabled ? '****************' : extraEnvs.DO_API_TOKEN
                  }
                  className={classes.marginTop}
                  InputLabelProps={{
                    shrink: true,
                  }}
                  onChange={(e) =>
                    setExtraEnvs({
                      ...extraEnvs,
                      DO_API_TOKEN: e.target.value,
                    })
                  }
                />
              }
            />
          ) : state.provider === 'hetzner' ? (
            <SimpleInstanceDocumentation
              state={state}
              handleDeploy={handleDeploy}
              deployingState={deployingState}
              isLoading={deployingState.status === 'loading' || isConnected}
              documentation="https://docs.hetzner.com/cloud/api/getting-started/generating-api-token"
              secondStep={
                <TextField
                  type="text"
                  required
                  label="HCLOUD_API_TOKEN"
                  disabled={isFormDisabled}
                  variant="outlined"
                  fullWidth
                  value={
                    isFormDisabled
                      ? '****************'
                      : extraEnvs.HCLOUD_API_TOKEN
                  }
                  className={classes.marginTop}
                  InputLabelProps={{
                    shrink: true,
                  }}
                  onChange={(e) =>
                    setExtraEnvs({
                      ...extraEnvs,
                      HCLOUD_API_TOKEN: e.target.value,
                    })
                  }
                />
              }
            />
          ) : state.provider === 'aws' ? (
            <SimpleInstanceDocumentation
              state={state}
              handleDeploy={handleDeploy}
              deployingState={deployingState}
              isLoading={deployingState.status === 'loading' || isConnected}
              documentation="https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html"
              secondStep={
                <>
                  <TextField
                    type="text"
                    required
                    id="aws_access_key_id"
                    label="AWS_ACCESS_KEY_ID"
                    variant="outlined"
                    fullWidth
                    disabled={isFormDisabled}
                    value={
                      isFormDisabled
                        ? '****************'
                        : extraEnvs.AWS_ACCESS_KEY_ID
                    }
                    className={classes.marginTop}
                    InputLabelProps={{
                      shrink: true,
                    }}
                    onChange={(e) =>
                      setExtraEnvs({
                        ...extraEnvs,
                        AWS_ACCESS_KEY_ID: e.target.value,
                      })
                    }
                  />
                  <TextField
                    type="text"
                    required
                    label="AWS_SECRET_ACCESS_KEY"
                    variant="outlined"
                    disabled={isFormDisabled}
                    fullWidth
                    value={
                      isFormDisabled
                        ? '****************'
                        : extraEnvs.AWS_SECRET_ACCESS_KEY
                    }
                    className={classes.marginTop}
                    InputLabelProps={{
                      shrink: true,
                    }}
                    onChange={(e) =>
                      setExtraEnvs({
                        ...extraEnvs,
                        AWS_SECRET_ACCESS_KEY: e.target.value,
                      })
                    }
                  />
                </>
              }
            />
          ) : state.provider === 'gcp' ? (
            <SimpleInstanceDocumentation
              state={state}
              handleDeploy={handleDeploy}
              deployingState={deployingState}
              isLoading={deployingState.status === 'loading' || isConnected}
              documentation="https://developers.google.com/identity/protocols/oauth2/service-account#creatinganaccount"
              secondStep={
                <TextField
                  type="text"
                  required
                  label="GCP_SERVICE_ACCOUNT_CONTENTS"
                  variant="outlined"
                  disabled={isFormDisabled}
                  fullWidth
                  onBlur={(e) => {
                    e.target.style.color = 'transparent'
                    e.target.style.textShadow = '0 0 8px rgba(0,0,0,0.5)'
                  }}
                  onFocus={(e) => {
                    e.target.style.color = 'black'
                    e.target.style.textShadow = 'none'
                  }}
                  multiline
                  value={
                    isFormDisabled
                      ? '****************'
                      : extraEnvs.GCP_SERVICE_ACCOUNT_CONTENTS
                  }
                  className={classes.marginTop}
                  InputLabelProps={{
                    shrink: true,
                  }}
                  onChange={(e) =>
                    setExtraEnvs({
                      ...extraEnvs,
                      GCP_SERVICE_ACCOUNT_CONTENTS: e.target.value,
                    })
                  }
                />
              }
            />
          ) : null}
          {deployingState.status === 'loading' ||
          deployingState.status === 'finished' ? (
            <SyntaxHighlight
              id="logs-container"
              style={{
                overflowY: 'auto',
                maxHeight: !hasTaskID ? '350px' : '450px',
                minHeight: '50px',
                overflowX: 'hidden',
                marginTop: '20px',
                border: '1px solid #b4b4b4',
                borderRadius: '4px',
                backgroundColor: 'rgb(250, 250, 250)',
              }}
              content={
                deployingState.status === 'loading'
                  ? 'Deploying...'
                  : deployingState.error
                  ? deployingState.error
                  : ''
              }
            />
          ) : null}
          <Box
            sx={{
              display: 'flex',
              gap: '10px',
              margin: '20px 0',
            }}
          >
            <Button variant="outlined" color="secondary" onClick={goBackToForm}>
              Back to form
            </Button>
          </Box>
        </>
      )}
    </InstanceFormCreation>
  )
}
