import { InstanceState } from '@postgres.ai/shared/types/api/entities/instanceState'

export const getTypeByStatus = (
  status: Exclude<InstanceState['retrieving'], undefined>['status'],
) => {
  if (status === 'finished') return 'ok'
  if (status === 'refreshing') return 'waiting'
  if (status === 'failed') return 'error'
  return 'unknown'
}
