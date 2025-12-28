import { Config, ConfigUpdateResponse } from '@postgres.ai/shared/types/api/entities/config'

export type UpdateConfigInput = Config

export type UpdateConfig = (
  values: UpdateConfigInput,
  instanceId: string,
) => Promise<{
  response: ConfigUpdateResponse | null
  error: Response | null
}>
