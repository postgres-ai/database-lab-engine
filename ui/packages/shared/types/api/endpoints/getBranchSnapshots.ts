import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'

export type GetBranchSnapshots = (
  snapshotId: string,
) => Promise<{ response: Snapshot[] | null; error: Response | null }>
