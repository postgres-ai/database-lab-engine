import {
  RequestOptions,
  request as requestCore,
} from '@postgres.ai/shared/helpers/request'

const sign = require('jwt-encode')

export const SI_API_SERVER = 'https://si.dblab.dev'

export const JWT_SECRET = 'some-jwt-secret'
export const JWT_PAYLOAD = (userID?: number) => ({
  id: userID?.toString(),
})
export const JWT_HEADER = {
  alg: 'HS256',
  typ: 'JWT',
}

export const simpleInstallRequest = async (
  path: string,
  options: RequestOptions,
  userID?: number,
) => {
  const jwtToken = sign(JWT_PAYLOAD(userID), JWT_SECRET, JWT_HEADER)

  const response = await requestCore(`${SI_API_SERVER}${path}`, {
    ...options,
    headers: {
      Authorization: `Bearer ${jwtToken}`,
    },
  })

  return response
}
