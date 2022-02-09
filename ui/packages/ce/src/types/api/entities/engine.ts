export type EngineDto = {
  version: string
}

export const formatEngineDto = (dto: EngineDto) => dto

export type Engine = ReturnType<typeof formatEngineDto>
