import { CloudImage } from 'api/cloud/getCloudImages'
import { CloudRegion } from 'api/cloud/getCloudRegions'
import { CloudVolumes } from 'api/cloud/getCloudVolumes'
import { addIndentForDocker } from 'components/PostgresClusterInstallForm/utils'
import { ClassesType } from 'components/types'
import { useCloudProviderProps } from 'hooks/useCloudProvider'

const API_SERVER = process.env.REACT_APP_API_SERVER
export const DEBUG_API_SERVER = 'https://v2.postgres.ai/api/general'

export const availableTags = ['3.5.0', '3.4.0', '4.0.0-beta.0']

export const sePackageTag = 'v1.4'

export const dockerRunCommand = (provider: string) => {
  switch (provider) {
    case 'aws':
      return `docker run --rm -it \\\r
  --env AWS_ACCESS_KEY_ID=\${AWS_ACCESS_KEY_ID} \\\r
  --env AWS_SECRET_ACCESS_KEY=\${AWS_SECRET_ACCESS_KEY}`

    case 'gcp':
      return `docker run --rm -it \\\r
  --env GCP_SERVICE_ACCOUNT_CONTENTS=\${GCP_SERVICE_ACCOUNT_CONTENTS}`

    case 'hetzner':
      return `docker run --rm -it \\\r
  --env HCLOUD_API_TOKEN=\${HCLOUD_API_TOKEN}`

    case 'digitalocean':
      return `docker run --rm -it \\\r
  --env DO_API_TOKEN=\${DO_API_TOKEN}`

    default:
      throw new Error('Provider is not supported')
  }
}

export const getPlaybookCommand = (
  state: useCloudProviderProps['initialState'],
  cloudImages: CloudImage,
  orgKey: string,
  isDocker?: boolean,
) =>{
  const playbookCommand = `ansible-playbook deploy_dle.yml --extra-vars \\\r
  "provision='${state.provider}' \\\r
  server_name='${state.name}' \\\r
  server_type='${state.instanceType.native_name}' \\\r
  server_image='${cloudImages?.native_os_image}' \\\r
  server_location='${state.location.native_code}' \\\r
  volume_size='${state.storage}' \\\r
  dblab_engine_verification_token='${state.verificationToken}' \\\r
  dblab_engine_version='${state.tag}' \\\r
  ${ state.snapshots > 1 ? `zpool_datasets_number='${state.snapshots}' \\\r` : `` }
  ${ orgKey ? `platform_org_key='${orgKey}' \\\r` : `` }
  ${ API_SERVER === DEBUG_API_SERVER ? `platform_url='${DEBUG_API_SERVER}' \\\r` : `` }
  ${state.publicKeys ? `ssh_public_keys='${state.publicKeys}' \\\r` : ``}
  platform_project_name='${state.name}'"`

  if (isDocker) {
    return `${dockerRunCommand(state.provider)} \\\r
    postgresai/dle-se-ansible:${sePackageTag} \\\r
      ${addIndentForDocker(playbookCommand)}`
  } else {
    return playbookCommand
  }
}

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

export const getNetworkSubnet = (provider: string, classNames: ClassesType) => {
  const AnchorElement = ({
    href,
    text = 'created',
  }: {
    href: string
    text?: string
  }) => (
    <a
      href={href}
      target="blank"
      rel="noopener noreferrer"
      className={classNames.link}
    >
      {text}
    </a>
  )

  const NetworkSubnet = ({
    note,
    code,
    optionalText,
  }: {
    note: React.ReactNode
    code: string
    optionalText: React.ReactNode
  }) => (
    <div className={classNames.containerMargin}>
      <p>Additional parameters:</p>
      <ul className={classNames.smallMarginTop}>
        <li>
          <code className={classNames.code}>{code}</code> {optionalText} {note}.
        </li>
      </ul>
    </div>
  )

  switch (provider) {
    case 'aws':
      return (
        <NetworkSubnet
          note={
            <>
              {' '}
              <AnchorElement
                href="https://docs.aws.amazon.com/vpc/latest/userguide/default-vpc.html"
                text="(More about default VPCs)"
              />
            </>
          }
          code="server_network='subnet-xxx'"
          optionalText={
            <>
              (optional) Subnet ID. The VM will use this subnet ({' '}
              <AnchorElement
                href="https://docs.aws.amazon.com/vpc/latest/userguide/create-subnets.html"
                text="must be created in advance"
              />{' '}
              in the selected region). If not specified, default VPC and subnet
              will be used.
            </>
          }
        />
      )
    case 'hetzner':
      return (
        <NetworkSubnet
          note={
            <>
              Public network is always attached; this is needed to access the
              server the installation process. The variable 'server_network' is
              used to define an additional network will be attached
            </>
          }
          code="server_network='network-xx'"
          optionalText={
            <>
              (optional) Subnet ID. The VM will use this network ({' '}
              <AnchorElement
                href="https://docs.hetzner.com/cloud/networks/getting-started/creating-a-network/"
                text="must be created in advance"
              />{' '}
              in the selected region). If not specified, no additional networks
              will be used.
            </>
          }
        />
      )
    case 'digitalocean':
      return (
        <NetworkSubnet
          note={
            <>
              {' '}
              <AnchorElement
                href="https://docs.digitalocean.com/products/networking/vpc/"
                text="(More about VPCs)"
              />
            </>
          }
          code="server_network='vpc-name'"
          optionalText={
            <>
              (optional) VPC name. The droplet will use this VPC ({' '}
              <AnchorElement
                href="https://docs.digitalocean.com/products/networking/vpc/how-to/create/"
                text="must be created in advance"
              />{' '}
              in the selected region). If not specified, default VPC will be
              used.
            </>
          }
        />
      )
    case 'gcp':
      return (
        <NetworkSubnet
          note={
            <>
              {' '}
              <AnchorElement
                href="https://cloud.google.com/vpc/docs/vpc"
                text="(More about VPCs)"
              />
            </>
          }
          code="server_network='vpc-network-name'"
          optionalText={
            <>
              optional) VPC network name. The VM will use this network ({' '}
              <AnchorElement
                href="https://cloud.google.com/vpc/docs/create-modify-vpc-networks"
                text="must be created in advance"
              />{' '}
              in the selected region). If not specified, default VPC and network
              will be used.
            </>
          }
        />
      )
  }
}

export const uniqueRegionsByProvider = (cloudRegions: CloudRegion[]) => {
  return cloudRegions
    .map((region) => region.world_part)
    .filter((value, index, self) => self.indexOf(value) === index)
}

export const filteredRegions = (
  cloudRegions: CloudRegion[],
  selectedRegion: string,
) => {
  return cloudRegions.filter((region) => region.world_part === selectedRegion)
}

export const formatVolumeDetails = (
  ssdCloudVolume: CloudVolumes,
  storage: number,
) => ({
  volumeType: `${ssdCloudVolume.api_name} (${ssdCloudVolume.cloud_provider}: ${ssdCloudVolume.native_name})`,
  volumeCurrency: ssdCloudVolume.native_reference_price_currency,
  volumePricePerHour:
    ssdCloudVolume.native_reference_price_per_1000gib_per_hour,
  volumePrice:
    (storage * ssdCloudVolume.native_reference_price_per_1000gib_per_hour) /
    1000,
})
