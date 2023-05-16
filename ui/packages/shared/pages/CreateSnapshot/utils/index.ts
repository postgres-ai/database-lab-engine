export const getCliCreateSnapshotCommand = (cloneID: string) => {
  return `dblab snapshot create ${cloneID ? cloneID : `<CLONE_ID>`}`
}
