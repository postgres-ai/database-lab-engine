import { parseISO9075Date } from '@postgres.ai/shared/utils/date'

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
  createdAtDate: parseISO9075Date(dto.createdAt),
  dataStateAtDate: parseISO9075Date(dto.dataStateAt)
})

export type Snapshot = ReturnType<typeof formatSnapshotDto>
