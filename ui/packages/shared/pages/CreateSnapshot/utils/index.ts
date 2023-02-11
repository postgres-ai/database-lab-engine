export const getCliCreateSnapshotCommand = (cloneID: string) => {
  return `dblab branch create ${cloneID ? cloneID : `<CLONE_ID>`}`
}
