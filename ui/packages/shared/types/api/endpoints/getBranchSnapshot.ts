import { BranchSnapshotDto } from '@postgres.ai/shared/types/api/entities/branchSnapshot'

export type GetBranchSnapshot = (
  snapshotId: string,
) => Promise<{ response: BranchSnapshotDto | null; error: Response | null }>
