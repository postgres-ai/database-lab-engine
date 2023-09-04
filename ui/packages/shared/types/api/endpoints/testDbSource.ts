import { dbSource } from '@postgres.ai/shared/types/api/entities/dbSource'

export type TestDbSource = (values: dbSource) => Promise<{
  response: {
    status: number
    message: string
    result: string
    dbVersion: number
    tuningParams: {
      [key: string]: string
    }
  } | null
  error: {
    status: number
    message: string
  }
}>

export const formatTuningParams = (
  tuningParams: { [key: string]: string } | undefined,
) => {
  let formattedTuningParams = ''

  if (tuningParams && Object.keys(tuningParams).length > 0) {
    Object.entries(tuningParams).forEach(([key, value], index) => {
      if (key !== 'shared_preload_libraries' && key !== 'shared_buffers') {
        formattedTuningParams += `${key}=${value}\n`
      }
    })
    formattedTuningParams = formattedTuningParams.slice(0, -1)
  }

  return formattedTuningParams
}

export const formatTuningParamsToObj = (tuningParams: string | undefined) => {
  let formattedTuningParams: { [key: string]: string } = {}

  if (tuningParams) {
    const tuningParamsArr = tuningParams.split('\n')
    tuningParamsArr.forEach((param) => {
      const paramArr = param.split('=')
      formattedTuningParams[paramArr[0]] = paramArr[1]
    })
  }

  return formattedTuningParams
}
