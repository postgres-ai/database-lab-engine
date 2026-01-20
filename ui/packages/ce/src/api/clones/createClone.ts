import { CreateClone } from '@postgres.ai/shared/types/api/endpoints/createClone'
import {
  CloneDto,
  formatCloneDto,
} from '@postgres.ai/shared/types/api/entities/clone'

import { request } from 'helpers/request'

export const createClone: CreateClone = async (req) => {
  const isProtected = req.protectionDurationMinutes !== 'none'
  const protectionDurationMinutes = isProtected ? parseInt(req.protectionDurationMinutes, 10) : undefined

  const response = await request('/clone', {
    method: 'POST',
    body: JSON.stringify({
      id: req.cloneId,
      snapshot: {
        id: req.snapshotId,
      },
      protected: isProtected,
      ...(protectionDurationMinutes !== undefined && { protectionDurationMinutes }),
      ...(req.branch && { branch: req.branch }),
      db: {
        username: req.dbUser,
        password: req.dbPassword,
      },
    }),
  })

  return {
    response: response.ok
      ? formatCloneDto((await response.json()) as CloneDto)
      : null,
    error: response.ok ? null : response,
  }
}
