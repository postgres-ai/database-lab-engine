import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

const { getBillingStatus } = vi.hoisted(() => ({ getBillingStatus: vi.fn() }))

vi.mock('api/configs/getBillingStatus', () => ({ getBillingStatus }))
vi.mock('api/configs/activateBilling', () => ({ activateBilling: vi.fn() }))

import { StickyTopBar } from 'App/Menu/StickyTopBar'

describe('StickyTopBar', () => {
  it('shows the missing org key banner when billing status has no recognized org', async () => {
    getBillingStatus.mockResolvedValue({ response: null, error: { code: 'BAD_REQUEST', message: 'org key is missing' } })

    render(<StickyTopBar />)

    await waitFor(() => expect(screen.getByText(/org key is missing/i)).toBeTruthy())
  })

  it('shows no banner when the admin route refuses the token', async () => {
    getBillingStatus.mockResolvedValue({
      response: null,
      error: { code: 'UNAUTHORIZED', message: 'personal tokens are not accepted' },
    })

    render(<StickyTopBar />)

    await waitFor(() => expect(getBillingStatus).toHaveBeenCalled())
    expect(screen.queryByText(/personal tokens are not accepted/i)).toBeNull()
    expect(screen.queryByText(/re-activate DBLab/i)).toBeNull()
  })
})
