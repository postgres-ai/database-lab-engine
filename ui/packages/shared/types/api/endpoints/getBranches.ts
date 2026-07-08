import { formatDateToISO } from '@postgres.ai/shared/utils/date'

export interface Branch {
  name: string
  baseDataset: string
  parent: string
  dataStateAt: string
  snapshotID: string
  dataset: string
  numSnapshots: number
  protected: boolean
  protectedTill?: string
  deleteAt?: string
}

export const formatBranchesDto = (dto: Branch[]) =>
  dto.map((item) => ({
    ...item,
    dataStateAt: formatDateToISO(item.dataStateAt),
  }))

export type GetBranches = (instanceId: string) => Promise<{
  response: Branch[] | null
  error: Response | null
}>
