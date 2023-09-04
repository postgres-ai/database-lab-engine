export const getTypeByStatus = (status: string | undefined) => {
  if (status === 'finished') return 'ok'
  if (status === 'refreshing') return 'waiting'
  if (status === 'failed') return 'error'
  return 'unknown'
}

export const isRetrievalUnknown = (mode: string | undefined) => {
  return mode === 'unknown' || mode === ''
}
