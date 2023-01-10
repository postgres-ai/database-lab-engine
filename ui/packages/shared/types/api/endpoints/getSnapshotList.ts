export interface GetSnapshotListResponseType {
  branch: string[]
  id: string
  dataStateAt: string
  comment?: string
}

export type GetSnapshotList = (branchName: string) => Promise<{
  response: GetSnapshotListResponseType[] | null
  error: Response | null
}>
