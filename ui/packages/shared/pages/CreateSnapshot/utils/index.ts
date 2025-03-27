export const getCliCreateSnapshotCommand = (
  cloneID: string,
  message: string,
) => {
  return `dblab commit --clone-id ${cloneID || '<CLONE_ID>'} --message ${
    message || '<MESSAGE>'
  }`
}
