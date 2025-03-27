import { parseDate } from '@postgres.ai/shared/utils/date'
import { SnapshotDto } from './snapshot'

export const formatBranchSnapshotDto = (dto: SnapshotDto[]) =>
  dto.map((item) => ({
    ...item,
    numClones: item.numClones.toString(),
    createdAtDate: parseDate(item.createdAt),
    dataStateAtDate: parseDate(item.dataStateAt),
  }))
