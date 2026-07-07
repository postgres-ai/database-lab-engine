import { describe, it, expect } from 'vitest'
import {
  connectionStringFromFields,
  connectionStringToFields,
  ConnectionStringError,
  ErrPasswordInConnectionString,
  ErrMultiHost,
  ErrInvalidConnectionString,
} from '@postgres.ai/shared/pages/Instance/Configuration/connectionString'

describe('connectionStringToFields — URI form', () => {
  it('parses the canonical postgres:// URI', () => {
    const out = connectionStringToFields('postgres://app@db.example.com:5432/shop')
    expect(out.fields).toEqual({ host: 'db.example.com', port: '5432', username: 'app', dbname: 'shop' })
    expect(out.portWasExplicit).toBe(true)
  })

  it('treats postgresql:// scheme as equivalent', () => {
    const out = connectionStringToFields('postgresql://app@db.example.com:5432/shop')
    expect(out.fields.host).toBe('db.example.com')
  })

  it('defaults port to 5432 when omitted and marks it implicit', () => {
    const out = connectionStringToFields('postgres://app@db.example.com/shop')
    expect(out.fields.port).toBe('5432')
    expect(out.portWasExplicit).toBe(false)
  })

  it('decodes percent-encoded username and dbname', () => {
    const out = connectionStringToFields('postgres://my%20app@host/data%20base')
    expect(out.fields.username).toBe('my app')
    expect(out.fields.dbname).toBe('data base')
  })

  it('returns empty username/dbname when not present', () => {
    const out = connectionStringToFields('postgres://host:5432')
    expect(out.fields.username).toBe('')
    expect(out.fields.dbname).toBe('')
  })

  it('handles bracketed IPv6 hosts', () => {
    const out = connectionStringToFields('postgres://app@[::1]:5432/shop')
    expect(out.fields.host).toBe('::1')
  })

  it('rejects a password in URI userinfo', () => {
    expect(() => connectionStringToFields('postgres://app:secret@host/shop')).toThrowError(
      new ConnectionStringError(ErrPasswordInConnectionString),
    )
  })

  it('rejects a multi-host URI', () => {
    expect(() => connectionStringToFields('postgres://app@a.example.com,b.example.com/shop')).toThrowError(
      new ConnectionStringError(ErrMultiHost),
    )
  })

  it('rejects malformed URIs', () => {
    expect(() => connectionStringToFields('postgres://')).toThrow()
  })
})

describe('connectionStringToFields — DSN form', () => {
  it('parses a basic DSN', () => {
    const out = connectionStringToFields('host=db.example.com port=5432 user=app dbname=shop')
    expect(out.fields).toEqual({ host: 'db.example.com', port: '5432', username: 'app', dbname: 'shop' })
    expect(out.portWasExplicit).toBe(true)
  })

  it('defaults port to 5432 and marks it implicit when absent', () => {
    const out = connectionStringToFields('host=db.example.com user=app dbname=shop')
    expect(out.fields.port).toBe('5432')
    expect(out.portWasExplicit).toBe(false)
  })

  it('accepts username and database as aliases', () => {
    const out = connectionStringToFields('host=h username=u database=d')
    expect(out.fields.username).toBe('u')
    expect(out.fields.dbname).toBe('d')
  })

  it('unquotes single-quoted values with spaces', () => {
    const out = connectionStringToFields("host=h user='my app' dbname='data base'")
    expect(out.fields.username).toBe('my app')
    expect(out.fields.dbname).toBe('data base')
  })

  it('rejects DSN containing a password', () => {
    expect(() => connectionStringToFields('host=h user=u password=secret dbname=d')).toThrowError(
      new ConnectionStringError(ErrPasswordInConnectionString),
    )
  })
})

describe('connectionStringToFields — error paths', () => {
  it('rejects empty input', () => {
    expect(() => connectionStringToFields('   ')).toThrowError(
      new ConnectionStringError(ErrInvalidConnectionString),
    )
  })

  it('rejects gibberish that is neither URI nor DSN', () => {
    expect(() => connectionStringToFields('just-a-hostname')).toThrowError(
      new ConnectionStringError(ErrInvalidConnectionString),
    )
  })
})

describe('connectionStringFromFields', () => {
  it('serializes the canonical URI', () => {
    const s = connectionStringFromFields({ host: 'db.example.com', port: '5432', username: 'app', dbname: 'shop' })
    expect(s).toBe('postgres://app@db.example.com:5432/shop')
  })

  it('omits port 5432 when opts.omitDefaultPort is true', () => {
    const s = connectionStringFromFields(
      { host: 'db.example.com', port: '5432', username: 'app', dbname: 'shop' },
      { omitDefaultPort: true },
    )
    expect(s).toBe('postgres://app@db.example.com/shop')
  })

  it('still serializes a non-default port even when omitDefaultPort is set', () => {
    const s = connectionStringFromFields(
      { host: 'db.example.com', port: '6432', username: 'app', dbname: 'shop' },
      { omitDefaultPort: true },
    )
    expect(s).toBe('postgres://app@db.example.com:6432/shop')
  })

  it('wraps IPv6 hosts in brackets', () => {
    const s = connectionStringFromFields({ host: '::1', port: '5432', username: 'app', dbname: 'shop' })
    expect(s).toBe('postgres://app@[::1]:5432/shop')
  })

  it('encodes special characters in username/dbname', () => {
    const s = connectionStringFromFields({ host: 'h', port: '5432', username: 'my app', dbname: 'data base' })
    expect(s).toBe('postgres://my%20app@h:5432/data%20base')
  })
})

describe('round-trip: fields → URI → fields', () => {
  it('preserves all four fields', () => {
    const original = { host: 'db.example.com', port: '6543', username: 'app', dbname: 'shop' }
    const uri = connectionStringFromFields(original)
    const parsed = connectionStringToFields(uri)
    expect(parsed.fields).toEqual(original)
    expect(parsed.portWasExplicit).toBe(true)
  })

  it('respects empty-port preservation: serialize without port → parse marks it implicit', () => {
    const fields = { host: 'h', port: '5432', username: 'u', dbname: 'd' }
    const uri = connectionStringFromFields(fields, { omitDefaultPort: true })
    const parsed = connectionStringToFields(uri)
    expect(parsed.fields.port).toBe('5432')
    expect(parsed.portWasExplicit).toBe(false)
  })
})
