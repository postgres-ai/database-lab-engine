export const getCliCreateBranchCommand = (
  branchName: string,
  parentBranchName: string,
) => {
  const branchArg = branchName || `<BRANCH_NAME>`;
  const parentBranchArg =
    parentBranchName && parentBranchName !== `main`
      ? `--parent-branch ${parentBranchName}`
      : ``;

  return `dblab branch ${branchArg} ${parentBranchArg}`.trim();
};



export const getCliBranchListCommand = () => {
  return `dblab branch`
}
