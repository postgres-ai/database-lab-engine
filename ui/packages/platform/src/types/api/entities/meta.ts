export type MetaDto = {
  buildTimestamp: number
}

export const formatMetaDto = (dto: MetaDto) => dto

export type Meta = ReturnType<typeof formatMetaDto>
