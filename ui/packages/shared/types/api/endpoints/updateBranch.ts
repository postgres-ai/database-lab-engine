export type UpdateBranch = (args: {
  instanceId: string
  branchName: string
  branch: {
    isProtected: boolean
    protectionDurationMinutes?: number
  }
}) => Promise<{
  response: true | null
  error: Response | null
}>
