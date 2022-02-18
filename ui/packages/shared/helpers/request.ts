import 'whatwg-fetch'

type RequestParams = Record<string, unknown>

export type RequestOptions = RequestInit & {
  params?: RequestParams
}

const serializeParams = (params: RequestParams | null) => {
  if (!params) return null

  const searchParams = new URLSearchParams()

  Object.entries(params).map((param) => {
    const [key, value] = param
    searchParams.append(key, String(value))
  })

  return searchParams.toString()
}

const createUrl = (path: string, params: RequestParams | null) => {
  const serializedParams = serializeParams(params)
  const queryString = serializedParams ? `?${serializedParams}` : ''
  return `${path}${queryString}`
}

export const request = async (path: string, options?: RequestOptions) => {
  const { params = null, ...requestInit } = options ?? {}

  const url = createUrl(path, params)

  try {
    return await window.fetch(url, {
      ...requestInit,
      headers: {
        'Content-Type': 'application/json',
        ...requestInit?.headers,
      },
    })
  } catch (e) {
    return new Response(null, {
      status: 500,
      statusText: `Unknown error`,
    })
  }
}
