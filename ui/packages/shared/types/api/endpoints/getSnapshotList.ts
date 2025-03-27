export interface SnapshotList {
  branch: string[]
  id: string
  dataStateAt: string
  message: string
}

export type GetSnapshotList = (
  branchName: string,
  instanceId: string,
) => Promise<{
  response: SnapshotList[] | null
  error: Response | null
}>
