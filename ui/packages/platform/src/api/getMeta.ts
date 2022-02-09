import { request } from '@postgres.ai/shared/helpers/request'

import { MetaDto, formatMetaDto } from 'types/api/entities/meta'

export const getMeta = async () => {
  const response = await request('/meta.json')

  return {
    response: response.ok ? formatMetaDto(await response.json() as MetaDto) : null,
    error: response.ok ? null : response,
  }
}
