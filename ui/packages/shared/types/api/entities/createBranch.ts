export type CreateBranchDTO = {
  name: string
}

export const formatCreateBranchDto = (dto: CreateBranchDTO) => dto

export type CreateBranchResponse = ReturnType<typeof formatCreateBranchDto>
