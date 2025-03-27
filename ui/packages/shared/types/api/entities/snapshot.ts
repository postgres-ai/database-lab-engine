import { parseDate } from '@postgres.ai/shared/utils/date'

export type SnapshotDto = {
  numClones: string | number
  clones: string[]
  createdAt: string
  dataStateAt: string
  id: string
  pool: string
  physicalSize: number
  logicalSize: number
  message: string
  branch: string
}

export const formatSnapshotDto = (dto: SnapshotDto) => ({
  ...dto,
  createdAtDate: parseDate(dto.createdAt),
  dataStateAtDate: parseDate(dto.dataStateAt),
})

export type Snapshot = ReturnType<typeof formatSnapshotDto>
