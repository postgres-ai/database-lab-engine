import { UpdateClone } from '@postgres.ai/shared/types/api/endpoints/updateClone'

import { request } from 'helpers/request'

export const updateClone: UpdateClone = async (req) => {
  const response = await request('/rpc/dblab_clone_update', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      clone_id: req.cloneId,
      clone: {
        protected: req.clone.isProtected,
      },
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}
