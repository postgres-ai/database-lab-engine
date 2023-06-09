import { request } from 'helpers/request'

export const getOrgKeys = async (org_id: number) => {
  const response = await request(`/org_keys?org_id=eq.${org_id}`)

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}
