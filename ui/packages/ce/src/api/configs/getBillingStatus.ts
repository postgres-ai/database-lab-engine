import { request } from 'helpers/request'

export type ResponseType = {
  result: string
  billing_active: boolean
  recognized_org: {
    id: string
    name: string
    alias: string
    billing_page: string
    priveleged_until: Date
  }
}

export const getBillingStatus = async () => {
  const response = await request('/admin/billing-status')

  return {
    response: response.ok ? ((await response.json()) as ResponseType) : null,
    error: response.ok ? null : await response.json(),
  }
}
