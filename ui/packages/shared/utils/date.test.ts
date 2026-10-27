import { describe, it, expect } from 'vitest'

import { formatDataLag } from '@postgres.ai/shared/utils/date'

describe('formatDataLag', () => {
  const now = new Date('2026-10-26T10:28:00Z')

  it.each([
    { name: 'iso timestamp', dataStateAt: '2026-10-26T08:24:39Z', expected: '2 hours 3 minutes' },
    { name: 'utc suffix timestamp', dataStateAt: '2026-10-26 10:12:00 UTC', expected: '16 minutes' },
    { name: 'multiple days', dataStateAt: '2026-10-23T09:28:00Z', expected: '3 days 1 hour' },
    { name: 'over a month', dataStateAt: '2026-09-16T10:28:00Z', expected: '40 days' },
    { name: 'exactly a year', dataStateAt: '2025-10-26T10:28:00Z', expected: '365 days' },
    { name: 'future value', dataStateAt: '2026-10-26T10:40:00Z', expected: 'less than a minute' },
    { name: 'under a minute', dataStateAt: '2026-10-26T10:27:30Z', expected: 'less than a minute' },
    { name: 'empty value', dataStateAt: '', expected: '-' },
    { name: 'missing value', dataStateAt: undefined, expected: '-' },
    { name: 'malformed value', dataStateAt: 'not a date', expected: '-' },
  ])('formats $name', ({ dataStateAt, expected }) => {
    expect(formatDataLag(dataStateAt, now)).toBe(expected)
  })
})
