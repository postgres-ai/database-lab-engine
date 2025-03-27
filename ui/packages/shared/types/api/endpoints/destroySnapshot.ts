export type DestroySnapshot = (
  snapshotId: string,
  forceDelete: boolean,
  instanceId: string,
) => Promise<{
  response: boolean | null
  error: Response | null
}>
