import { DEBUG_API_SERVER, sePackageTag } from 'components/DbLabInstanceForm/utils'
import { addIndentForDocker } from 'components/PostgresClusterInstallForm/utils'
import { useCloudProviderProps } from 'hooks/useCloudProvider'

const API_SERVER = process.env.REACT_APP_API_SERVER

export const getPlaybookCommand = (
  state: useCloudProviderProps['initialState'],
  orgKey: string,
  isDocker?: boolean,
) => {
  const playbookCommand = `ansible-playbook deploy_dle.yml --extra-vars \\\r
  "dblab_engine_host='user@server-ip-address' \\\r
  platform_project_name='${state.name}' \\\r
  dblab_engine_version='${state.tag}' \\\r
  ${ orgKey ? `platform_org_key='${orgKey}' \\\r` : `` }
  ${ API_SERVER === DEBUG_API_SERVER ? `platform_url='${DEBUG_API_SERVER}' \\\r` : `` }
  ${ state.publicKeys ? `ssh_public_keys='${state.publicKeys}' \\\r` : `` }
  dblab_engine_verification_token='${state.verificationToken}'"`

  if (isDocker) {
    return `docker run --rm -it \\\r
  --volume $HOME/.ssh:/root/.ssh:ro \\\r
  --env ANSIBLE_SSH_ARGS="-F none" \\\r
  postgresai/dle-se-ansible:${sePackageTag} \\\r
      ${addIndentForDocker(playbookCommand)}`
  } else {
    return playbookCommand
  }
}

export const getAnsibleInstallationCommand = () =>
  `sudo apt update
sudo apt install -y python3-pip
pip3 install ansible`

export const cloneRepositoryCommand = () =>
  `git clone https://gitlab.com/postgres-ai/dle-se-ansible.git
# Go to the playbook directory
cd dle-se-ansible/
`
