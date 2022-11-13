export type GetFullConfig = () => Promise<{
  response: string | null
  error: Response | any | null
}>
