import { CloudImage } from 'api/cloud/getCloudImages';
import { DEBUG_API_SERVER, dockerRunCommand } from "components/DbLabInstanceForm/utils";
import { addIndentForAnsible, addIndentForDocker, clusterExtensions } from "components/PostgresClusterInstallForm/utils";
import { useCloudProviderProps } from "hooks/useCloudProvider";

const API_SERVER = process.env.REACT_APP_API_SERVER

const isDebugServer = API_SERVER === DEBUG_API_SERVER

export const getClusterPlaybookCommand = (
  state: useCloudProviderProps["initialState"],
  cloudImages: CloudImage,
  orgKey: string,
  isDocker?: boolean,
) => {
  const playbookVariables = `ansible-playbook deploy_pgcluster.yml --extra-vars \\\r
  "ansible_user='${state.provider === "aws" ? 'ubuntu' : 'root'}' \\\r
  provision='${state.provider}' \\\r
  servers_count='${state.numberOfInstances}' \\\r
  server_type='${state.instanceType.native_name}' \\\r
  server_image='${cloudImages?.native_os_image}' \\\r
  server_location='${state.location.native_code}' \\\r
  volume_size='${state.storage}' \\\r
  postgresql_version='${state.version}' \\\r
  database_public_access='${state.database_public_access}' \\\r
  with_haproxy_load_balancing='${state.with_haproxy_load_balancing}' \\\r
  pgbouncer_install='${state.pgbouncer_install}' \\\r
  pg_data_mount_fstype='${state.fileSystem}' \\\r
  synchronous_mode='${state.synchronous_mode}' \\\r
  ${state.synchronous_mode ? `synchronous_node_count='${state.synchronous_node_count}' \\\r` : ``}
  netdata_install='${state.netdata_install}' \\\r
  patroni_cluster_name='${state.name}' \\\r
  ${addIndentForAnsible(clusterExtensions(state as unknown as {[key: string]: boolean | string | number} ))}
  ${state.publicKeys ? `ssh_public_keys='${state.publicKeys}' \\\r` : ``}
  ${orgKey ? `platform_org_key='${orgKey}'${isDebugServer ? `\\\r` : `"`}` : ``}
  ${isDebugServer ? `platform_url='${DEBUG_API_SERVER}'"` : ``}`

  if (isDocker) {
    return `${dockerRunCommand(state.provider)} \\\r
    vitabaks/postgresql_cluster:cloud \\\r
      ${addIndentForDocker(playbookVariables)}`
  }
  return playbookVariables
}

export const cloneClusterRepositoryCommand = () =>
  `git clone --depth 1 --branch cloud \\\r
  https://github.com/vitabaks/postgresql_cluster.git \\\r
&& cd postgresql_cluster/`
