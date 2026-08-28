export type UpgradeClone = (args: {
  instanceId: string
  cloneId: string
  targetVersion: number
  dockerImage?: string
}) => Promise<{
  response: true | null
  error: Response | null
}>
