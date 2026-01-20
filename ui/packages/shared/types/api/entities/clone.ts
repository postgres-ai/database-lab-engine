/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { parseDate } from '@postgres.ai/shared/utils/date'
import {
  SnapshotDto,
  formatSnapshotDto,
} from '@postgres.ai/shared/types/api/entities/snapshot'

export type CloneDto = {
  createdAt: string
  id: string
  branch: string
  status: {
    code: 'OK' | 'CREATING' | 'DELETING' | 'RESETTING' | 'FATAL'
    message: string
  }
  protected: boolean
  protectedTill?: string
  metadata: {
    cloneDiffSize: number
    cloningTime: number
    protectionLeaseDurationMinutes?: number
    protectionRenewalDurationMinutes?: number
  }
  db: {
    username: string
    port: string
    host: string
  }
  // Possible bug, when snapshot is null.
  snapshot: SnapshotDto | null
}

export const formatCloneDto = (dto: CloneDto) => ({
  ...dto,
  createdAt: dto.createdAt,
  createdAtDate: parseDate(dto.createdAt),
  protectedTillDate: dto.protectedTill ? parseDate(dto.protectedTill) : null,
  snapshot: dto.snapshot ? formatSnapshotDto(dto.snapshot) : null,
})

export type Clone = ReturnType<typeof formatCloneDto>
