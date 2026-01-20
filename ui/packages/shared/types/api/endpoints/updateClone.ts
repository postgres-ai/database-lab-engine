export type UpdateClone = (args: {
  instanceId: string
  cloneId: string
  clone: {
    isProtected: boolean
    protectionDurationMinutes?: number
  }
}) => Promise<{
  response: true | null
  error: Response | null
}>
