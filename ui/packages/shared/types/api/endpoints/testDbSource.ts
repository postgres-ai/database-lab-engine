import { dbSource, TestSourceDTO } from 'types/api/entities/dbSource'

export type TestDbSource = (values: dbSource) => Promise<{
  response: TestSourceDTO | null
  error: Response | null
}>
