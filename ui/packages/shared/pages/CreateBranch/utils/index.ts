export const getCliCreateBranchCommand = (
  branchName: string,
  parentBranchName: string,
) => {
  return `dblab branch create ${branchName ? branchName : `<BRANCH_NAME>`} ${
    parentBranchName !== `master` ? parentBranchName : ``
  }`
}

export const getCliBranchListCommand = () => {
  return `dblab branch`
}
