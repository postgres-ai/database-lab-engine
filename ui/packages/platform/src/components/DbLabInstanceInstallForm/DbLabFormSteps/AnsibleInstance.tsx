import { Button } from '@material-ui/core'
import { Box } from '@mui/material'
import { useEffect, useState } from 'react'

import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'

import { getOrgKeys } from 'api/cloud/getOrgKeys'

import { formStyles } from 'components/DbLabInstanceForm/DbLabFormSteps/AnsibleInstance'
import { InstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/InstanceFormCreation'
import { SetupStep } from 'components/DbLabInstanceInstallForm/DbLabFormSteps/SetupStep'
import {
  cloneRepositoryCommand,
  getAnsibleInstallationCommand,
  getPlaybookCommand,
} from 'components/DbLabInstanceInstallForm/utils'

import { useCloudProviderProps } from 'hooks/useCloudProvider'

export const AnsibleInstance = ({
  state,
  orgId,
  goBack,
  goBackToForm,
  formStep,
  setFormStep,
}: {
  state: useCloudProviderProps['initialState']
  orgId: number
  goBack: () => void
  goBackToForm: () => void
  formStep: string
  setFormStep: (step: string) => void
}) => {
  const classes = formStyles()
  const [orgKey, setOrgKey] = useState('')
  const [isLoading, setIsLoading] = useState(false)
  const [orgKeyError, setOrgKeyError] = useState(false)

  useEffect(() => {
    setIsLoading(true)
    getOrgKeys(orgId).then((data) => {
      console.log('data', data)
      if (data.error !== null || !Array.isArray(data.response)) {
        setIsLoading(false)
        setOrgKeyError(true)
      } else {
        setIsLoading(false)
        setOrgKeyError(false)
        setOrgKey(data.response[0].value)
      }
    })
  }, [orgId])

  return (
    <InstanceFormCreation formStep={formStep} setFormStep={setFormStep} install>
      {isLoading ? (
        <span className={classes.spinner}>
          <Spinner />
        </span>
      ) : (
        <>
          {orgKeyError ? (
            <ErrorStub
              title="Error 404"
              message="Unexpected error occurred. Please try again later"
            />
          ) : (
            <>
              <SetupStep classes={classes} />
              <p className={classes.title}>
                2. Install Ansible on your local machine (such as a laptop)
              </p>
              <SyntaxHighlight content={getAnsibleInstallationCommand()} />
              <span className={classes.marginBottom}>
                For additional instructions on installing Ansible, please refer
                to{' '}
                <a
                  target="_blank"
                  href="https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html"
                  rel="noreferrer"
                >
                  this guide
                </a>
                .
              </span>
              <p className={classes.title}>
                3. Clone the "dle-se-ansible" repository to your local machine
              </p>
              <SyntaxHighlight content={cloneRepositoryCommand()} />
              <p className={classes.title}>4. Install necessary dependencies</p>
              <SyntaxHighlight
                content={'ansible-galaxy install -r requirements.yml'}
              />{' '}
              <p className={classes.title}>
                5. Execute the Ansible playbook to install DBLab SE on the remote
                server
              </p>
              <p>
                Replace{' '}
                <code className={classes.code}>'user@server-ip-address'</code>
                with the specific username and IP address of the server where
                you will be installing DBLab.
              </p>
              <SyntaxHighlight
                content={getPlaybookCommand(state, orgKey, false)}
              />
              <p className={classes.important}>Please be aware:</p>
              <p>
                The script will attempt to automatically detect an empty volume
                by default. If needed, you can manually specify the path to your
                desired empty disk using the{' '}
                <code className={classes.code}>zpool_disk</code>&nbsp; variable
                (e.g.,{' '}
                <code className={classes.code}>zpool_disk=/dev/nvme1n1</code>).
              </p>
              <p className={classes.title}>
                7. After the code snippet runs successfully, follow the
                directions displayed in the resulting output to start using DBLab
                UI/API/CLI.
              </p>{' '}
              <Box
                sx={{
                  display: 'flex',
                  gap: '10px',
                  margin: '20px 0',
                }}
              >
                <Button
                  variant="outlined"
                  color="secondary"
                  onClick={goBackToForm}
                >
                  Back to form
                </Button>
                <Button variant="contained" color="primary" onClick={goBack}>
                  See list of instances
                </Button>
              </Box>
            </>
          )}
        </>
      )}
    </InstanceFormCreation>
  )
}
