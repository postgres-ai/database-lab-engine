import {
  request as requestCore,
  RequestOptions,
} from '@postgres.ai/shared/helpers/request'

import { localStorage } from 'helpers/localStorage'
import { appStore } from 'stores/app'
import { API_URL_PREFIX } from 'config/env'

export const request = async (path: string, options?: RequestOptions) => {
  const authToken = localStorage.getAuthToken()

  const response = await requestCore(`${API_URL_PREFIX}${path}`, {
    ...options,
    headers: {
      ...(authToken && { 'Verification-Token': authToken }),
      ...options?.headers,
    },
  })

  if (response.status === 401) {
    appStore.setIsInvalidAuthToken()
    localStorage.removeAuthToken()
  } else {
    appStore.setIsValidAuthToken()
  }

  return response
}
