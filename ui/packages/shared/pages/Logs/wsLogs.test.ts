import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

import { Api } from '@postgres.ai/shared/pages/Instance/stores/Main'
import {
  closeConnection,
  establishConnection,
  restartConnection,
} from '@postgres.ai/shared/pages/Logs/wsLogs'

type TokenResult = {
  response: { token: string } | null
  error: Response | null
}

type FakeSocket = {
  close: ReturnType<typeof vi.fn>
  onopen: (() => void) | null
  onclose: ((event: unknown) => void) | null
  onerror: ((error: unknown) => void) | null
  onmessage: ((event: { data: string }) => void) | null
}

const createSocket = (): FakeSocket => ({
  close: vi.fn(),
  onopen: null,
  onclose: null,
  onerror: null,
  onmessage: null,
})

const buildApi = (sockets: FakeSocket[], getToken: () => Promise<TokenResult>) =>
  ({
    getWSToken: getToken,
    initWS: () => {
      const socket = createSocket()
      sockets.push(socket)
      return socket as unknown as WebSocket
    },
  } as unknown as Api)

const grantedToken = (): Promise<TokenResult> =>
  Promise.resolve({ response: { token: 'ws-token' }, error: null })

const deferredToken = () => {
  let grant: (result: TokenResult) => void = () => undefined
  const promise = new Promise<TokenResult>((resolve) => {
    grant = resolve
  })

  return { promise, grant }
}

describe('log socket lifecycle', () => {
  let sockets: FakeSocket[]

  beforeEach(() => {
    sockets = []
    const container = document.createElement('div')
    container.id = 'logs-container'
    document.body.appendChild(container)
    localStorage.setItem('logsFilter', JSON.stringify({ '[other]': true }))
  })

  afterEach(() => {
    closeConnection()
    document.body.innerHTML = ''
    localStorage.clear()
  })

  it('closes a socket that opened after the caller disconnected', async () => {
    const token = deferredToken()
    const api = buildApi(sockets, () => token.promise)

    const connecting = establishConnection(api, 'instance-1')
    closeConnection()
    token.grant({ response: { token: 'ws-token' }, error: null })
    await connecting

    expect(sockets).toHaveLength(1)
    expect(sockets[0].close).toHaveBeenCalledTimes(1)
    expect(sockets[0].onmessage).toBeNull()

    closeConnection()

    expect(sockets[0].close).toHaveBeenCalledTimes(1)
  })

  it('closes the previous socket when the page reconnects', async () => {
    const api = buildApi(sockets, grantedToken)

    await establishConnection(api, 'instance-1')
    await establishConnection(api, 'instance-1')

    expect(sockets).toHaveLength(2)
    expect(sockets[0].close).toHaveBeenCalledTimes(1)
    expect(sockets[1].close).not.toHaveBeenCalled()
    expect(sockets[1].onmessage).toBeInstanceOf(Function)
  })

  it('detaches the drop handler when the caller closes on purpose', async () => {
    const api = buildApi(sockets, grantedToken)

    await establishConnection(api, 'instance-1')

    expect(sockets[0].onclose).toBeInstanceOf(Function)

    closeConnection()

    expect(sockets[0].onclose).toBeNull()
    expect(sockets[0].close).toHaveBeenCalledTimes(1)
  })

  it('keeps the drop handler for a close the server initiates', async () => {
    const api = buildApi(sockets, grantedToken)

    await establishConnection(api, 'instance-1')
    sockets[0].onclose?.({})
    closeConnection()

    expect(sockets[0].close).not.toHaveBeenCalled()
  })

  it('restart clears the log and closes the open socket', async () => {
    const api = buildApi(sockets, grantedToken)

    await establishConnection(api, 'instance-1')

    const container = document.getElementById('logs-container') as HTMLElement
    container.appendChild(document.createElement('p'))
    container.appendChild(document.createElement('p'))

    restartConnection(api, 'instance-1')
    await vi.waitFor(() => expect(sockets).toHaveLength(2))

    expect(sockets[0].close).toHaveBeenCalledTimes(1)
    expect(container.querySelectorAll('p')).toHaveLength(0)
  })

  it('appends a log line without a persisted filter state', async () => {
    localStorage.removeItem('logsFilter')

    const api = buildApi(sockets, grantedToken)

    await establishConnection(api, 'instance-1')

    expect(() =>
      sockets[0].onmessage?.({ data: btoa('2026-09-17 10:00:00 [base.go] [INFO] up') }),
    ).not.toThrow()
  })
})
