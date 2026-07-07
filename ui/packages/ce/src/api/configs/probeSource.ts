import {
  ProbeSourceRequest,
  ProposedConfig,
  ProbeSourceError,
} from '@postgres.ai/shared/types/api/endpoints/probeSource'
import { request } from 'helpers/request'

const BAD_REQUEST_MESSAGE =
  'Could not reach the source database. Check the connection string and password.'
const SERVER_ERROR_MESSAGE =
  'Internal server error while probing the source database.'

const readError = async (
  response: Response,
): Promise<ProbeSourceError> => {
  const fallback =
    response.status >= 500 ? SERVER_ERROR_MESSAGE : BAD_REQUEST_MESSAGE

  try {
    const body = (await response.json()) as { message?: string }
    return { status: response.status, message: body?.message || fallback }
  } catch {
    return { status: response.status, message: fallback }
  }
}

export const probeSource = async (
  req: ProbeSourceRequest,
): Promise<{
  response: ProposedConfig | null
  error: ProbeSourceError | null
}> => {
  const response = await request('/admin/probe-source', {
    method: 'POST',
    body: JSON.stringify({ url: req.url, password: req.password }),
  })

  if (response.ok) {
    return { response: (await response.json()) as ProposedConfig, error: null }
  }

  return { response: null, error: await readError(response) }
}
