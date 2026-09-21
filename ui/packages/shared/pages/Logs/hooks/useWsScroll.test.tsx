import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, act } from '@testing-library/react'

const { wsSnackbarMock } = vi.hoisted(() => ({ wsSnackbarMock: vi.fn() }))

vi.mock('@postgres.ai/shared/pages/Logs/wsSnackbar', () => ({
  wsSnackbar: wsSnackbarMock,
}))

import { useWsScroll } from '@postgres.ai/shared/pages/Logs/hooks/useWsScroll'

const Probe = () => {
  useWsScroll(false)
  return null
}

const flushObserver = async (mutate: () => void) => {
  await act(async () => {
    mutate()
    await new Promise((resolve) => setTimeout(resolve, 0))
  })
}

describe('useWsScroll', () => {
  let logsContainer: HTMLElement
  let scrollIntoView: ReturnType<typeof vi.fn>

  beforeEach(() => {
    const contentContainer = document.createElement('div')
    contentContainer.id = 'content-container'

    logsContainer = document.createElement('div')
    logsContainer.id = 'logs-container'
    scrollIntoView = vi.fn()
    logsContainer.scrollIntoView = scrollIntoView

    contentContainer.appendChild(logsContainer)
    document.body.appendChild(contentContainer)
  })

  afterEach(() => {
    document.body.innerHTML = ''
    wsSnackbarMock.mockClear()
  })

  it('scrolls to the newest line when a log line arrives', async () => {
    render(<Probe />)

    await flushObserver(() =>
      logsContainer.appendChild(document.createElement('p')),
    )

    expect(scrollIntoView).toHaveBeenCalledWith(false)
    expect(wsSnackbarMock).toHaveBeenLastCalledWith(true, true)
  })

  it('ignores page chrome inserted as a div', async () => {
    render(<Probe />)

    const callsBefore = wsSnackbarMock.mock.calls.length

    await flushObserver(() =>
      logsContainer.appendChild(document.createElement('div')),
    )

    expect(scrollIntoView).not.toHaveBeenCalled()
    expect(wsSnackbarMock.mock.calls).toHaveLength(callsBefore)
  })

  it('stops observing once the page unmounts', async () => {
    const { unmount } = render(<Probe />)

    unmount()

    await flushObserver(() =>
      logsContainer.appendChild(document.createElement('p')),
    )

    expect(scrollIntoView).not.toHaveBeenCalled()
  })
})
