export type GetInstanceRetrieval = (args: { instanceId: string }) => Promise<{
  response: Response | null
  error: Response | null
}>
