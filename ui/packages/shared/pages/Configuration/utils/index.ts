import { DatabaseType } from '@postgres.ai/shared/types/api/entities/config'

import { dockerImageOptions } from '../configOptions'
import { FormValues } from '../useForm'

const extendedCustomImage = 'custom-images/extended-postgres'
// used for creating an array for postgresImages, should be incremented if a new version comes out
const versionArrayLength = 7

export type FormValuesKey = keyof FormValues

export const uniqueChipValue = (values: string) => {
  const splitChipArray = values.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)
  let databaseArray = []

  for (let i in splitChipArray) {
    if (
      splitChipArray[i] !== '' &&
      databaseArray.indexOf(splitChipArray[i]) === -1
    ) {
      databaseArray.push(splitChipArray[i])
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
    imageUrl.split(`${extendedCustomImage}-`)[1]?.split(':')[0]

  const formattedDockerImageArray = formatDockerImageArray(
    postgresCustomImageType || '',
  )

  const satisfiesDockerTypeAndImage =
    dockerImageOptions.some(
      (element) => element.type === postgresCustomImageType,
    ) && formattedDockerImageArray.some((image) => image === imageUrl)

  if (imageUrl.includes('postgresai/extended-postgres')) {
    return 'Generic Postgres'
  } else if (postgresCustomImageType && satisfiesDockerTypeAndImage) {
    return postgresCustomImageType
  } else {
    return 'custom'
  }
}

export const formatDatabases = (databases: DatabaseType | null) => {
  let formattedDatabases = ''

  if (databases !== null) {
    Object.keys(databases).forEach(function (key) {
      formattedDatabases += key + ' '
    })
  }

  return formattedDatabases
}

export const formatDumpCustomOptions = (options: string[] | null) => {
  let formattedOptions = ''

  if (options !== null) {
    options.forEach(function (key) {
      formattedOptions += key + ' '
    })
  }

  return formattedOptions
}

export const postUniqueCustomOptions = (options: string) => {
  const splitOptionsArray = options.split(/[,(\s)(\n)(\r)(\t)(\r\n)]/)
  const uniqueOptions = splitOptionsArray.filter(
    (item, index) => splitOptionsArray.indexOf(item) === index && item !== '',
  )
  return uniqueOptions
}
