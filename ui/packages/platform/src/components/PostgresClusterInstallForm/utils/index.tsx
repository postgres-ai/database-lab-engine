import { initialState } from "../reducer"

export const getInventoryPreparationCommand = () =>
  `curl -fsSL https://raw.githubusercontent.com/vitabaks/postgresql_cluster/master/inventory \\\r
  --output ~/inventory
nano ~/inventory`

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

export const getClusterCommand = (
    state: typeof initialState,
  ) =>
    `docker run --rm -it \\\r
    -v ~/inventory:/postgresql_cluster/inventory:ro \\\r
    -v $HOME/.ssh:/root/.ssh:ro -e ANSIBLE_SSH_ARGS="-F none" \\\r
    vitabaks/postgresql_cluster:cloud \\\r
      ansible-playbook deploy_pgcluster.yml --extra-vars \\\r
        "postgresql_version='${state.version}' \\\r
        patroni_cluster_name='${state.patroni_cluster_name}' \\\r
        ${state.cluster_vip ? `cluster_vip='${state.cluster_vip} \\\r'` : ''}
        with_haproxy_load_balancing='${state.with_haproxy_load_balancing}' \\\r
        pgbouncer_install='${state.pgbouncer_install}' \\\r
        synchronous_mode='${state.synchronous_mode}' \\\r
        ${state.synchronous_mode ? `synchronous_node_count='${state.synchronous_node_count}' \\\r` : ''}
        netdata_install='${state.netdata_install}' \\\r
        postgresql_data_dir='${state.postgresql_data_dir}'"
`

export const getPostgresClusterInstallationCommand = () => 
`git clone --depth 1 --branch cloud \\\r
  https://github.com/vitabaks/postgresql_cluster.git \\\r
  && cd postgresql_cluster/
`

export const getAnsibleClusterCommand = (
    state: typeof initialState,
  ) =>
    `ansible-playbook deploy_pgcluster.yml --extra-vars \\\r
      "postgresql_version='${state.version}' \\\r
      patroni_cluster_name='${state.patroni_cluster_name}' \\\r
      ${state.cluster_vip ? `cluster_vip='${state.cluster_vip} \\\r'` : ''}
      with_haproxy_load_balancing='${state.with_haproxy_load_balancing}' \\\r
      pgbouncer_install='${state.pgbouncer_install}' \\\r
      synchronous_mode='${state.synchronous_mode}' \\\r
      ${state.synchronous_mode ? `synchronous_node_count='${state.synchronous_node_count}' \\\r` : ''}
      netdata_install='${state.netdata_install}' \\\r
      postgresql_data_dir='${state.postgresql_data_dir}'"
`

export function isIPAddress(input: string) {
  if (input === '') {
    return true
  }

  const ipPattern =
    /^(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
  return ipPattern.test(input)
}