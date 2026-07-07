import {
  formatDatabases,
  formatDumpCustomOptions,
  getImageMajorVersion,
  getImageType,
  isSeDockerImage,
} from '@postgres.ai/shared/pages/Instance/Configuration/utils'
import { formatTuningParams } from '../endpoints/testDbSource'

export interface DatabaseType {
  [name: string]: string | Object
}

export type RetrievalMode = 'logical' | 'physical' | 'unknown' | ''

export type configTypes = {
  global?: {
    debug?: boolean
  }
  retrievalMode?: RetrievalMode
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
          configs?: { [key: string]: string }
        }
      }
      physicalRestore?: {
        options?: {
          tool?: string
          dockerImage?: string
          sync?: {
            enabled?: boolean
          }
          walg?: {
            backupName?: string
          }
          pgbackrest?: {
            stanza?: string
            delta?: boolean
          }
          envs?: { [key: string]: string | number | boolean }
        }
      }
    }
  }
}

const formatPhysicalEnvs = (
  envs?: { [key: string]: string | number | boolean } | null,
): Array<{ key: string; value: string }> => {
  if (!envs) return []
  return Object.entries(envs).map(([key, value]) => ({
    key,
    value: typeof value === 'string' ? value : String(value),
  }))
}

const inferRetrievalMode = (config: configTypes): RetrievalMode => {
  if (config.retrievalMode) return config.retrievalMode
  if (config.retrieval?.spec?.physicalRestore) return 'physical'
  if (config.retrieval?.spec?.logicalDump) return 'logical'
  return ''
}

export const formatConfig = (config: configTypes) => {
  const dockerImage = config.databaseContainer?.dockerImage
  const physical = config.retrieval?.spec?.physicalRestore?.options
  return {
    debug: config.global?.debug,
    retrievalMode: inferRetrievalMode(config),
    physicalTool: physical?.tool ?? '',
    physicalDockerImage: physical?.dockerImage ?? '',
    physicalSyncEnabled: physical?.sync?.enabled ?? false,
    physicalWalgBackupName: physical?.walg?.backupName ?? '',
    physicalPgbackrestStanza: physical?.pgbackrest?.stanza ?? '',
    physicalPgbackrestDelta: physical?.pgbackrest?.delta ?? false,
    physicalEnvs: formatPhysicalEnvs(physical?.envs),
    dockerImage: isSeDockerImage(dockerImage)
      ? getImageMajorVersion(dockerImage)
      : dockerImage && getImageType(dockerImage) === 'Generic Postgres'
      ? getImageMajorVersion(dockerImage) || dockerImage
      : dockerImage,
    ...(dockerImage && {
      dockerImageType: getImageType(dockerImage),
    }),
    // Extract dockerTag for both SE images and Generic Postgres images
    ...(dockerImage && dockerImage.includes(':') && {
      dockerTag: dockerImage.split(':')[1],
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
      (config.retrieval?.spec?.logicalDump?.options
        ?.databases as DatabaseType | undefined) ?? null,
    ),
    dumpParallelJobs:
      config.retrieval?.spec?.logicalDump?.options?.parallelJobs,
    dumpIgnoreErrors:
      config.retrieval?.spec?.logicalDump?.options?.ignoreErrors,
    restoreParallelJobs:
      config.retrieval?.spec?.logicalRestore?.options?.parallelJobs,
    restoreIgnoreErrors:
      config.retrieval?.spec?.logicalRestore?.options?.ignoreErrors,
    restoreConfigs: (() => {
      const configs =
        config.retrieval?.spec?.logicalRestore?.options?.configs
      if (!configs || Object.keys(configs).length === 0) return ''
      return Object.entries(configs)
        .map(([k, v]) => `${k}=${v}`)
        .join('\n')
    })(),
    pgDumpCustomOptions: formatDumpCustomOptions(
      (config.retrieval?.spec?.logicalDump?.options
        ?.customOptions as string[] | undefined) ?? null,
    ),
    pgRestoreCustomOptions: formatDumpCustomOptions(
      (config.retrieval?.spec?.logicalRestore?.options
        ?.customOptions as string[] | undefined) ?? null,
    ),
  }
}

export type Config = ReturnType<typeof formatConfig>
