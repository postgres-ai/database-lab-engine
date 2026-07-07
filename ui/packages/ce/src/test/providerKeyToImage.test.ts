import { describe, it, expect } from 'vitest'
import { providerKeyToImage } from '@postgres.ai/shared/pages/Instance/Configuration/configOptions'

describe('providerKeyToImage', () => {
  it('resolves generic to Generic Postgres with default tag', () => {
    const out = providerKeyToImage('generic', 15)
    expect(out).toEqual({ imageType: 'Generic Postgres', defaultTag: '15-0.5.3', fallback: false })
  })

  it('resolves pg 18 to the dedicated 0.6.1 tag', () => {
    const out = providerKeyToImage('generic', 18)
    expect(out.defaultTag).toBe('18-0.6.1')
  })

  const seCases = [
    { key: 'rds', want: 'rds' },
    { key: 'aurora', want: 'aurora' },
    { key: 'cloudsql', want: 'google-cloud-sql' },
    { key: 'supabase', want: 'supabase' },
    { key: 'heroku', want: 'heroku' },
    { key: 'timescale', want: 'timescale-cloud' },
  ]
  seCases.forEach(({ key, want }) => {
    it(`resolves ${key} to ${want} with no default tag (UI fetches from getSeImages)`, () => {
      const out = providerKeyToImage(key, 15)
      expect(out).toEqual({ imageType: want, defaultTag: undefined, fallback: false })
    })
  })

  it('falls back to Generic Postgres with warning for azure (no SE mapping yet)', () => {
    const out = providerKeyToImage('azure', 15)
    expect(out).toEqual({ imageType: 'Generic Postgres', defaultTag: '15-0.5.3', fallback: true })
  })

  it('falls back with warning for an unknown provider key', () => {
    const out = providerKeyToImage('mythical-provider', 15)
    expect(out.fallback).toBe(true)
    expect(out.imageType).toBe('Generic Postgres')
  })

  it('omits defaultTag when pgMajorVersion is unknown (0)', () => {
    const out = providerKeyToImage('generic', 0)
    expect(out.defaultTag).toBeUndefined()
  })
})
