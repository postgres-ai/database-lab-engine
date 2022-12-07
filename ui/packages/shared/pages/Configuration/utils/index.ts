import { DatabaseType } from '@postgres.ai/shared/types/api/entities/config'
import { imageOptions } from '../imageOptions'

const extendedCustomImage = 'custom-images/extended-postgres'
// used for creating an array for postgresImages, should be incremented if a new version comes out
const versionArrayLength = 7

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

export const postUniqueDatabases = (values: string) => {
  const splitDatabaseArray = values.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)

  const databases = splitDatabaseArray.reduce((acc: DatabaseType, curr) => {
    acc[curr] = {}
    return acc
  }, {})

  const nonEmptyDatabase = Object.fromEntries(
    Object.entries(databases).filter(([name]) => name != ''),
  )

  return values.length !== 0 ? nonEmptyDatabase : null
}

export const formatDockerImageArray = (type: string) => {
  let images: string[] = []
  const versions = Array.from({ length: versionArrayLength }, (_, i) =>
    i === 0 ? i + 9.6 : Math.floor(i + 9.6),
  )

  if (type === 'Generic Postgres') {
    images = versions.map(
      (version) => `postgresai/extended-postgres:${version}`,
    )
  } else {
    images = versions.map(
      (version) =>
        `registry.gitlab.com/postgres-ai/${extendedCustomImage}-${type}:${version}`,
    )
  }

  return images
}

export const getImageType = (imageUrl: string) => {
  const postgresCustomImageType =
    imageUrl.includes(extendedCustomImage) &&
    imageUrl.split(`${extendedCustomImage}-`)[1].split(':')[0]

  if (imageUrl.includes('postgresai/extended-postgres')) {
    return 'Generic Postgres'
  } else if (
    postgresCustomImageType &&
    imageOptions.some((element) =>
      postgresCustomImageType.includes(element.type),
    )
  ) {
    return postgresCustomImageType
  } else {
    return 'custom'
  }
}
