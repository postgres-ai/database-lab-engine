import { request } from 'helpers/request'

export const getSubscription = async (orgId: number) => {
  const response = await request(`/rpc/billing_subscriptions`, {
    headers: {
      Accept: 'application/vnd.pgrst.object+json',
    },
    method: 'POST',
    body: JSON.stringify({
      org_id: orgId,
    }),
  })

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : response,
  }
}
