import { Clone } from '@postgres.ai/shared/types/api/entities/clone'

export type GetClone = (args: {
  instanceId: string
  cloneId: string
}) => Promise<{ response: Clone | null; error: Response | null }>
