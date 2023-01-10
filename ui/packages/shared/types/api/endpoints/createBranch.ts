import { CreateBranchResponse } from '@postgres.ai/shared/types/api/entities/createBranch'

export type CreateBranchFormValues = {
  branchName: string
  baseBranch: string
  snapshotID: string
  creationType?: 'branch' | 'snapshot'
}

export type CreateBranch = (values: CreateBranchFormValues) => Promise<{
  response: CreateBranchResponse | null
  error: Response | null
}>
