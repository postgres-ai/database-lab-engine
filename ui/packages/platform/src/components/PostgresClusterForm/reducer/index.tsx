/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { ReducerAction } from 'react'

import { CloudRegion } from 'api/cloud/getCloudRegions'
import { CloudInstance } from 'api/cloud/getCloudInstances'
import { CloudVolumes } from 'api/cloud/getCloudVolumes'

export const initialState = {
  isLoading: false,
  isReloading: false,
  formStep: 'create',
  provider: 'aws',
  storage: 100,
  region: 'North America',
  version: '16',
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
}

export const reducer = (
  state: typeof initialState,
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
    case 'change_plan': {
      return {
        ...state,
        plan: action.plan,
        size: action.size,
      }
    }
    case 'change_size': {
      return {
        ...state,
        size: action.size,
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
    case 'set_is_loading': {
      return {
        ...state,
        isLoading: action.isLoading,
      }
    }
    case 'set_is_reloading': {
      return {
        ...state,
        isReloading: action.isReloading,
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
    
    default:
      throw new Error()
  }
}
