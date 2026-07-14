/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

// interval between clone status polls while a clone is still unstable.
export const UNSTABLE_CLONE_UPDATE_TIMEOUT = 1000

// ClonePoller owns a single self-rescheduling timeout used to poll clone status
// until it becomes stable. It is reusable: start() re-arms a poller that was
// previously stopped, and stop() cancels any pending tick and blocks further
// scheduling — including a tick that an in-flight request is about to queue.
export class ClonePoller {
  private timeout?: ReturnType<typeof setTimeout>
  private stopped = false

  constructor(private readonly intervalMs: number = UNSTABLE_CLONE_UPDATE_TIMEOUT) {}

  get isStopped() {
    return this.stopped
  }

  // start re-arms the poller so a reused store polls again.
  start = () => {
    this.stopped = false
  }

  // stop halts polling and cancels any pending tick.
  stop = () => {
    this.stopped = true
    clearTimeout(this.timeout)
  }

  // cancel drops a pending tick without blocking future scheduling.
  cancel = () => {
    clearTimeout(this.timeout)
  }

  // scheduleNext queues the next tick, unless the poller has been stopped.
  scheduleNext = (tick: () => void) => {
    clearTimeout(this.timeout)
    if (this.stopped) return
    this.timeout = setTimeout(tick, this.intervalMs)
  }
}
