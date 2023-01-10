export type DestroySnapshot = (snapshotId: string) => Promise<{
  response: true | null
  error: Response | null
}>
