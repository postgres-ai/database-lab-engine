import { InstanceRetrievalType } from '@postgres.ai/shared/types/api/entities/instanceRetrieval'

export type GetInstanceRetrieval = (args: { instanceId: string }) => Promise<{
  response: InstanceRetrievalType | null
  error: Response | null
}>
