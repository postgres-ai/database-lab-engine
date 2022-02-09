import { Instance } from '@postgres.ai/shared/types/api/entities/instance'

export type GetInstance = (args: { instanceId: string }) => Promise<{
  response: Instance | null
  error: Response | null
}>
