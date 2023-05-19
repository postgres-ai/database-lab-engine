import { request } from 'helpers/request'

export const startBillingSession = async (orgId: number, returnUrl: string) => {
  const response = await request(`/rpc/billing_portal_start_session`, {
    headers: {
      Accept: 'application/vnd.pgrst.object+json',
    },
    method: 'POST',
    body: JSON.stringify({
      org_id: orgId,
      return_url: returnUrl,
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
