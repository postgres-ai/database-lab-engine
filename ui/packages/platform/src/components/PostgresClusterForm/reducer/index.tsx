/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { ReducerAction } from 'react'

import { CloudInstance } from 'api/cloud/getCloudInstances'
import { CloudRegion } from 'api/cloud/getCloudRegions'
import { CloudVolumes } from 'api/cloud/getCloudVolumes'
import { clusterExtensionsState } from 'components/PostgresClusterInstallForm/reducer'
import { useCloudProviderProps } from 'hooks/useCloudProvider'

export const initialState = {
  isLoading: false,
  isReloading: false,
  formStep: 'create',
  provider: 'aws',
  storage: 100,
  region: 'North America',
  version: 16,
  serviceProviders: [],
  cloudRegions: [],
  cloudInstances: [],
  volumes: [] as CloudVolumes[],
  api_name: 'ssd',
  databaseSize: 10,
  snapshots: 3,
  volumeType: '',
  volumePrice: 0,
  volumePricePerHour: 0,
  volumeCurrency: '',
  location: {} as CloudRegion,
  instanceType: {} as CloudInstance,
  name: '',
  tag: '',
  numberOfInstances: 3,
  publicKeys: '',
  verificationToken: '',
  seImageVersions: [],
  database_public_access: false,
  with_haproxy_load_balancing: false,
  pgbouncer_install: true,
  synchronous_mode: false,
  synchronous_node_count: 1,
  netdata_install: true,
  taskID: '',
  fileSystem: 'zfs',
  fileSystemArray: ['zfs', 'ext4', 'xfs'],
  ...clusterExtensionsState,
}

export const reducer = (
  state: useCloudProviderProps['initialState'],
  // @ts-ignore
  action: ReducerAction<unknown, void>,
) => {
  switch (action.type) {
    case 'set_initial_state': {
      return {
        ...state,
        isLoading: action.isLoading,
        serviceProviders: action.serviceProviders,
        volumes: action.volumes,
        volumeType: action.volumeType,
        volumePrice: action.volumePrice,
        volumePricePerHour: action.volumePricePerHour,
        volumeCurrency: action.volumeCurrency,
        region: initialState.region,
        location: action.cloudRegions.find(
          (region: CloudRegion) =>
            region.world_part === initialState.region &&
            region.cloud_provider === initialState.provider,
        ),
      }
    }
    case 'update_initial_state': {
      return {
        ...state,
        volumes: action.volumes,
        volumeType: action.volumeType,
        volumePricePerHour: action.volumePricePerHour,
        volumeCurrency: action.volumeCurrency,
        cloudRegions: action.cloudRegions,
        location: action.cloudRegions.find(
          (region: CloudRegion) => region.world_part === initialState.region,
        ),
      }
    }
    case 'update_instance_type': {
      return {
        ...state,
        cloudInstances: action.cloudInstances,
        instanceType: action.instanceType,
        isReloading: action.isReloading,
      }
    }
    case 'change_provider': {
      return {
        ...state,
        provider: action.provider,
        region: initialState.region,
        storage: initialState.storage,
      }
    }
    case 'change_region': {
      return {
        ...state,
        region: action.region,
        location: action.location,
      }
    }
    case 'change_location': {
      return {
        ...state,
        location: action.location,
      }
    }

    case 'change_name': {
      return {
        ...state,
        name: action.name,
      }
    }
    case 'change_instance_type': {
      return {
        ...state,
        instanceType: action.instanceType,
      }
    }
    case 'change_public_keys': {
      return {
        ...state,
        publicKeys: action.publicKeys,
      }
    }
    case 'change_number_of_instances': {
      return {
        ...state,
        numberOfInstances: action.number,
      }
    }
    case 'change_volume_type': {
      return {
        ...state,
        volumeType: action.volumeType,
        volumePrice: action.volumePrice,
        volumePricePerHour: action.volumePricePerHour,
      }
    }

    case 'change_volume_price': {
      return {
        ...state,
        volumePrice: action.volumePrice,
        storage: action.volumeSize,
      }
    }

    case 'set_form_step': {
      return {
        ...state,
        formStep: action.formStep,
        ...(action.taskID ? { taskID: action.taskID } : {}),
        ...(action.provider ? { provider: action.provider } : {}),
      }
    }
    case 'set_version': {
      return {
        ...state,
        version: action.version,
      }
    }
    case 'change_database_public_access': {
      return {
        ...state,
        database_public_access: action.database_public_access,
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

    case 'change_file_system': {
      return {
        ...state,
        fileSystem: action.fileSystem,
      }
    }

    default:
      throw new Error()
  }
}
