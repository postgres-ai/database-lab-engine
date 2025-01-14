export type DestroySnapshot = (snapshotId: string) => Promise<{
  response: boolean | null
  error: Response | null
}>
