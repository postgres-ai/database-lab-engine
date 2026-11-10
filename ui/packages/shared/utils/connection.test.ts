import { describe, it, expect } from 'vitest'

import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { getJdbcConnectionStr, getPsqlConnectionStr } from '@postgres.ai/shared/utils/connection'

const cloneOf = (username: string) => ({ db: { host: '10.0.0.1', port: '6000', username } }) as Clone

describe('getPsqlConnectionStr', () => {
  it.each([
    { name: 'plain', username: 'alice', expected: `"host=10.0.0.1 port=6000 user=alice dbname=DBNAME"` },
    { name: 'dot', username: 'app.24', expected: `"host=10.0.0.1 port=6000 user=app.24 dbname=DBNAME"` },
    { name: 'space', username: 'app user', expected: `"host=10.0.0.1 port=6000 user='app user' dbname=DBNAME"` },
    { name: 'apostrophe', username: `it's`, expected: `"host=10.0.0.1 port=6000 user='it\\'s' dbname=DBNAME"` },
    { name: 'backslash', username: 'back\\slash', expected: `"host=10.0.0.1 port=6000 user='back\\\\slash' dbname=DBNAME"` },
  ])('quotes a username with $name', ({ username, expected }) => {
    expect(getPsqlConnectionStr(cloneOf(username))).toBe(expected)
  })

  it('returns null without a username', () => {
    expect(getPsqlConnectionStr(cloneOf(''))).toBeNull()
  })
})

describe('getJdbcConnectionStr', () => {
  it.each([
    { name: 'plain', username: 'alice', expected: 'user=alice&' },
    { name: 'space', username: 'app user', expected: 'user=app%20user&' },
    { name: 'ampersand', username: 'a&b=c', expected: 'user=a%26b%3Dc&' },
  ])('encodes a username with $name', ({ username, expected }) => {
    expect(getJdbcConnectionStr(cloneOf(username))).toContain(expected)
  })
})
