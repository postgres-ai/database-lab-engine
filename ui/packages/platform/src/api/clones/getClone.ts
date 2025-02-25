import {
  CloneDto,
  formatCloneDto,
} from '@postgres.ai/shared/types/api/entities/clone'

import { request } from 'helpers/request'

type Request = {
  instanceId: string
  cloneId: string
}

export const getClone = async (req: Request) => {
  const response = (await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      action: '/clone/' + encodeURIComponent(req.cloneId),
      instance_id: req.instanceId,
      method: 'get'
    })
  }))

  return {
    response: response.ok
      ? formatCloneDto(await response.json() as CloneDto)
      : null,
    error: response.ok ? null : response,
  }
}
