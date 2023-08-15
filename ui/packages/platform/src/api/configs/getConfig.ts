import { formatConfig } from '@postgres.ai/shared/types/api/entities/config'
import { request } from 'helpers/request'

export const getConfig = async () => {
  const response = await request('/admin/config')

  return {
    response: response.ok ? formatConfig(await response.json()) : null,
    error: response.ok ? null : response,
  }
}
