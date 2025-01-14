import { formatDateToISO } from '@postgres.ai/shared/utils/date'

export interface Branch {
  name: string
  parent: string
  dataStateAt: string
  snapshotID: string
}

export const formatBranchesDto = (dto: Branch[]) =>
  dto.map((item) => ({
    ...item,
    dataStateAt: formatDateToISO(item.dataStateAt),
  }))

export type GetBranches = () => Promise<{
  response: Branch[] | null
  error: Response | null
}>
