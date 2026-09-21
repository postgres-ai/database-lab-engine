import { describe, it, expect } from 'vitest'

import { CloneDto, formatCloneDto } from '@postgres.ai/shared/types/api/entities/clone'

const buildCloneDto = (overrides: Partial<CloneDto> = {}): CloneDto => ({
  createdAt: '2026-09-13 10:00:00 UTC',
  id: 'clone-1',
  branch: 'main',
  revision: 2,
  deleteAt: null,
  status: { code: 'OK', message: 'Clone is ready' },
  protected: false,
  metadata: {
    cloneDiffSize: 1024,
    logicalSize: 4096,
    cloningTime: 1.5,
    maxIdleMinutes: 30,
  },
  db: { username: 'postgres', port: '6000', host: 'localhost' },
  snapshot: null,
  ...overrides,
})

describe('formatCloneDto', () => {
  it('keeps the engine fields the UI reads', () => {
    const clone = formatCloneDto(buildCloneDto())

    expect(clone.revision).toBe(2)
    expect(clone.metadata.logicalSize).toBe(4096)
    expect(clone.metadata.maxIdleMinutes).toBe(30)
  })

  it('parses createdAt into a date', () => {
    const clone = formatCloneDto(buildCloneDto())

    expect(clone.createdAt).toBe('2026-09-13 10:00:00 UTC')
    expect(clone.createdAtDate.toISOString()).toBe('2026-09-13T10:00:00.000Z')
  })

  it('parses deleteAt when the clone is scheduled for deletion', () => {
    const clone = formatCloneDto(
      buildCloneDto({ deleteAt: '2026-09-14T10:00:00Z' }),
    )

    expect(clone.deleteAtDate?.toISOString()).toBe('2026-09-14T10:00:00.000Z')
  })

  it('leaves deleteAt and protectedTill null when the engine omits them', () => {
    const clone = formatCloneDto(buildCloneDto())

    expect(clone.deleteAtDate).toBeNull()
    expect(clone.protectedTillDate).toBeNull()
  })

  it('formats an attached snapshot', () => {
    const clone = formatCloneDto(
      buildCloneDto({
        snapshot: {
          numClones: 1,
          clones: ['clone-1'],
          createdAt: '2026-09-12 09:00:00 UTC',
          dataStateAt: '2026-09-12 08:00:00 UTC',
          id: 'snapshot-1',
          pool: 'pool',
          physicalSize: 10,
          logicalSize: 20,
          message: '',
          branch: 'main',
          protected: false,
        },
      }),
    )

    expect(clone.snapshot?.dataStateAtDate.toISOString()).toBe(
      '2026-09-12T08:00:00.000Z',
    )
  })
})
