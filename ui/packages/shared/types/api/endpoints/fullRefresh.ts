export type FullRefresh = (args: {
  instanceId: string
}) => Promise<{
  response: string | null
  error: Response | null
}>
