import { CloneDto, formatCloneDto } from '@postgres.ai/shared/types/api/entities/clone'

import { request } from 'helpers/request'

type Request = {
  instanceId: string
  cloneId: string
}

export const getClone = async (req: Request) => {
  const response = await request('/rpc/dblab_clone_status', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      clone_id: req.cloneId,
    })
  })

  return {
    response: response.ok ? formatCloneDto(await response.json() as CloneDto) : null,
    error: response.ok ? null : response,
  }
}
