import { Box } from '@mui/material'
import { useEffect, useState } from 'react'
import { makeStyles, Button } from '@material-ui/core'

import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { Spinner } from '@postgres.ai/shared/components/Spinner'

import { initialState } from '../reducer'
import { DblabInstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/DbLabInstanceFormCreation'
import { getOrgKeys } from 'api/cloud/getOrgKeys'
import { ErrorStub } from '@postgres.ai/shared/components/ErrorStub'
import {
  cloneRepositoryCommand,
  getAnsibleInstallationCommand,
  getAnsiblePlaybookCommand,
} from 'components/DbLabInstanceInstallForm/utils'

const useStyles = makeStyles({
  marginTop: {
    marginTop: '20px',
  },
  marginBottom: {
    marginBottom: '10px',
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
  ul: {
    paddingInlineStart: '30px',
  },
  important: {
    fontWeight: 600,
    margin: 0,
  },
})

export const DblabInstanceSnippet = ({
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
              message="Unexpected error occurred. Please try again later"
            />
          ) : (
            <>
              <p className={classes.title}>1. Set up your machine</p>
              <ul className={classes.ul}>
                <li>
                  Obtain a machine running Ubuntu 22.04 (although other versions
                  may work, we recommend using an LTS version for optimal
                  compatibility).
                </li>
                <li>
                  Attach an empty disk that is at least twice the size of the
                  database you plan to use with DLE.
                </li>
                <li>
                  Ensure that your SSH public key is added to the machine (in
                  <code className={classes.code}>~/.ssh/authorized_keys</code>),
                  allowing for secure SSH access.
                </li>
              </ul>
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
                5. Execute the Ansible playbook to install DLE SE on the remote
                server
              </p>
              <p>
                Replace{' '}
                <code className={classes.code}>'user@server-ip-address'</code>
                with the specific username and IP address of the server where
                you will be installing DLE.
              </p>
              <SyntaxHighlight
                content={getAnsiblePlaybookCommand(state, orgKey)}
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
