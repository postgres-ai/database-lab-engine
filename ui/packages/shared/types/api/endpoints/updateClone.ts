export type UpdateClone = (args: {
  instanceId: string
  cloneId: string
  clone: {
    isProtected: boolean
  }
}) => Promise<{
  response: true | null
  error: Response | null
}>
