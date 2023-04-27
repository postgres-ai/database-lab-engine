import { request } from 'helpers/request'

export const activateBilling = async () => {
  const response = await request('/admin/activate', {
    method: 'POST',
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
