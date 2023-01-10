export const generateSnapshotPageId = (id: string) => {
  const splitSnapshotId = id?.split(`@`)[1]
  const snapshotPageId = splitSnapshotId?.includes('snapshot_')
    ? splitSnapshotId?.split('snapshot_')[1]
    : splitSnapshotId

  return snapshotPageId
}
