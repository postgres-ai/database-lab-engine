import { Box } from '@mui/material'
import { useEffect, useState } from 'react'
import { Button } from '@material-ui/core'

import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'

import { getOrgKeys } from 'api/cloud/getOrgKeys'
import { getCloudImages } from 'api/cloud/getCloudImages'

import {
  getNetworkSubnet,
  getGcpAccountContents,
  getPlaybookCommand,
} from 'components/DbLabInstanceForm/utils'
import {
  InstanceDocumentation,
  formStyles,
} from 'components/DbLabInstanceForm/DbLabFormSteps/AnsibleInstance'
import { InstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/InstanceFormCreation'

import { initialState } from '../reducer'
import { getClusterPlaybookCommand } from 'components/PostgresClusterForm/utils'

export const DockerInstance = ({
  cluster,
  state,
  orgId,
  goBack,
  goBackToForm,
  formStep,
  setFormStep,
}: {
  state: typeof initialState
  cluster?: boolean
  orgId: number
  goBack: () => void
  goBackToForm: () => void
  formStep: string
  setFormStep: (step: string) => void
}) => {
  const classes = formStyles()
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
    <InstanceFormCreation formStep={formStep} setFormStep={setFormStep}>
      {isLoading ? (
        <span className={classes.spinner}>
          <Spinner />
        </span>
      ) : (
        <>
          {orgKeyError ? (
            <ErrorStub title="Error 404" message="orgKey not found" />
          ) : state.provider === 'digitalocean' ? (
            <InstanceDocumentation
              firstStep="Create Personal Access Token"
              documentation="https://docs.digitalocean.com/reference/api/create-personal-access-token"
              secondStep={<code className={classes.code}>DO_API_TOKEN</code>}
              snippetContent="export DO_API_TOKEN=XXXXXX"
              classes={classes}
            />
          ) : state.provider === 'hetzner' ? (
            <InstanceDocumentation
              firstStep="Create API Token"
              documentation="https://docs.hetzner.com/cloud/api/getting-started/generating-api-token"
              secondStep={
                <code className={classes.code}>HCLOUD_API_TOKEN</code>
              }
              snippetContent="export HCLOUD_API_TOKEN=XXXXXX"
              classes={classes}
            />
          ) : state.provider === 'aws' ? (
            <InstanceDocumentation
              firstStep="Create access key"
              documentation="https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html"
              secondStep={
                <>
                  <code className={classes.code}>AWS_ACCESS_KEY_ID</code> and
                  <code className={classes.code}>AWS_SECRET_ACCESS_KEY</code>
                </>
              }
              snippetContent={`export AWS_ACCESS_KEY_ID=XXXXXX\nexport AWS_SECRET_ACCESS_KEY=XXXXXXXXXXXX`}
              classes={classes}
            />
          ) : state.provider === 'gcp' ? (
            <>
              <InstanceDocumentation
                firstStep="Create a service account"
                firsStepDescription={
                  <>
                    Create and save the JSON key for the service account and
                    point to them using{' '}
                    <code className={classes.code}>
                      GCP_SERVICE_ACCOUNT_CONTENTS
                    </code>{' '}
                    variable.
                  </>
                }
                documentation="https://developers.google.com/identity/protocols/oauth2/service-account#creatinganaccount"
                secondStep={
                  <code className={classes.code}>
                    GCP_SERVICE_ACCOUNT_CONTENTS
                  </code>
                }
                snippetContent={getGcpAccountContents()}
                classes={classes}
              />
            </>
          ) : null}
          <p className={classes.title}>
            3. Run ansible playbook to{' '}
            {cluster
              ? 'deploy Postgres Cluster'
              : 'create server and install DBLab SE'}
          </p>
          <SyntaxHighlight
            content={
              cluster
                ? getClusterPlaybookCommand(state, cloudImages[0], orgKey)
                : getPlaybookCommand(state, cloudImages[0], orgKey)
            }
          />
          {getNetworkSubnet(state.provider, classes)}
          <p className={classes.title}>
            4. After the code snippet runs successfully, follow the directions
            displayed in the resulting output to start using{' '}
            {cluster ? 'the database.' : 'DBLab UI/API/CLI.'}
          </p>
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
            <Button variant="contained" color="primary" onClick={goBack}>
              See list of {cluster ? ' clusters' : ' instances'}
            </Button>
          </Box>
        </>
      )}
    </InstanceFormCreation>
  )
}
