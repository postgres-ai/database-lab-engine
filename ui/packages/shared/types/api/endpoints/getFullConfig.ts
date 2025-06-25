export type GetFullConfig = (instanceId: string) => Promise<{
  response: string | null
  error: Response | any | null
}>
