import { describe, it, expect } from 'vitest'
import { getInitialConfigMode } from '@postgres.ai/shared/pages/Instance/Configuration/configMode'

describe('getInitialConfigMode', () => {
  const cases: {
    name: string
    host: string | undefined
    mode?: string
    want: 'simple' | 'expert'
  }[] = [
    { name: 'undefined host (fresh install) → simple', host: undefined, want: 'simple' },
    { name: 'empty string host → simple', host: '', want: 'simple' },
    { name: 'whitespace-only host treated as filled', host: ' ', want: 'expert' },
    { name: 'populated host → expert', host: 'db.example.com', want: 'expert' },
    { name: 'ipv6 host → expert', host: '::1', want: 'expert' },
    { name: 'physical mode with no host → expert', host: '', mode: 'physical', want: 'expert' },
    { name: 'logical mode with no host → simple', host: '', mode: 'logical', want: 'simple' },
    { name: 'unknown mode with no host → simple', host: '', mode: 'unknown', want: 'simple' },
  ]

  cases.forEach((tc) => {
    it(tc.name, () => {
      expect(getInitialConfigMode(tc.host, tc.mode)).toBe(tc.want)
    })
  })
})
