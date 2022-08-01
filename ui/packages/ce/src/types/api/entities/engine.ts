export type EngineDto = {
  version: string
  edition?: string
}

export const formatEngineDto = (dto: EngineDto) => dto

export type Engine = ReturnType<typeof formatEngineDto>
