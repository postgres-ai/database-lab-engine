import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

import {
  ClonePoller,
  UNSTABLE_CLONE_UPDATE_TIMEOUT,
} from '@postgres.ai/shared/utils/clonePoller'

describe('ClonePoller', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('is running by default', () => {
    expect(new ClonePoller().isStopped).toBe(false)
  })

  it('runs the scheduled tick after the default interval', () => {
    const poller = new ClonePoller()
    const tick = vi.fn()

    poller.scheduleNext(tick)
    vi.advanceTimersByTime(UNSTABLE_CLONE_UPDATE_TIMEOUT - 1)
    expect(tick).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1)
    expect(tick).toHaveBeenCalledTimes(1)
  })

  it('honors a custom interval', () => {
    const poller = new ClonePoller(50)
    const tick = vi.fn()

    poller.scheduleNext(tick)
    vi.advanceTimersByTime(50)
    expect(tick).toHaveBeenCalledTimes(1)
  })

  it('keeps only the latest pending tick', () => {
    const poller = new ClonePoller(100)
    const first = vi.fn()
    const second = vi.fn()

    poller.scheduleNext(first)
    poller.scheduleNext(second)
    vi.advanceTimersByTime(100)

    expect(first).not.toHaveBeenCalled()
    expect(second).toHaveBeenCalledTimes(1)
  })

  it('stop() cancels a pending tick and marks the poller stopped', () => {
    const poller = new ClonePoller(100)
    const tick = vi.fn()

    poller.scheduleNext(tick)
    poller.stop()
    vi.advanceTimersByTime(100)

    expect(tick).not.toHaveBeenCalled()
    expect(poller.isStopped).toBe(true)
  })

  it('scheduleNext() is a no-op once stopped', () => {
    const poller = new ClonePoller(100)
    const tick = vi.fn()

    poller.stop()
    poller.scheduleNext(tick)
    vi.advanceTimersByTime(100)

    expect(tick).not.toHaveBeenCalled()
  })

  it('start() re-arms a stopped poller', () => {
    const poller = new ClonePoller(100)
    const tick = vi.fn()

    poller.stop()
    poller.start()
    expect(poller.isStopped).toBe(false)

    poller.scheduleNext(tick)
    vi.advanceTimersByTime(100)
    expect(tick).toHaveBeenCalledTimes(1)
  })

  it('cancel() drops the pending tick but keeps scheduling enabled', () => {
    const poller = new ClonePoller(100)
    const dropped = vi.fn()
    const next = vi.fn()

    poller.scheduleNext(dropped)
    poller.cancel()
    vi.advanceTimersByTime(100)
    expect(dropped).not.toHaveBeenCalled()
    expect(poller.isStopped).toBe(false)

    poller.scheduleNext(next)
    vi.advanceTimersByTime(100)
    expect(next).toHaveBeenCalledTimes(1)
  })
})
