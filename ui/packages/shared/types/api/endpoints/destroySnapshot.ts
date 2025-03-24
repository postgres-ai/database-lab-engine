export type DestroySnapshot = (
  snapshotId: string,
  forceDelete: boolean,
) => Promise<{
  response: boolean | null
  error: Response | null
}>
