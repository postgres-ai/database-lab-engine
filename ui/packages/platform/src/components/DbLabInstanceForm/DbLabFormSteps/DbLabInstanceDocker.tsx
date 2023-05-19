import { Box } from '@mui/material'
import { useEffect, useState } from 'react'
import { makeStyles, Button } from '@material-ui/core'

import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { Spinner } from '@postgres.ai/shared/components/Spinner'

import { getCloudImages } from 'api/cloud/getCloudImages'
import {
  getGcpAccountContents,
  getPlaybookCommand,
} from 'components/DbLabInstanceForm/utils'
import { initialState } from '../reducer'
import { getOrgKeys } from 'api/cloud/getOrgKeys'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { DblabInstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/DbLabInstanceFormCreation'

const useStyles = makeStyles({
  marginTop: {
    marginTop: '20px',
  },
  marginBottom: {
    marginBottom: '20px',
    display: 'block',
  },
  spinner: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    height: '100%',
  },
  title: {
    fontWeight: 600,
    fontSize: '15px',
    margin: '10px 0',
  },
  code: {
    backgroundColor: '#eee',
    borderRadius: '3px',
    padding: '0 3px',
    marginLeft: '0.25em',
  },
})

export const DblabInstanceDocker = ({
  state,
  orgId,
  goBack,
  goBackToForm,
  formStep,
  setFormStep,
}: {
  state: typeof initialState
  orgId: number
  goBack: () => void
  goBackToForm: () => void
  formStep: string
  setFormStep: (step: string) => void
}) => {
  const classes = useStyles()
  const [orgKey, setOrgKey] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [cloudImages, setCloudImages] = useState([])
  const [orgKeyError, setOrgKeyError] = useState(false)

  useEffect(() => {
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
  }, [
    orgId,
    state.instanceType.arch,
    state.location.native_code,
    state.provider,
  ])

  return (
    <DblabInstanceFormCreation formStep={formStep} setFormStep={setFormStep}>
      {isLoading ? (
        <span className={classes.spinner}>
          <Spinner />
        </span>
      ) : (
        <>
          {orgKeyError ? (
            <ErrorStub
              title="Error 404"
              message="orgKey not found"
            />
          ) : state.provider === 'digitalocean' ? (
            <>
              <p className={classes.title}>1. Create Personal Access Token</p>
              <p className={classes.marginBottom}>
                Documentation:{' '}
                <a
                  href="https://docs.digitalocean.com/reference/api/create-personal-access-token"
                  target="_blank"
                  rel="noreferrer"
                >
                  https://docs.digitalocean.com/reference/api/create-personal-access-token
                </a>
              </p>{' '}
              <p className={classes.title}>
                2. Export <code className={classes.code}>DO_API_TOKEN</code>
              </p>{' '}
              <SyntaxHighlight content={`export DO_API_TOKEN=XXXXXX`} />{' '}
            </>
          ) : state.provider === 'hetzner' ? (
            <>
              <p className={classes.title}>1. Create API Token</p>
              <p className={classes.marginBottom}>
                Documentation:{' '}
                <a
                  href="https://docs.hetzner.com/cloud/api/getting-started/generating-api-token"
                  target="_blank"
                  rel="noreferrer"
                >
                  https://docs.hetzner.com/cloud/api/getting-started/generating-api-token
                </a>
              </p>{' '}
              <p className={classes.title}>
                2. Export <code className={classes.code}>HCLOUD_API_TOKEN</code>
              </p>{' '}
              <SyntaxHighlight content={`export HCLOUD_API_TOKEN=XXXXXX`} />{' '}
            </>
          ) : state.provider === 'aws' ? (
            <>
              <p className={classes.title}>1. Create access key</p>
              <p className={classes.marginBottom}>
                Documentation:{' '}
                <a
                  href="https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html"
                  target="_blank"
                  rel="noreferrer"
                >
                  https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html
                </a>
              </p>{' '}
              <p className={classes.title}>
                2. Export{' '}
                <code className={classes.code}>AWS_ACCESS_KEY_ID </code> and
                <code className={classes.code}>AWS_SECRET_ACCESS_KEY</code>
              </p>{' '}
              <SyntaxHighlight
                content={`export AWS_ACCESS_KEY_ID=XXXXXX\nexport AWS_SECRET_ACCESS_KEY=XXXXXXXXXXXX`}
              />
            </>
          ) : state.provider === 'gcp' ? (
            <>
              <p className={classes.title}>1. Create a service account</p>
              <p>
                Create and save the JSON key for the service account and point
                to them using GCP_SERVICE_ACCOUNT_CONTENTS variable.
              </p>
              <p className={classes.marginBottom}>
                Documentation:{' '}
                <a
                  href="https://developers.google.com/identity/protocols/oauth2/service-account#creatinganaccount"
                  target="_blank"
                  rel="noreferrer"
                >
                  https://developers.google.com/identity/protocols/oauth2/service-account#creatinganaccount
                </a>
              </p>{' '}
              <p className={classes.title}>
                2. Export{' '}
                <code className={classes.code}>
                  GCP_SERVICE_ACCOUNT_CONTENTS{' '}
                </code>
              </p>{' '}
              <SyntaxHighlight content={getGcpAccountContents()} />{' '}
              <p className={classes.title}>
                3. Run ansible playbook to create server and install DLE SE
              </p>{' '}
            </>
          ) : null}{' '}
          {!orgKeyError && (
            <>
              <p className={classes.title}>
                3. Run ansible playbook to create server and install DLE SE
              </p>{' '}
              <SyntaxHighlight
                content={getPlaybookCommand(state, cloudImages[0], orgKey)}
              />{' '}
              <p className={classes.title}>
                4. After the code snippet runs successfully, follow the
                directions displayed in the resulting output to start using DLE
                UI/API/CLI.
              </p>{' '}
              <Box
                sx={{
                  display: 'flex',
                  gap: '10px',
                  margin: '20px 0',
                }}
              >
                <Button variant="contained" color="primary" onClick={goBack}>
                  See list of instances
                </Button>
                <Button
                  variant="outlined"
                  color="secondary"
                  onClick={goBackToForm}
                >
                  Back to form
                </Button>
              </Box>
            </>
          )}
        </>
      )}
    </DblabInstanceFormCreation>
  )
}
