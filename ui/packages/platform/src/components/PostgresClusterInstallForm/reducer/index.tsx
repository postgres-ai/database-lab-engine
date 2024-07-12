/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { ReducerAction } from 'react'

export const clusterExtensionsState = {
  pg_repack: true,
  pg_cron: true,
  pgaudit: true,
  pgvector: true,
  postgis: false,
  pgrouting: false,
  timescaledb: false,
  citus: false,
  pg_partman: false,
  pg_stat_kcache: false,
  pg_wait_sampling: false,
}

export const initialState = {
  isLoading: false,
  isReloading: false,
  formStep: 'create',
  patroni_cluster_name: '',
  version: 16,
  postgresql_data_dir: '/var/lib/postgresql/16/<cluster-name>',
  cluster_vip: '',
  with_haproxy_load_balancing: false,
  pgbouncer_install: true,
  synchronous_mode: false,
  synchronous_node_count: 1,
  netdata_install: true,
  taskID: '',
  ...clusterExtensionsState,
}

export const reducer = (
  state: typeof initialState,
  // @ts-ignore
  action: ReducerAction<unknown, void>,
) => {
  switch (action.type) {
    case 'change_patroni_cluster_name': {
      return {
        ...state,
        patroni_cluster_name: action.patroni_cluster_name,
        postgresql_data_dir: `/var/lib/postgresql/${state.version}/${
          action.patroni_cluster_name || '<cluster-name>'
        }`,
      }
    }

    case 'change_version': {
      return {
        ...state,
        version: action.version,
        postgresql_data_dir: `/var/lib/postgresql/${action.version}/${
          state.patroni_cluster_name || '<cluster-name>'
        }`,
      }
    }

    case 'change_postgresql_data_dir': {
      return {
        ...state,
        postgresql_data_dir: action.postgresql_data_dir,
      }
    }

    case 'change_cluster_vip': {
      return {
        ...state,
        cluster_vip: action.cluster_vip,
      }
    }

    case 'change_with_haproxy_load_balancing': {
      return {
        ...state,
        with_haproxy_load_balancing: action.with_haproxy_load_balancing,
      }
    }

    case 'change_pgbouncer_install': {
      return {
        ...state,
        pgbouncer_install: action.pgbouncer_install,
      }
    }

    case 'change_synchronous_mode': {
      return {
        ...state,
        synchronous_mode: action.synchronous_mode,
      }
    }

    case 'change_synchronous_node_count': {
      return {
        ...state,
        synchronous_node_count: action.synchronous_node_count,
      }
    }

    case 'change_netdata_install': {
      return {
        ...state,
        netdata_install: action.netdata_install,
      }
    }

    case 'set_form_step': {
      return {
        ...state,
        formStep: action.formStep,
      }
    }
    case 'change_pg_repack': {
      return {
        ...state,
        pg_repack: action.pg_repack,
      }
    }
    case 'change_pg_cron': {
      return {
        ...state,
        pg_cron: action.pg_cron,
      }
    }
    case 'change_pgaudit': {
      return {
        ...state,
        pgaudit: action.pgaudit,
      }
    }
    case 'change_pgvector': {
      return {
        ...state,
        pgvector: action.pgvector,
      }
    }
    case 'change_postgis': {
      return {
        ...state,
        postgis: action.postgis,
      }
    }
    case 'change_pgrouting': {
      return {
        ...state,
        pgrouting: action.pgrouting,
      }
    }
    case 'change_timescaledb': {
      return {
        ...state,
        timescaledb: action.timescaledb,
      }
    }
    case 'change_citus': {
      return {
        ...state,
        citus: action.citus,
      }
    }
    case 'change_pg_partman': {
      return {
        ...state,
        pg_partman: action.pg_partman,
      }
    }
    case 'change_pg_stat_kcache': {
      return {
        ...state,
        pg_stat_kcache: action.pg_stat_kcache,
      }
    }
    case 'change_pg_wait_sampling': {
      return {
        ...state,
        pg_wait_sampling: action.pg_wait_sampling,
      }
    }
    default:
      throw new Error()
  }
}
