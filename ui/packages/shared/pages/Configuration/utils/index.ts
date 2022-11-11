import { DatabaseType } from '@postgres.ai/shared/types/api/entities/config'

export const uniqueDatabases = (values: string) => {
  const splitDatabaseArray = values.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)
  let databaseArray = []

  for (let i in splitDatabaseArray) {
    if (
      splitDatabaseArray[i] !== '' &&
      databaseArray.indexOf(splitDatabaseArray[i]) === -1
    ) {
      databaseArray.push(splitDatabaseArray[i])
    }
  }

  return databaseArray.join(' ')
}

export const postUniqueDatabases = (values: any) => {
  const splitDatabaseArray = values.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)

  const databases = splitDatabaseArray.reduce(
    (acc: DatabaseType, curr: number) => {
      acc[curr] = {}
      return acc
    },
    {},
  )

  const nonEmptyDatabase = Object.fromEntries(
    Object.entries(databases).filter(([name]) => name != ''),
  )

  return values.length !== 0 ? nonEmptyDatabase : null
}
