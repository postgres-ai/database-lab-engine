import { Clone } from '@postgres.ai/shared/types/api/entities/clone'

export type CreateClone = (args: {
  instanceId: string
  cloneId: string
  snapshotId: string
  dbUser: string
  dbPassword: string
  isProtected: boolean
}) => Promise<{ response: Clone | null; error: Response | null }>
