export interface SnapshotList {
  branch: string[]
  id: string
  dataStateAt: string
  comment?: string
}

export type GetSnapshotList = (branchName: string) => Promise<{
  response: SnapshotList[] | null
  error: Response | null
}>
