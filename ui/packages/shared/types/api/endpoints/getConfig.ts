import { Config } from '../entities/config'

export type GetConfig = (instanceId: string) => Promise<{
  response: Config | null
  error: Response | null
}>
