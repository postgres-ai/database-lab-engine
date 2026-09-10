import { describe, it, expect, afterAll } from 'vitest'
import { formatDateToISO } from '@postgres.ai/shared/utils/date'

const originalTimezone = process.env.TZ

describe('formatDateToISO', () => {
  afterAll(() => {
    process.env.TZ = originalTimezone
  })

  const timezones = [
    'UTC',
    'Europe/Moscow',
    'America/Los_Angeles',
    'Asia/Kolkata',
  ]

  timezones.forEach((timezone) => {
    it(`keeps the UTC wall-clock in ${timezone}`, () => {
      process.env.TZ = timezone
      expect(formatDateToISO('20260908100810')).toBe('2026-09-08T10:08:10Z')
    })
  })

  it('points to the same instant regardless of the browser timezone', () => {
    process.env.TZ = 'UTC'
    const utc = new Date(formatDateToISO('20260908100810')).getTime()

    process.env.TZ = 'Europe/Moscow'
    expect(new Date(formatDateToISO('20260908100810')).getTime()).toBe(utc)
  })

  it('keeps a wall-clock that does not exist in the local timezone', () => {
    process.env.TZ = 'Europe/Berlin'
    expect(formatDateToISO('20260329023000')).toBe('2026-03-29T02:30:00Z')
  })

  const invalidCases = [
    '',
    '   ',
    'not-a-date',
    '2026-09-08T10:08:10Z',
    '202609081008',
    '20261308100810',
    '20260908250810',
  ]

  invalidCases.forEach((value) => {
    it(`returns an empty string for ${JSON.stringify(value)}`, () => {
      expect(formatDateToISO(value)).toBe('')
    })
  })
})
