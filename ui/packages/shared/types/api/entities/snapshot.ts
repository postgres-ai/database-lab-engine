import { parseDate } from '@postgres.ai/shared/utils/date'

export type SnapshotDto = {
  createdAt: string
  dataStateAt: string
  id: string
  pool: string
  physicalSize: number
  logicalSize: number
}

export const formatSnapshotDto = (dto: SnapshotDto) => ({
  ...dto,
  createdAtDate: parseDate(dto.createdAt),
  dataStateAtDate: parseDate(dto.dataStateAt)
})

export type Snapshot = ReturnType<typeof formatSnapshotDto>
