export type UpdateSnapshot = (args: {
  instanceId: string
  snapshotId: string
  snapshot: {
    isProtected: boolean
    protectionDurationMinutes?: number
  }
}) => Promise<{
  response: true | null
  error: Response | null
}>
