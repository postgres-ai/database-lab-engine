import { useCloudProviderProps } from "hooks/useCloudProvider"

export const getInventoryPreparationCommand = () =>
  `curl -fsSL https://raw.githubusercontent.com/vitabaks/postgresql_cluster/master/inventory \\\r
  --output ~/inventory
nano ~/inventory`

export const addIndentForDocker = (command: string) =>
  command
    .split('\r\n')
    .map((line, index) => {
      if (index === 0) {
        return `${line}`
      } else {
        return `${' '.repeat(8)}${line.replace(/^\s+/, '')}`
      }
    })
    .join('\r\n')

export const addIndentForAnsible = (command: string) =>
command
  .split('\r\n')
  .map((line, index) => {
    if (index === 0) {
      return `${line.replace(/^\s+/, '')}`
    } else {
      return `${' '.repeat(2)}${line.replace(/^\s+/, '')}`
    }
  })
  .join('\r\n')

export const getClusterExampleCommand = () => 
`[etcd_cluster]
10.0.1.1 ansible_host=5.161.228.76
10.0.1.2 ansible_host=5.161.224.229
10.0.1.3 ansible_host=5.161.63.15

[consul_instances]

[balancers]
10.0.1.1 ansible_host=5.161.228.76
10.0.1.2 ansible_host=5.161.224.229
10.0.1.3 ansible_host=5.161.63.15

[master]
10.0.1.1 ansible_host=5.161.228.76 hostname=pgnode01

[replica]
10.0.1.2 ansible_host=5.161.224.229 hostname=pgnode02
10.0.1.3 ansible_host=5.161.63.15 hostname=pgnode03

[postgres_cluster:children]
master
replica

[pgbackrest]

[all:vars]
ansible_connection='ssh'
ansible_ssh_port='22'
ansible_user='root'
ansible_ssh_private_key_file=/root/.ssh/id_rsa

[pgbackrest:vars]
`

export const clusterExtensions = (state: {[key: string]: boolean | string | number}) => `${state.pg_repack ? `enable_pg_repack='${state.pg_repack}' \\\r` : ''}
${state.pg_cron ? `enable_pg_cron='${state.pg_cron}' \\\r` : ''}
${state.pgaudit ? `enable_pgaudit='${state.pgaudit}' \\\r` : ''}
${state.version !== 10 && state.pgvector ? `enable_pgvector='${state.pgvector}' \\\r` : ''}
${state.postgis ? `enable_postgis='${state.postgis}' \\\r` : ''}
${state.pgrouting ? `enable_pgrouting='${state.pgrouting}' \\\r` : ''}
${state.version !== 10 && state.version !== 11 && state.timescaledb ? `enable_timescaledb='${state.timescaledb}' \\\r` : ''}
${state.version !== 10 && state.citus ? `enable_citus='${state.citus}' \\\r` : ''}
${state.pg_partman ? `enable_pg_partman='${state.pg_partman}' \\\r` : ''}
${state.pg_stat_kcache ? `enable_pg_stat_kcache='${state.pg_stat_kcache}' \\\r` : ''}
${state.pg_wait_sampling ? `enable_pg_wait_sampling='${state.pg_wait_sampling}' \\\r` : ''}`

export const getClusterCommand = (
  state: useCloudProviderProps['initialState'],
  isDocker?: boolean,
) => {
  const playbookVariables = `ansible-playbook deploy_pgcluster.yml --extra-vars \\\r
  "postgresql_version='${state.version}' \\\r
  patroni_cluster_name='${state.patroni_cluster_name}' \\\r
  ${addIndentForAnsible(clusterExtensions(state))}
  ${state.cluster_vip ? `cluster_vip='${state.cluster_vip} \\\r'` : ''}
  with_haproxy_load_balancing='${state.with_haproxy_load_balancing}' \\\r
  pgbouncer_install='${state.pgbouncer_install}' \\\r
  synchronous_mode='${state.synchronous_mode}' \\\r
  ${state.synchronous_mode ? `synchronous_node_count='${state.synchronous_node_count}' \\\r` : ''}
  netdata_install='${state.netdata_install}' \\\r
  postgresql_data_dir='${state.postgresql_data_dir}'"`

  if (isDocker) {
    return `docker run --rm -it \\\r
    -v ~/inventory:/postgresql_cluster/inventory:ro \\\r
    -v $HOME/.ssh:/root/.ssh:ro -e ANSIBLE_SSH_ARGS="-F none" \\\r
    vitabaks/postgresql_cluster:cloud \\\r
      ${addIndentForDocker(playbookVariables)}`
  } 
  
  return playbookVariables
}

export const getPostgresClusterInstallationCommand = () => 
`git clone --depth 1 --branch cloud \\\r
  https://github.com/vitabaks/postgresql_cluster.git \\\r
  && cd postgresql_cluster/
`

export function isIPAddress(input: string) {
  if (input === '') {
    return true
  }

  const ipPattern =
    /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
  return ipPattern.test(input)
}