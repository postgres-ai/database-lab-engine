import {
  DEBUG_API_SERVER,
  sePackageTag,
} from 'components/DbLabInstanceForm/utils'
import { simpleInstallRequest } from 'helpers/simpleInstallRequest'
import { useCloudProviderProps } from 'hooks/useCloudProvider'

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
  state: useCloudProviderProps['initialState']
  orgKey: string
  userID?: number
  extraEnvs: {
    [key: string]: string
  }
  cloudImage: string
  launchType: 'cluster' | 'instance'
}) => {
  const instanceExtraVars = [
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
  ]

  const instanceOptionalVars = [
    state.publicKeys && `ssh_public_keys="${state.publicKeys}"`,
    API_SERVER === DEBUG_API_SERVER &&
      `platform_url=https://v2.postgres.ai/api/general`,
  ].filter(Boolean)

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
    extraVars: [...instanceExtraVars, ...instanceOptionalVars],
    extraEnvs: formatExtraEnvs(extraEnvs),
  }

  const user = state.provider === 'aws' ? 'ubuntu' : 'root'

  const extraVars = [
    `ansible_user=${user}`,
    `provision=${state.provider}`,
    `servers_count=${state.numberOfInstances}`,
    `server_type=${state.instanceType.native_name}`,
    `server_image=${cloudImage}`,
    `server_location=${state.location.native_code}`,
    `volume_size=${state.storage}`,
    `postgresql_version=${state.version}`,
    `database_public_access=${state.database_public_access}`,
    `with_haproxy_load_balancing=${state.with_haproxy_load_balancing}`,
    `pgbouncer_install=${state.pgbouncer_install}`,
    `pg_data_mount_fstype=${state.fileSystem}`,
    `synchronous_mode=${state.synchronous_mode}`,
    `netdata_install=${state.netdata_install}`,
    `patroni_cluster_name=${state.name}`,
    `platform_org_key=${orgKey}`,
  ]

  const optionalVars = [
    state.synchronous_mode &&
      `synchronous_node_count=${state.synchronous_node_count}`,
    state.pg_repack && `enable_pg_repack=${state.pg_repack}`,
    state.pg_cron && `enable_pg_cron=${state.pg_cron}`,
    state.pgaudit && `enable_pgaudit=${state.pgaudit}`,
    state.version !== 10 &&
      state.pgvector &&
      `enable_pgvector=${state.pgvector}`,
    state.postgis && `enable_postgis=${state.postgis}`,
    state.pgrouting && `enable_pgrouting=${state.pgrouting}`,
    state.version !== 10 &&
      state.version !== 11 &&
      state.timescaledb &&
      `enable_timescaledb=${state.timescaledb}`,
    state.version !== 10 && state.citus && `enable_citus=${state.citus}`,
    state.pg_partman && `enable_pg_partman=${state.pg_partman}`,
    state.pg_stat_kcache && `enable_pg_stat_kcache=${state.pg_stat_kcache}`,
    state.pg_wait_sampling &&
      `enable_pg_wait_sampling=${state.pg_wait_sampling}`,
    state.publicKeys && `ssh_public_keys="${state.publicKeys}"`,
    API_SERVER === DEBUG_API_SERVER &&
      `platform_url=https://v2.postgres.ai/api/general`,
  ].filter(Boolean)

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
    extraVars: [...extraVars, ...optionalVars],
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
