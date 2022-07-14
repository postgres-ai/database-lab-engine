import { WSToken } from '@postgres.ai/shared/types/api/entities/wsToken'

export type GetWSToken = (args: { instanceId: string }) => Promise<{
  response: WSToken | null
  error: Response | null
}>
