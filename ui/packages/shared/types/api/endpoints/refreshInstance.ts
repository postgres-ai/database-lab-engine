export type RefreshInstance = (args: { instanceId: string }) => Promise<{
  response: boolean | null
  error: Response | null
}>
