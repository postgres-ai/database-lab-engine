import { CloneDto, formatCloneDto } from '@postgres.ai/shared/types/api/entities/clone'

import { request } from 'helpers/request'

type Req = {
  instanceId: string
  cloneId: string
  snapshotId: string
  dbUser: string
  dbPassword: string
  isProtected: boolean
}

export const createClone = async (req: Req) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      action: '/clone',
      method: 'post',
      data: {
        id: req.cloneId,
        snapshot: {
          id: req.snapshotId,
        },
        db: {
          username: req.dbUser,
          password: req.dbPassword,
        },
        protected: req.isProtected,
      },
    })
  })

  return {
    response: response.ok ? formatCloneDto(await response.json() as CloneDto) : null,
    error: response.ok ? null : response,
  }
}
