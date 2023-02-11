export const getCliResetCloneCommand = (cloneId: string) => {
  return `dblab clone reset ${cloneId ? cloneId : `<CLONE_ID>`}`
}

export const getCliDestroyCloneCommand = (cloneId: string) => {
  return `dblab clone destroy ${cloneId ? cloneId : `<CLONE_ID>`}`
}

export const getCliProtectedCloneCommand = (enabled: boolean) => {
  return `dblab clone update --protected ${enabled ? '' : 'false'}`
}

export const getCreateSnapshotCommand = (cloneId: string) => {
  return `dblab branch snapshot --clone-id ${cloneId}`
}
