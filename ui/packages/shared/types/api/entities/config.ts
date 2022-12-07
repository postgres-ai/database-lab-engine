import { getImageType } from "@postgres.ai/shared/pages/Configuration/utils"

export interface DatabaseType {
  [name: string]: string | Object
}

export type configTypes = {
  global?: {
    debug?: boolean
  }
  databaseContainer?: {
    dockerImage?: string
  }
  databaseConfigs?: {
    configs?: {
      shared_buffers?: string
      shared_preload_libraries?: string
    }
  }
  retrieval?: {
    refresh?: {
      timetable?: string
    }
    spec?: {
      logicalDump?: {
        options?: {
          databases?: DatabaseType | null
          parallelJobs?: string | number
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
          parallelJobs?: string | number
        }
      }
    }
  }
}

const formatDatabases = (databases: DatabaseType | null) => {
  let formattedDatabases = ''

  if (databases !== null) {
    Object.keys(databases).forEach(function (key) {
      formattedDatabases += key + ','
    })
  }

  return formattedDatabases
}

export const formatConfig = (config: configTypes) => {
  return {
    debug: config.global?.debug,
    dockerImage: config.databaseContainer?.dockerImage,
    ...config.databaseContainer?.dockerImage && {dockerImageType: getImageType(config.databaseContainer?.dockerImage)},
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
    pg_dump: config.retrieval?.spec?.logicalDump?.options?.parallelJobs,
    pg_restore: config.retrieval?.spec?.logicalRestore?.options?.parallelJobs,
  }
}

export type Config = ReturnType<typeof formatConfig>
