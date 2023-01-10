export interface GetBranchesResponseType {
  name: string
  parent: string
  dataStateAt: string
  snapshotID: string
}

export type GetBranches = () => Promise<{
  response: GetBranchesResponseType[] | null
  error: Response | null
}>
