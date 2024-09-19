import { DatabaseType } from '@postgres.ai/shared/types/api/entities/config'

import { dockerImageOptions } from '../configOptions'
import { FormValues } from '../useForm'

const seContainerRegistry = 'se-images'
const genericImagePrefix = 'postgresai/extended-postgres'
// since some tags are rc, we need to specify the exact tags to use
const dockerImagesConfig = {
  '9.6': ['0.5.0', '0.4.6', '0.4.5'],
  '10': ['0.5.0', '0.4.6', '0.4.5'],
  '11': ['0.5.0', '0.4.6', '0.4.5'],
  '12': ['0.5.0', '0.4.6', '0.4.5'],
  '13': ['0.5.0', '0.4.6', '0.4.5'],
  '14': ['0.5.0', '0.4.6', '0.4.5'],
  '15': ['0.5.0', '0.4.6', '0.4.5'],
  '16': ['0.5.0', '0.4.6', '0.4.5'],
  '17rc1': ['0.5.0'],
}

export type FormValuesKey = keyof FormValues

interface DockerImage {
  package_group: string
  pg_major_version: string
  tag: string
  location: string
}

type DockerImagesConfig = Record<string, string[]>

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

const createDockerImages = (
  dockerImagesConfig: DockerImagesConfig,
): DockerImage[] => {
  const dockerImages: DockerImage[] = []

  for (const pg_major_version in dockerImagesConfig) {
    if (dockerImagesConfig.hasOwnProperty(pg_major_version)) {
      const customTags = dockerImagesConfig[pg_major_version]

      customTags.forEach((tag) => {
        const image: DockerImage = {
          package_group: 'postgresai',
          pg_major_version,
          tag: `${pg_major_version}-${tag}`,
          location: `${genericImagePrefix}:${pg_major_version}-${tag}`,
        }

        dockerImages.push(image)
      })
    }
  }

  return dockerImages
}

export const genericDockerImages = createDockerImages(dockerImagesConfig)

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
