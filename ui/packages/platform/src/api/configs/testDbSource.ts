import { dbSource } from '@postgres.ai/shared/types/api/entities/dbSource'
import { request } from 'helpers/request'

export const testDbSource = async (req: dbSource) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: req.instanceId,
      action: '/admin/test-db-source',
      method: 'post',
      data: {
        host: req.host,
        port: req.port.toString(),
        dbname: req.dbname,
        username: req.username,
        password: req.password,
        db_list: req.db_list,
      },
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
