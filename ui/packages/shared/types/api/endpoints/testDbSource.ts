import {
  dbSource,
  TestSourceDTO,
} from '@postgres.ai/shared/types/api/entities/dbSource'

export type TestDbSource = (values: dbSource) => Promise<{
  response: TestSourceDTO | null
  error: Response | null
}>
