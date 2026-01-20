export type UpdateClone = (args: {
  instanceId: string
  cloneId: string
  clone: {
    isProtected: boolean
    renewLease?: boolean
  }
}) => Promise<{
  response: true | null
  error: Response | null
}>
