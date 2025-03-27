import { CreateSnapshotResponse } from '@postgres.ai/shared/types/api/entities/createSnapshot'

export type CreateSnapshot = (
  cloneID: string,
  message: string,
  instanceId: string,
) => Promise<{
  response: CreateSnapshotResponse | null
  error: Response | null
}>
