import { CloudImage } from 'api/cloud/getCloudImages'
import { initialState } from '../reducer'

const API_SERVER = process.env.REACT_APP_API_SERVER
const DEBUG_API_SERVER = 'https://v2.postgres.ai/api/general'

export const availableTags = ['3.4.0-rc.5', '4.0.0-alpha.5']

export const dockerRunCommand = (provider: string) => {
  /* eslint-disable no-template-curly-in-string */
  switch (provider) {
    case 'aws':
      return 'docker run --rm -it --env AWS_ACCESS_KEY_ID=${AWS_ACCESS_KEY_ID} --env AWS_SECRET_ACCESS_KEY=${AWS_SECRET_ACCESS_KEY}'
    case 'gcp':
      return 'docker run --rm -it --env GCP_SERVICE_ACCOUNT_CONTENTS=${GCP_SERVICE_ACCOUNT_CONTENTS}'
    case 'hetzner':
      return 'docker run --rm -it --env HCLOUD_API_TOKEN=${HCLOUD_API_TOKEN}'
    case 'digitalocean':
      return 'docker run --rm -it --env DO_API_TOKEN=${DO_API_TOKEN}'
    default:
      throw new Error('Provider is not supported')
  }
}

export const getPlaybookCommand = (
  state: typeof initialState,
  cloudImages: CloudImage,
  orgKey: string,
) =>
  `${dockerRunCommand(state.provider)} \\\r
  postgresai/dle-se-ansible:v1.0-rc.1 \\\r
    ansible-playbook deploy_dle.yml --extra-vars \\\r
    "provision='${state.provider}' \\\r
    server_name='${state.name}' \\\r
    server_type='${state.instanceType.native_name}' \\\r
    server_image='${cloudImages?.native_os_image}' \\\r
    server_location='${state.location.native_code}' \\\r
    volume_size='${state.storage}' \\\r
    dle_verification_token='${state.verificationToken}' \\\r
    dle_version='${state.tag}' \\\r
    ${
      state.snapshots > 1
        ? `zpool_datasets_number='${state.snapshots}' \\\r`
        : ``
    }
    ${orgKey ? `dle_platform_org_key='${orgKey}' \\\r` : ``}
    ${
      API_SERVER === DEBUG_API_SERVER
        ? `dle_platform_url='${DEBUG_API_SERVER}' \\\r`
        : ``
    }
    ${state.publicKeys ? `ssh_public_keys='${state.publicKeys}' \\\r` : ``}
    dle_platform_project_name='${state.name}'"`

export const getPlaybookCommandWithoutDocker = (
  state: typeof initialState,
  cloudImages: CloudImage,
  orgKey: string,
) =>
  `ansible-playbook deploy_dle.yml --extra-vars \\\r
  "provision='${state.provider}' \\\r
  server_name='${state.name}' \\\r
  server_type='${state.instanceType.native_name}' \\\r
  server_image='${cloudImages?.native_os_image}' \\\r
  server_location='${state.location.native_code}' \\\r
  volume_size='${state.storage}' \\\r
  dle_verification_token='${state.verificationToken}' \\\r
  dle_version='${state.tag}' \\\r
  ${
    state.snapshots > 1 ? `zpool_datasets_number='${state.snapshots}' \\\r` : ``
  }
  ${orgKey ? `dle_platform_org_key='${orgKey}' \\\r` : ``}
  ${
    API_SERVER === DEBUG_API_SERVER
      ? `dle_platform_url='${DEBUG_API_SERVER}' \\\r`
      : ``
  }
  ${state.publicKeys ? `ssh_public_keys='${state.publicKeys}' \\\r` : ``}
  dle_platform_project_name='${state.name}'"`

export const getGcpAccountContents = () =>
  `export GCP_SERVICE_ACCOUNT_CONTENTS='{
  "type": "service_account",
  "project_id": "my-project",
  "private_key_id": "c764349XXXXXXXXXX72f",
  "private_key": "XXXXXXXXXX",
  "client_email": "my-sa@my-project.iam.gserviceaccount.com",
  "client_id": "111111112222222",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/my-sat%40my-project.iam.gserviceaccount.com"
}'`

export const cloudProviderName = (provider: string) => {
  switch (provider) {
    case 'aws':
      return 'AWS'
    case 'gcp':
      return 'GCP'
    case 'digitalocean':
      return 'DigitalOcean'
    case 'hetzner':
      return 'Hetzner'
    default:
      return provider
  }
}

export const pricingPageForProvider = (provider: string) => {
  switch (provider) {
    case 'aws':
      return 'https://instances.vantage.sh/'
    case 'gcp':
      return 'https://cloud.google.com/compute/docs/general-purpose-machines'
    case 'digitalocean':
      return 'https://www.digitalocean.com/pricing/droplets'
    case 'hetzner':
      return 'https://www.hetzner.com/cloud'
  }
}
