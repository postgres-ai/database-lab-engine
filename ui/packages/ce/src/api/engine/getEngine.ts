import {
  EngineDto,
  formatEngineDto,
} from '@postgres.ai/shared/types/api/endpoints/getEngine'
import { request } from 'helpers/request'

export const getEngine = async () => {
  const response = await request('/healthz')

  return {
    response: response.ok
      ? formatEngineDto((await response.json()) as EngineDto)
      : null,
    error: response.ok ? null : response,
  }
}
