// connectionString.ts — Expert-mode helper that converts between the
// (host, port, username, dbname) shape stored in `server.yml` and the
// single URL field the form exposes. The engine remains the source of
// truth for actual probing (see engine/internal/retrieval/probe/parser.go);
// this parser only powers the form's load/save round-trip.
//
// Accepts both URI form (postgres://user@host:5432/dbname) and DSN form
// (host=db port=5432 user=app dbname=shop). Rejects any input that carries
// a password — passwords belong in the masked field, never the URL field.

export type ConnectionFields = {
  host: string
  port: string
  username: string
  dbname: string
}

export type ParseResult = {
  fields: ConnectionFields
  portWasExplicit: boolean
}

export class ConnectionStringError extends Error {}

export const ErrPasswordInConnectionString = 'Password must be entered in the Password field, not the connection string.'
export const ErrMultiHost = 'Multi-host connection strings are not supported.'
export const ErrInvalidConnectionString = 'Could not parse the connection string.'

const URI_SCHEMES = ['postgres:', 'postgresql:']

const isUri = (s: string) => URI_SCHEMES.some((p) => s.toLowerCase().startsWith(p))

const stripBrackets = (h: string) => (h.startsWith('[') && h.endsWith(']') ? h.slice(1, -1) : h)

const parseUri = (s: string): ParseResult => {
  let url: URL
  try {
    url = new URL(s)
  } catch {
    throw new ConnectionStringError(ErrInvalidConnectionString)
  }

  if (url.password) throw new ConnectionStringError(ErrPasswordInConnectionString)

  if (!url.hostname) throw new ConnectionStringError(ErrInvalidConnectionString)
  if (url.hostname.includes(',')) throw new ConnectionStringError(ErrMultiHost)

  const portWasExplicit = url.port !== ''
  const path = url.pathname.startsWith('/') ? url.pathname.slice(1) : url.pathname
  const dbname = path.includes(',') ? '' : decodeURIComponent(path)

  return {
    fields: {
      host: stripBrackets(url.hostname),
      port: portWasExplicit ? url.port : '5432',
      username: url.username ? decodeURIComponent(url.username) : '',
      dbname,
    },
    portWasExplicit,
  }
}

// Tokenizer for libpq DSN form: key=value pairs, values may be single-quoted
// when they contain spaces (libpq spec). Throws on malformed input.
const tokenizeDsn = (s: string): Record<string, string> => {
  const out: Record<string, string> = {}
  let i = 0

  while (i < s.length) {
    // skip leading whitespace
    while (i < s.length && /\s/.test(s[i])) i++
    if (i >= s.length) break

    // read key
    const keyStart = i
    while (i < s.length && s[i] !== '=' && !/\s/.test(s[i])) i++
    if (i >= s.length || s[i] !== '=') throw new ConnectionStringError(ErrInvalidConnectionString)
    const key = s.slice(keyStart, i)
    i++ // skip '='

    // read value (possibly quoted)
    let value = ''
    if (s[i] === "'") {
      i++
      while (i < s.length && s[i] !== "'") {
        if (s[i] === '\\' && i + 1 < s.length) {
          value += s[i + 1]
          i += 2
          continue
        }
        value += s[i]
        i++
      }
      if (i >= s.length) throw new ConnectionStringError(ErrInvalidConnectionString)
      i++ // skip closing quote
    } else {
      while (i < s.length && !/\s/.test(s[i])) {
        value += s[i]
        i++
      }
    }

    out[key.toLowerCase()] = value
  }

  return out
}

const parseDsn = (s: string): ParseResult => {
  const tokens = tokenizeDsn(s)
  if (tokens.password) throw new ConnectionStringError(ErrPasswordInConnectionString)

  const host = tokens.host ?? tokens.hostaddr ?? ''
  if (host.includes(',')) throw new ConnectionStringError(ErrMultiHost)

  const portRaw = tokens.port ?? ''
  const portWasExplicit = portRaw !== ''

  return {
    fields: {
      host,
      port: portWasExplicit ? portRaw : '5432',
      username: tokens.user ?? tokens.username ?? '',
      dbname: tokens.dbname ?? tokens.database ?? '',
    },
    portWasExplicit,
  }
}

export const connectionStringToFields = (s: string): ParseResult => {
  const trimmed = s.trim()
  if (!trimmed) throw new ConnectionStringError(ErrInvalidConnectionString)

  if (isUri(trimmed)) return parseUri(trimmed)
  if (trimmed.includes('=')) return parseDsn(trimmed)
  throw new ConnectionStringError(ErrInvalidConnectionString)
}

// formatHost wraps IPv6 literals in brackets so the resulting URI parses cleanly.
const formatHost = (host: string) => (host.includes(':') ? `[${host}]` : host)

export type SerializeOptions = {
  // When true, omit `:5432` from the URI even if `fields.port === '5432'`.
  // Used to preserve configs that originally had no `port:` key.
  omitDefaultPort?: boolean
}

export const connectionStringFromFields = (
  fields: ConnectionFields,
  opts: SerializeOptions = {},
): string => {
  const userPart = fields.username ? `${encodeURIComponent(fields.username)}@` : ''
  const showPort = fields.port && !(opts.omitDefaultPort && fields.port === '5432')
  const portPart = showPort ? `:${fields.port}` : ''
  const dbPart = fields.dbname ? `/${encodeURIComponent(fields.dbname)}` : ''
  return `postgres://${userPart}${formatHost(fields.host)}${portPart}${dbPart}`
}
