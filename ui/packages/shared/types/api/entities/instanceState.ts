import {
  CloneDto,
  formatCloneDto,
} from '@postgres.ai/shared/types/api/entities/clone'
import {
  formatPoolDto,
  PoolDto,
} from '@postgres.ai/shared/types/api/entities/pool'

export type InstanceStateDto = {
  cloning: {
    clones: CloneDto[]
    expectedCloningTime: number
    numClones?: number
    protectionLeaseDurationMinutes?: number
    protectionMaxDurationMinutes?: number
  }
  status: {
    code: 'OK' | 'WARNING' | 'NO_RESPONSE'
    message: string
  }
  // Fallback for capability with old API.
  clones?: CloneDto[]
  // Fallback for capability with old API.
  expectedCloningTime?: number
  // Fallback for capability with old API.
  pools?: PoolDto[]
  // Fallback for capability with old API.
  retrieving?: {
    lastRefresh: string | null
    nextRefresh: string | null
    mode: string
    status: 'finished' | 'failed' | 'refreshing' | 'pending'
    alerts?: {
      refresh_failed?: {
        level: 'error'
        message: string
        lastSeen: string
        count: number
      }
    }
  }
  // Fallback for capability with old API.
  fileSystem?: {
    free: number
    size: number
    used: number
  }
  engine?: {
    version: string
    startedAt: string
    edition?: string
    disableConfigModification?: boolean
  }
  dataSize?: number
}

export const formatInstanceStateDto = (dto: InstanceStateDto) => {
  if (!dto) return null

  const pools = dto.pools?.map(formatPoolDto) ?? null
  const clones =
    dto?.clones?.map(formatCloneDto) ?? dto.cloning?.clones?.map(formatCloneDto)
  const expectedCloningTime =
    dto?.expectedCloningTime ?? dto.cloning?.expectedCloningTime

  return {
    ...dto,
    engine: {
      version: dto.engine?.version ?? null,
      startedAt: dto.engine?.startedAt && new Date(dto.engine?.startedAt),
      disableConfigModification: false,
    },
    retrieving: dto.retrieving && {
      lastRefresh: dto.retrieving.lastRefresh
        ? new Date(dto.retrieving.lastRefresh)
        : null,
      nextRefresh: dto.retrieving.nextRefresh
        ? new Date(dto.retrieving.nextRefresh)
        : null,
      mode: dto.retrieving.mode,
      status: dto.retrieving.status,
      alerts: dto.retrieving.alerts && {
        refreshFailed: dto.retrieving.alerts.refresh_failed && {
          level: dto.retrieving.alerts.refresh_failed.level,
          message: dto.retrieving.alerts.refresh_failed.message,
          lastSeen: new Date(dto.retrieving.alerts.refresh_failed.lastSeen),
          count: dto.retrieving.alerts.refresh_failed.count,
        },
      },
    },
    cloning: {
      clones: clones,
      expectedCloningTime: expectedCloningTime,
      protectionLeaseDurationMinutes: dto.cloning?.protectionLeaseDurationMinutes,
      protectionMaxDurationMinutes: dto.cloning?.protectionMaxDurationMinutes,
    },
    pools,
    dataSize:
      dto.dataSize ??
      pools?.reduce((sum, pool) => pool.fileSystem.dataSize + sum, 0) ??
      null,
  }
}

export type InstanceState = ReturnType<typeof formatInstanceStateDto>
