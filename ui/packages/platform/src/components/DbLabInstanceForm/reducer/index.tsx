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

import { availableTags } from 'components/DbLabInstanceForm/utils'
import { clusterExtensionsState } from 'components/PostgresClusterInstallForm/reducer'

export const initialState = {
  isLoading: false,
  isReloading: false,
  formStep: 'create',
  provider: 'aws',
  storage: 30,
  region: 'North America',
  tag: availableTags[0],
  serviceProviders: [] as string[],
  cloudRegions: [] as CloudRegion[],
  cloudInstances: [] as CloudInstance[],
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
  publicKeys: '',
  verificationToken: '',
  numberOfInstances: 3,
  version: 16,
  database_public_access: false,
  with_haproxy_load_balancing: false,
  pgbouncer_install: true,
  synchronous_mode: false,
  synchronous_node_count: 1,
  netdata_install: true,
  taskID: '',
  fileSystem: 'zfs',
  ...clusterExtensionsState,
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
        databaseSize: initialState.databaseSize,
        snapshots: initialState.snapshots,
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
        isReloading: action.isReloading,
        databaseSize: initialState.databaseSize,
        snapshots: initialState.snapshots,
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
    case 'change_verification_token': {
      return {
        ...state,
        verificationToken: action.verificationToken,
      }
    }
    case 'change_public_keys': {
      return {
        ...state,
        publicKeys: action.publicKeys,
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
    case 'change_snapshots': {
      return {
        ...state,
        snapshots: action.snapshots,
        storage: action.storage,
        volumePrice: action.volumePrice,
      }
    }

    case 'change_volume_price': {
      return {
        ...state,
        volumePrice: action.volumePrice,
        databaseSize: action.databaseSize,
        storage: action.storage,
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
    case 'set_tag': {
      return {
        ...state,
        tag: action.tag,
      }
    }
    default:
      throw new Error()
  }
}
