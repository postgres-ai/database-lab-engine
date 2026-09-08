export type UpgradeClone = (args: {
  instanceId: string
  cloneId: string
  dockerImage?: string
}) => Promise<{
  response: true | null
  error: Response | null
}>
