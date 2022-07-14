export type WSTokenDTO = {
  token: string
}

export const formatWSTokenDto = (dto: WSTokenDTO) => dto

export type WSToken = ReturnType<typeof formatWSTokenDto>
