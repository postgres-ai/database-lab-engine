export type CreateSnapshotDTO = {
  snapshotID: string
}

export const formatCreateSnapshotDto = (dto: CreateSnapshotDTO) => dto

export type CreateSnapshotResponse = ReturnType<typeof formatCreateSnapshotDto>
