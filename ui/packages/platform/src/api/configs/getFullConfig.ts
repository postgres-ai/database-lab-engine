import { request } from 'helpers/request'

export const getFullConfig = async (instanceId: string) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: '/admin/config.yaml',
      method: 'get',
    }),
  })
    .then((res) => res.blob())
    .then((blob) => blob.text())
    .then((yamlAsString) => {
      return yamlAsString
    })

  return {
    response: response ? response : null,
    error: response && null,
  }
}
