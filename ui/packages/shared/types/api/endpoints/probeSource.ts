export type ProbeSourceRequest = {
  url: string
  password: string
}

export type SourceConnection = {
  host: string
  port: number
  username: string
  dbname: string
}

export type ProposedConfig = {
  source: SourceConnection
  detectedProvider: string
  dockerImage: string
  dockerTag: string
  pgMajorVersion: number
  databases: string[]
  sharedBuffers: string
  memoryProbed: boolean
  sharedPreloadLibraries: string
  queryTuning: { [key: string]: string }
}

export type ProbeSourceError = {
  status: number
  message: string
}

export type ProbeSource = (req: ProbeSourceRequest) => Promise<{
  response: ProposedConfig | null
  error: ProbeSourceError | null
}>
