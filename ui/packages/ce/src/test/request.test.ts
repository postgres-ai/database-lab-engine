import { describe, it, expect, vi, beforeEach } from 'vitest'

const { requestCore, removeAuthToken, setIsInvalidAuthToken, setIsValidAuthToken } = vi.hoisted(() => ({
  requestCore: vi.fn(),
  removeAuthToken: vi.fn(),
  setIsInvalidAuthToken: vi.fn(),
  setIsValidAuthToken: vi.fn(),
}))

vi.mock('@postgres.ai/shared/helpers/request', () => ({ request: requestCore }))
vi.mock('helpers/localStorage', () => ({ localStorage: { getAuthToken: () => 'token', removeAuthToken } }))
vi.mock('stores/app', () => ({ appStore: { setIsInvalidAuthToken, setIsValidAuthToken } }))
vi.mock('config/env', () => ({ API_URL_PREFIX: '' }))

import { request } from 'helpers/request'

describe('request', () => {
  beforeEach(() => vi.clearAllMocks())

  const cases: { name: string; path: string; status: number; wantLogout: boolean }[] = [
    { name: '401 on a regular route invalidates the session', path: '/status', status: 401, wantLogout: true },
    { name: '401 on an admin route keeps the session', path: '/admin/config', status: 401, wantLogout: false },
    { name: '200 on an admin route keeps the session', path: '/admin/config', status: 200, wantLogout: false },
  ]

  cases.forEach((tc) => {
    it(tc.name, async () => {
      requestCore.mockResolvedValue({ status: tc.status })

      const response = await request(tc.path)

      expect(response.status).toBe(tc.status)
      expect(removeAuthToken).toHaveBeenCalledTimes(tc.wantLogout ? 1 : 0)
      expect(setIsInvalidAuthToken).toHaveBeenCalledTimes(tc.wantLogout ? 1 : 0)
    })
  })
})
