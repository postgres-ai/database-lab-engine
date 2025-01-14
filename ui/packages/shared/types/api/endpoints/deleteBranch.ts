export type DeleteBranch = (
  branchName: string,
) => Promise<{ response: Response | null; error: Error | null }>
