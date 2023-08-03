import {
  formatDatabases,
  formatDumpCustomOptions,
  getImageMajorVersion,
  getImageType,
  isSeDockerImage,
} from '@postgres.ai/shared/pages/Configuration/utils'
import { formatTuningParams } from '../endpoints/testDbSource'

export interface DatabaseType {
  [name: string]: string | Object
}

export type configTypes = {
  global?: {
    debug?: boolean
  }
  databaseContainer?: {
    dockerImage?: string
    dockerPath?: string
  }
  databaseConfigs?: {
    configs?: {
      [key: string]: string
    }
  }
  retrieval?: {
    refresh?: {
      timetable?: string
    }
    spec?: {
      logicalDump?: {
        options?: {
          customOptions?: string[]
          databases?: DatabaseType | null
          parallelJobs?: string | number
          ignoreErrors?: boolean
          source?: {
            connection?: {
              dbname?: string
              host?: string
              port?: string | number
              username?: string
              password?: string
            }
          }
        }
      }
      logicalRestore?: {
        options?: {
          customOptions?: string[]
          parallelJobs?: string | number
          ignoreErrors?: boolean
        }
      }
    }
  }
}

export const formatConfig = (config: configTypes) => {
  const dockerImage = config.databaseContainer?.dockerImage
  return {
    debug: config.global?.debug,
    dockerImage: isSeDockerImage(dockerImage)
      ? getImageMajorVersion(dockerImage)
      : dockerImage,
    ...(dockerImage && {
      dockerImageType: getImageType(dockerImage),
    }),
    ...(isSeDockerImage(dockerImage) && {
      dockerTag: dockerImage?.split(':')[1],
    }),
    dockerPath: dockerImage,
    tuningParams: formatTuningParams(config.databaseConfigs?.configs),
    sharedBuffers: config.databaseConfigs?.configs?.shared_buffers,
    sharedPreloadLibraries:
      config.databaseConfigs?.configs?.shared_preload_libraries,
    timetable: config.retrieval?.refresh?.timetable,
    dbname:
      config.retrieval?.spec?.logicalDump?.options?.source?.connection?.dbname,
    host: config.retrieval?.spec?.logicalDump?.options?.source?.connection
      ?.host,
    port: config.retrieval?.spec?.logicalDump?.options?.source?.connection
      ?.port,
    username:
      config.retrieval?.spec?.logicalDump?.options?.source?.connection
        ?.username,
    password:
      config.retrieval?.spec?.logicalDump?.options?.source?.connection
        ?.password,
    databases: formatDatabases(
      config.retrieval?.spec?.logicalDump?.options
        ?.databases as DatabaseType | null,
    ),
    dumpParallelJobs:
      config.retrieval?.spec?.logicalDump?.options?.parallelJobs,
    dumpIgnoreErrors:
      config.retrieval?.spec?.logicalDump?.options?.ignoreErrors,
    restoreParallelJobs:
      config.retrieval?.spec?.logicalRestore?.options?.parallelJobs,
    restoreIgnoreErrors:
      config.retrieval?.spec?.logicalRestore?.options?.ignoreErrors,
    pgDumpCustomOptions: formatDumpCustomOptions(
      (config.retrieval?.spec?.logicalDump?.options
        ?.customOptions as string[]) || null,
    ),
    pgRestoreCustomOptions: formatDumpCustomOptions(
      (config.retrieval?.spec?.logicalRestore?.options
        ?.customOptions as string[]) || null,
    ),
  }
}

export type Config = ReturnType<typeof formatConfig>
