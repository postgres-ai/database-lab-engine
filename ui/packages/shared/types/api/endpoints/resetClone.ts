export type ResetClone = (args: {
  instanceId: string
  cloneId: string
  snapshotId: string
}) => Promise<{
  response: true | null
  error: Response | null
}>
