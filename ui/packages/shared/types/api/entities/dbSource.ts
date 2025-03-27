export type dbSource = {
  host: string
  port: string
  dbname: string
  username: string
  password: string
  instanceId: string
  db_list?: string[]
}

export type TestSourceDTO = {
  message: string
  status: string
}
