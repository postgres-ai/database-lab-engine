import { Snapshot } from '@postgres.ai/shared/types/api/entities/snapshot'

export type GetSnapshots = (args: { instanceId: string, branchName?: string, dataset?: string }) => Promise<{
  response: Snapshot[] | null
  error: Response | null
}>
