import { UpdateClone } from '@postgres.ai/shared/types/api/endpoints/updateClone'

import { request } from 'helpers/request'

export const updateClone: UpdateClone = async (req) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      action: '/clone/' + encodeURIComponent(req.cloneId),
      instance_id: req.instanceId,
      method: 'patch',
      data: {
        protected: req.clone.isProtected,
      },
    }),
  })

  return {
    response: response.ok ? true : null,
    error: response.ok ? null : response,
  }
}
