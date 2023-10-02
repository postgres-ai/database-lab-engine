import { Box } from '@mui/material'
import {
  Button,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@material-ui/core'

import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { formStyles } from 'components/DbLabInstanceForm/DbLabFormSteps/AnsibleInstance'
import { InstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/InstanceFormCreation'
import { initialState } from '../reducer'
import {
  getAnsibleClusterCommand,
  getClusterExampleCommand,
  getPostgresClusterInstallationCommand,
} from '../utils'
import { getAnsibleInstallationCommand } from 'components/DbLabInstanceInstallForm/utils'
import { icons } from '@postgres.ai/shared/styles/icons'

export const AnsibleInstance = ({
  state,
  goBack,
  goBackToForm,
  formStep,
  setFormStep,
}: {
  state: typeof initialState
  goBack: () => void
  goBackToForm: () => void
  formStep: string
  setFormStep: (step: string) => void
}) => {
  const classes = formStyles()

  return (
    <InstanceFormCreation formStep={formStep} setFormStep={setFormStep} install>
      <>
        <p className={classes.title}>1. Set up your machine</p>
        <ul className={classes.ul}>
          <li>
            Obtain a machine running on one of the supported Linux
            distributions: Ubuntu 20.04/22.04, Debian 11/12, CentOS Stream 8/9,
            Rocky Linux 8/9, AlmaLinux 8/9, or Red Hat Enterprise Linux 8/9.
          </li>
          <li>
            (Recommended) Attach and mount the disk for the data directory.
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
          For additional instructions on installing Ansible, please refer to{' '}
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
          3. Clone the postgresql_cluster repository
        </p>
        <SyntaxHighlight content={getPostgresClusterInstallationCommand()} />
        <p className={classes.title}>4. Prepare the inventory file</p>
        <ul className={classes.ul}>
          <li>
            Specify private IP addresses (non-public) and connection settings ({' '}
            <code className={classes.code}>ansible_user</code>,
            <code className={classes.code}>ansible_ssh_private_key_file</code>{' '}
            or <code className={classes.code}>ansible_ssh_pass</code> for your
            environment.
          </li>
          <li>
            For deploying via public IPs, add '{' '}
            <code className={classes.code}>ansible_host=public_ip_address</code>
            ' variable for each node.
          </li>
        </ul>
        <SyntaxHighlight content={`nano inventory`} />
        <Box
          sx={{
            marginBottom: '35px',
          }}
        >
          <Accordion className={classes.marginBottom}>
            <AccordionSummary
              aria-controls="panel1a-content"
              id="panel1a-header"
              expandIcon={icons.sortArrowDown}
            >
              Example
            </AccordionSummary>
            <AccordionDetails>
              <SyntaxHighlight content={getClusterExampleCommand()} />
            </AccordionDetails>
          </Accordion>
        </Box>
        <p className={classes.title}>
          5. Execute the Ansible playbook to Postgres Cluster
        </p>
        <SyntaxHighlight wrapLines content={getAnsibleClusterCommand(state)} />
        <p className={classes.title}>
          6. After the code snippet runs successfully, follow the directions
          displayed in the resulting output to start using the database.
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
            See list of clusters
          </Button>
        </Box>
      </>
    </InstanceFormCreation>
  )
}
