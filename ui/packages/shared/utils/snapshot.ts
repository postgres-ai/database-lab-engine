import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'

export const compareSnapshotsDesc = (a: Snapshot, b: Snapshot) =>
  b.dataStateAtDate.getTime() - a.dataStateAtDate.getTime()
