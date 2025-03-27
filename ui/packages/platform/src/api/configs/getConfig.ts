import { formatConfig } from '@postgres.ai/shared/types/api/entities/config'
import { request } from 'helpers/request'

export const getConfig = async (instanceId: string) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: '/admin/config',
      method: 'get',
    }),
  })

  return {
    response: response.ok ? formatConfig(await response.json()) : null,
    error: response.ok ? null : response,
  }
}
