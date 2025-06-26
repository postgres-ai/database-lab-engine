import { DatabaseType } from '@postgres.ai/shared/types/api/entities/config'

import { dockerImageOptions } from '../configOptions'
import { FormValues } from '../useForm'

const seContainerRegistry = 'se-images'
const genericImagePrefix = 'postgresai/extended-postgres'
// Predefined list of Docker images for UI display
// This list is shown to users for convenient selection
// IMPORTANT: if user specified an image in config that's not in this list,
// it will be automatically added via createEnhancedDockerImages()
const dockerImagesConfig = {
  '9.6': ['0.5.3', '0.5.2', '0.5.1'],
  '10': ['0.5.3', '0.5.2', '0.5.1'],
  '11': ['0.5.3', '0.5.2', '0.5.1'],
  '12': ['0.5.3', '0.5.2', '0.5.1'],
  '13': ['0.5.3', '0.5.2', '0.5.1'],
  '14': ['0.5.3', '0.5.2', '0.5.1'],
  '15': ['0.5.3', '0.5.2', '0.5.1'],
  '16': ['0.5.3', '0.5.2', '0.5.1'],
  '17': ['0.5.3', '0.5.2', '0.5.1'],
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
  if (!pgImage) return undefined
  
  try {
    const pgImageVersion = pgImage.split(':')[1]
    if (!pgImageVersion) return undefined
    
    const pgServerVersion = pgImageVersion.split('-')[0]
    if (!pgServerVersion) return undefined
    
    return pgServerVersion.includes('.')
      ? pgServerVersion.split('.')[0]
      : pgServerVersion
  } catch (error) {
    // Return undefined for malformed image strings
    return undefined
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

export const customOrGenericImage = (dockerImage: string | undefined) =>
  dockerImage === 'Generic Postgres' || dockerImage === 'custom'

export const createFallbackDockerImage = (
  dockerPath: string,
  dockerTag: string,
): DockerImage => {
  const majorVersion = getImageMajorVersion(dockerPath) || '17' // Default to 17 if version can't be extracted
  
  return {
    package_group: 'postgresai',
    pg_major_version: majorVersion,
    tag: dockerTag || `${majorVersion}-custom`,
    location: dockerPath,
  }
}

// Creates enhanced list of Docker images, including image from configuration
export const createEnhancedDockerImages = (
  configDockerPath?: string,
  configDockerTag?: string,
): DockerImage[] => {
  let enhancedImages = [...genericDockerImages]

  // If there's an image in config, check if we need to add it
  if (configDockerPath && configDockerTag) {
    const existingImage = genericDockerImages.find(
      (image) => image.location === configDockerPath || image.tag === configDockerTag
    )

    // If image not found in predefined list, add it
    if (!existingImage) {
      // Check if this is a Generic Postgres image
      if (configDockerPath.includes(genericImagePrefix)) {
        // For Generic Postgres images create proper structure
        const majorVersion = getImageMajorVersion(configDockerPath)
        if (majorVersion) {
          const configImage: DockerImage = {
            package_group: 'postgresai',
            pg_major_version: majorVersion,
            tag: configDockerTag,
            location: configDockerPath,
          }
          enhancedImages.push(configImage)
        } else {
          // Fallback if version extraction failed
          const configImage = createFallbackDockerImage(configDockerPath, configDockerTag)
          enhancedImages.push(configImage)
        }
      } else {
        // For custom images use fallback
        const configImage = createFallbackDockerImage(configDockerPath, configDockerTag)
        enhancedImages.push(configImage)
      }
    }
  }

  return enhancedImages
}

// Checks if image is loaded from configuration (not from predefined list)
export const isConfigLoadedImage = (
  dockerPath: string,
  dockerTag: string,
): boolean => {
  const existingImage = genericDockerImages.find(
    (image) => image.location === dockerPath || image.tag === dockerTag
  )
  return !existingImage
}
