import { DatabaseType } from '@postgres.ai/shared/types/api/entities/config'

import { dockerImageOptions } from '../configOptions'
import { FormValues } from '../useForm'

const seContainerRegistry = 'se-images'
const genericImagePrefix = 'postgresai/extended-postgres'

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

export const genericDockerImages = [
  {
    package_group: 'postgresai',
    pg_major_version: '9.6',
    tag: '9.6-0.3.0',
    location: `${genericImagePrefix}:9.6-0.3.0`,
  },
  {
    package_group: 'postgresai',
    pg_major_version: '10',
    tag: '10-0.3.0',
    location: `${genericImagePrefix}:10-0.3.0`,
  },
  {
    package_group: 'postgresai',
    pg_major_version: '11',
    tag: '11-0.3.0',
    location: `${genericImagePrefix}:11-0.3.0`,
  },
  {
    package_group: 'postgresai',
    pg_major_version: '12',
    tag: '12-0.3.0',
    location: `${genericImagePrefix}:12-0.3.0`,
  },
  {
    package_group: 'postgresai',
    pg_major_version: '13',
    tag: '13-0.3.0',
    location: `${genericImagePrefix}:13-0.3.0`,
  },
  {
    package_group: 'postgresai',
    pg_major_version: '14',
    tag: '14-0.3.0',
    location: `${genericImagePrefix}:14-0.3.0`,
  },
  {
    package_group: 'postgresai',
    pg_major_version: '15',
    tag: '15-0.3.0',
    location: `${genericImagePrefix}:15-0.3.0`,
  },
]

export const isSeDockerImage = (dockerImage: string | undefined) => {
  const dockerImageType =
    dockerImage?.includes(seContainerRegistry) &&
    dockerImage.split(`${seContainerRegistry}/`)[1]?.split(':')[0]

  return dockerImageOptions.some((element) => element.type === dockerImageType)
}

export const getImageType = (imageUrl: string) => {
  const postgresCustomImageType =
    imageUrl.includes(seContainerRegistry) &&
    imageUrl.split(`${seContainerRegistry}/`)[1]?.split(':')[0]

  if (imageUrl.includes(genericImagePrefix)) {
    return 'Generic Postgres'
  } else if (postgresCustomImageType && isSeDockerImage(imageUrl)) {
    return postgresCustomImageType
  } else {
    return 'custom'
  }
}

export const getImageMajorVersion = (pgImage: string | undefined) => {
  const pgImageVersion = pgImage?.split(':')[1]
  const pgServerVersion = pgImageVersion?.split('-')[0]
  return pgServerVersion?.includes('.')
    ? pgServerVersion?.split('.')[0]
    : pgServerVersion
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

export const customOrGenericImage = (dockerImage: string | undefined) =>
  dockerImage === 'Generic Postgres' || dockerImage === 'custom'
