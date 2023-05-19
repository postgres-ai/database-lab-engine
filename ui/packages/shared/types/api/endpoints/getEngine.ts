export type EngineDto = {
  version: string
  edition?: string
}

export type GetEngine = () => Promise<{
  response: EngineType | null
  error: Response | null
}>

export const formatEngineDto = (dto: EngineDto) => dto

export type EngineType = ReturnType<typeof formatEngineDto>
