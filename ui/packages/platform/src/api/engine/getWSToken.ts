import { request } from 'helpers/request'
import { formatWSTokenDto, WSTokenDTO } from '@postgres.ai/shared/types/api/entities/wsToken'
import { GetWSToken } from "@postgres.ai/shared/types/api/endpoints/getWSToken";

export const getWSToken: GetWSToken = async (req ) => {
  const response = await request('/admin/ws-auth')

  return {
    response: response.ok
      ? formatWSTokenDto((await response.json()) as WSTokenDTO)
      : null,
    error: response.ok ? null : response,
  }
}
