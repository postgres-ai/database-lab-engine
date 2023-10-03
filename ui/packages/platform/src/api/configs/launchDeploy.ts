import { simpleInstallRequest } from 'helpers/simpleInstallRequest'
import { initialState } from 'components/DbLabInstanceForm/reducer'
import { DEBUG_API_SERVER, sePackageTag } from 'components/DbLabInstanceForm/utils'

const API_SERVER = process.env.REACT_APP_API_SERVER

const formatExtraEnvs = (extraEnvs: { [key: string]: string }) => {
  return Object.entries(extraEnvs)
    .filter(([key, value]) => value)
    .map(([key, value]) => {
      if (key === 'GCP_SERVICE_ACCOUNT_CONTENTS') {
        return `${key}=${value.replace(/\n\s+/g, '')}`
      }
      return `${key}=${value}`
    })
}

export const launchDeploy = async ({
  state,
  userID,
  orgKey,
  extraEnvs,
  cloudImage,
  launchType,
}: {
  state: typeof initialState
  orgKey: string
  userID?: number
  extraEnvs: {
    [key: string]: string
  }
  cloudImage: string
  launchType: 'cluster' | 'instance'
}) => {
  const instanceBody = {
    playbook: 'deploy_dle.yml',
    provision: state.provider,
    server: {
      name: state.name,
      serverType: state.instanceType.native_name,
      image: cloudImage,
      location: state.location.native_code,
    },
    image: `postgresai/dle-se-ansible:${sePackageTag}`,
    extraVars: [
      `provision=${state.provider}`,
      `server_name=${state.name}`,
      `platform_project_name=${state.name}`,
      `server_type=${state.instanceType.native_name}`,
      `server_image=${cloudImage}`,
      `server_location=${state.location.native_code}`,
      `volume_size=${state.storage}`,
      `dblab_engine_version=${state.tag}`,
      `zpool_datasets_number=${state.snapshots}`,
      `dblab_engine_verification_token=${state.verificationToken}`,
      `platform_org_key=${orgKey}`,
      ...(state.publicKeys
        ? // eslint-disable-next-line no-useless-escape
          [`ssh_public_keys=\"${state.publicKeys}\"`]
        : []),
      ...(API_SERVER === DEBUG_API_SERVER
        ? [`platform_url=https://v2.postgres.ai/api/general`]
        : []),
    ],
    extraEnvs: formatExtraEnvs(extraEnvs),
  }

  const clusterBody = {
    playbook: 'deploy_pgcluster.yml',
    provision: state.provider,
    server: {
      name: state.name,
      serverType: state.instanceType.native_name,
      image: cloudImage,
      location: state.location.native_code,
    },
    image: 'vitabaks/postgresql_cluster:cloud',
    extraVars: [
      `ansible_user=${state.provider === "aws" ? 'ubuntu' : 'root'}`,
      `provision=${state.provider}`,
      `servers_count=${state.numberOfInstances}`,
      `server_type=${state.instanceType.native_name}`,
      `server_image=${cloudImage}`,
      `server_location=${state.location.native_code}`,
      `volume_size=${state.storage}`,
      `postgresql_version=${state.version}`,
      `database_public_access=${state.database_public_access}`,
      `database_public_access=${state.database_public_access}`,
      `with_haproxy_load_balancing=${state.with_haproxy_load_balancing}`,
      `pgbouncer_install=${state.pgbouncer_install}`,
      `synchronous_mode=${state.synchronous_mode}`,
      ...(state.synchronous_mode ? [`synchronous_node_count=${state.synchronous_node_count}`] : []),
      `netdata_install=${state.netdata_install}`,
      `patroni_cluster_name=${state.name}`,
      `platform_org_key=${orgKey}`,
      ...(state.publicKeys
        ? // eslint-disable-next-line no-useless-escape
          [`ssh_public_keys=\"${state.publicKeys}\"`]
        : []),
      ...(API_SERVER === DEBUG_API_SERVER
        ? [`platform_url=https://v2.postgres.ai/api/general`]
        : []),
    ],
    extraEnvs: formatExtraEnvs(extraEnvs),
  }

  const response = await simpleInstallRequest(
    '/launch',
    {
      method: 'POST',
      body: JSON.stringify(
        launchType === 'cluster' ? clusterBody : instanceBody,
      ),
    },
    userID,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
