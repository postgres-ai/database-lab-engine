import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Button,
} from '@material-ui/core'
import { Box } from '@mui/material'

import { SyntaxHighlight } from '@postgres.ai/shared/components/SyntaxHighlight'
import { icons } from '@postgres.ai/shared/styles/icons'
import { formStyles } from 'components/DbLabInstanceForm/DbLabFormSteps/AnsibleInstance'
import { InstanceFormCreation } from 'components/DbLabInstanceForm/DbLabFormSteps/InstanceFormCreation'
import { useCloudProviderProps } from 'hooks/useCloudProvider'
import {
  getClusterCommand,
  getClusterExampleCommand,
  getInventoryPreparationCommand,
} from '../utils'

export const DockerInstance = ({
  state,
  goBack,
  goBackToForm,
  formStep,
  setFormStep,
}: {
  state: useCloudProviderProps['initialState']
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
        <p className={classes.title}>2. Prepare the inventory file</p>
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
        <SyntaxHighlight content={getInventoryPreparationCommand()} />
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
          3. Run ansible playbook to deploy Postgres Cluster
        </p>
        <SyntaxHighlight wrapLines content={getClusterCommand(state, true)} />
        <p className={classes.title}>
          4. After the code snippet runs successfully, follow the directions
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
