export const getCliCreateBranchCommand = (branchName: string) => {
  return `dblab branch create ${branchName ? branchName : `<BRANCH_NAME>`}`
}

export const getCliBranchListCommand = () => {
  return `dblab branch`
}
