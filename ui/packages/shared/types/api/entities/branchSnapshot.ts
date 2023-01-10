export type BranchSnapshotDTO = {
  message: string
  branch: string[]
}

export const formatBranchSnapshotDto = (dto: BranchSnapshotDTO) => dto

export type BranchSnapshotDto = ReturnType<typeof formatBranchSnapshotDto>
