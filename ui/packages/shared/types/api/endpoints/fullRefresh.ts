export type FullRefresh = (args: {
  instanceId: string
}) => Promise<{
  response: true | null
  error: Response | null
}>
