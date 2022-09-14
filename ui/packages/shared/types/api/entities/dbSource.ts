export type dbSource = {
  host: string
  port: string
  dbname: string
  username: string
  password: string
  db_list?: string[]
}

export type TestSourceDTO = {
  message: string
  status: string
}