import { request } from 'helpers/request'
import {
  EngineDto,
  formatEngineDto,
} from '@postgres.ai/shared/types/api/endpoints/getEngine'

export const getEngine = async (instanceId: string) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      action: '/healthz',
      instance_id: instanceId,
      method: 'get',
    }),
  })

  return {
    response: response.ok
      ? formatEngineDto((await response.json()) as EngineDto)
      : null,
    error: response.ok ? null : response,
  }
}
