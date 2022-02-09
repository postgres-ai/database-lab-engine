import { request } from 'helpers/request'
import { EngineDto, formatEngineDto } from 'types/api/entities/engine'

export const getEngine = async () => {
  const response = await request('/healthz')

  return {
    response: response.ok
      ? formatEngineDto((await response.json()) as EngineDto)
      : null,
    error: response.ok ? null : response,
  }
}
