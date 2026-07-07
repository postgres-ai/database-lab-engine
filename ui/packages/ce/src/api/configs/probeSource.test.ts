import { describe, it, expect, vi, beforeEach } from 'vitest'

const requestMock = vi.fn()

vi.mock('helpers/request', () => ({
  request: (...args: unknown[]) => requestMock(...args),
}))

import { probeSource } from './probeSource'

const makeResponse = (status: number, body?: unknown) =>
  new Response(body === undefined ? null : JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })

describe('probeSource', () => {
  beforeEach(() => {
    requestMock.mockReset()
  })

  it('posts URL and password to /admin/probe-source', async () => {
    requestMock.mockResolvedValueOnce(
      makeResponse(200, {
        source: { host: 'h', port: 5432, username: 'u', dbname: 'd' },
        detectedProvider: 'generic',
        dockerImage: 'generic',
        dockerTag: '',
        pgMajorVersion: 15,
        databases: ['d'],
        sharedBuffers: '1GB',
        memoryProbed: true,
        sharedPreloadLibraries: 'pg_stat_statements',
        queryTuning: { work_mem: '4MB' },
      }),
    )

    const out = await probeSource({ url: 'postgres://h/d', password: 'pw' })

    expect(requestMock).toHaveBeenCalledWith('/admin/probe-source', {
      method: 'POST',
      body: JSON.stringify({ url: 'postgres://h/d', password: 'pw' }),
    })
    expect(out.error).toBeNull()
    expect(out.response).toMatchObject({
      detectedProvider: 'generic',
      pgMajorVersion: 15,
      memoryProbed: true,
    })
  })

  const errorCases = [
    {
      name: '400 with engine message uses engine message',
      status: 400,
      body: { message: 'invalid url' },
      expected: 'invalid url',
    },
    {
      name: '400 with empty body falls back to bad-URL message',
      status: 400,
      body: undefined,
      expectedIncludes: 'connection string',
    },
    {
      name: '500 with empty body falls back to server-error message',
      status: 500,
      body: undefined,
      expectedIncludes: 'Internal server error',
    },
    {
      name: '500 with engine message uses engine message',
      status: 500,
      body: { message: 'panic recovered' },
      expected: 'panic recovered',
    },
  ]

  errorCases.forEach((tc) => {
    it(tc.name, async () => {
      requestMock.mockResolvedValueOnce(makeResponse(tc.status, tc.body))
      const out = await probeSource({ url: 'x', password: 'y' })
      expect(out.response).toBeNull()
      expect(out.error?.status).toBe(tc.status)
      if (tc.expected) expect(out.error?.message).toBe(tc.expected)
      if (tc.expectedIncludes)
        expect(out.error?.message).toContain(tc.expectedIncludes)
    })
  })
})
