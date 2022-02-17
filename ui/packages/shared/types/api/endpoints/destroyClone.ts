export type DestroyClone = (args: {
  instanceId: string
  cloneId: string
}) => Promise<{
  response: true | null
  error: Response | null
}>
