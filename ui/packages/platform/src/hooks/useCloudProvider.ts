import { Reducer, useCallback, useEffect, useReducer } from 'react'

import { getCloudInstances } from 'api/cloud/getCloudInstances'
import { getCloudProviders } from 'api/cloud/getCloudProviders'
import { getCloudRegions } from 'api/cloud/getCloudRegions'
import { CloudVolumes, getCloudVolumes } from 'api/cloud/getCloudVolumes'
import { formatVolumeDetails } from 'components/DbLabInstanceForm/utils'
import { generateToken } from 'utils/utils'

interface State {
  [key: string]: StateValue
}

type StateValue =
  | string
  | number
  | boolean
  | undefined
  | unknown
  | { [key: string]: string }

export interface useCloudProviderProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  initialState: any
  reducer: Reducer<useCloudProviderProps['initialState'], State>
}

export const useCloudProvider = ({
  initialState,
  reducer,
}: useCloudProviderProps) => {
  const [state, dispatch] = useReducer(reducer, initialState)

  const urlParams = new URLSearchParams(window.location.search)
  const urlTaskID = urlParams.get('taskID')
  const urlProvider = urlParams.get('provider')

  const fetchCloudData = useCallback(
    async (provider: string) => {
      const cloudRegions = await getCloudRegions(provider)
      const cloudVolumes = await getCloudVolumes(provider)
      const ssdCloudVolumes = cloudVolumes.response.find(
        (volume: CloudVolumes) => volume.api_name === initialState?.api_name,
      )

      return {
        cloudRegions: cloudRegions.response,
        cloudVolumes: cloudVolumes.response,
        ssdCloudVolume: ssdCloudVolumes,
      }
    },
    [initialState.api_name],
  )

  useEffect(() => {
    const action = {
      type: 'set_form_step',
      formStep: urlTaskID && urlProvider ? 'simple' : initialState.formStep,
      taskID: urlTaskID || undefined,
      provider: urlProvider || initialState.provider,
    }

    dispatch(action)
  }, [urlTaskID, urlProvider, initialState.formStep, initialState.provider])

  useEffect(() => {
    const fetchInitialCloudDetails = async () => {
      try {
        const { cloudRegions, cloudVolumes, ssdCloudVolume } =
          await fetchCloudData(initialState.provider as string)
        const volumeDetails = formatVolumeDetails(
          ssdCloudVolume,
          initialState.storage as number,
        )
        const serviceProviders = await getCloudProviders()

        dispatch({
          type: 'set_initial_state',
          cloudRegions,
          volumes: cloudVolumes,
          ...volumeDetails,
          serviceProviders: serviceProviders.response,
          isLoading: false,
        })
      } catch (error) {
        console.error(error)
      }
    }

    fetchInitialCloudDetails()
  }, [initialState.provider, initialState.storage, fetchCloudData])

  useEffect(() => {
    const fetchUpdatedCloudDetails = async () => {
      try {
        const { cloudRegions, cloudVolumes, ssdCloudVolume } =
          await fetchCloudData(state.provider)
        const volumeDetails = formatVolumeDetails(
          ssdCloudVolume,
          initialState.storage as number,
        )

        dispatch({
          type: 'update_initial_state',
          cloudRegions,
          volumes: cloudVolumes,
          ...volumeDetails,
        })
      } catch (error) {
        console.error(error)
      }
    }

    fetchUpdatedCloudDetails()
  }, [state.api_name, state.provider, initialState.storage, fetchCloudData])

  useEffect(() => {
    if (state.location.native_code && state.provider) {
      const fetchUpdatedDetails = async () => {
        try {
          const cloudInstances = await getCloudInstances({
            provider: state.provider,
            region: state.location.native_code,
          })

          dispatch({
            type: 'update_instance_type',
            cloudInstances: cloudInstances.response,
            instanceType: cloudInstances.response[0],
            isReloading: false,
          })
        } catch (error) {
          console.log(error)
        }
      }
      fetchUpdatedDetails()
    }
  }, [state.location.native_code, state.provider])

  const handleReturnToForm = () => {
    dispatch({ type: 'set_form_step', formStep: initialState.formStep })
  }

  const handleSetFormStep = (step: string) => {
    dispatch({ type: 'set_form_step', formStep: step })
  }

  const handleGenerateToken = () => {
    dispatch({
      type: 'change_verification_token',
      verificationToken: generateToken(),
    })
  }

  const handleChangeVolume = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const volumeApiName = event.target.value.split(' ')[0]
    const selectedVolume = state.volumes.filter(
      (volume: CloudVolumes) => volume.api_name === volumeApiName,
    )[0]

    dispatch({
      type: 'change_volume_type',
      volumeType: event.target.value,
      volumePricePerHour:
        selectedVolume.native_reference_price_per_1000gib_per_hour,
      volumePrice:
        (state.storage *
          selectedVolume.native_reference_price_per_1000gib_per_hour) /
        1000,
    })
  }

  return {
    state,
    dispatch,
    handleReturnToForm,
    handleSetFormStep,
    handleGenerateToken,
    handleChangeVolume,
  }
}
