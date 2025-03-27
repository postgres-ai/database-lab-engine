export type DeleteBranch = (
  branchName: string,
  instanceId: string,
) => Promise<{ response: Response | null; error: Error | null }>
