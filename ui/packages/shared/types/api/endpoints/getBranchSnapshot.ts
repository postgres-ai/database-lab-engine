import { BranchSnapshotDto } from '@postgres.ai/shared/types/api/entities/branchSnapshot'

export type GetBranchSnapshot = (
  snapshotId: string,
  instanceId: string,
) => Promise<{ response: BranchSnapshotDto | null; error: Response | null }>
