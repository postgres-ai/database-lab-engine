import { Engine } from '@postgres.ai/ce/src/types/api/entities/engine'

export type GetEngine = () => Promise<{
  response: Engine | null
  error: Response | null
}>
