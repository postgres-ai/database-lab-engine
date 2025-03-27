import { formatDateToISO } from '@postgres.ai/shared/utils/date'

export interface Branch {
  name: string
  parent: string
  dataStateAt: string
  snapshotID: string
  numSnapshots: number
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
