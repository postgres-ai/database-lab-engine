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

// WebhookHook represents a single webhook configuration
export interface WebhookHook {
  url: string
  secret?: string
  trigger: string[]
}

// ConfigWarning represents a warning returned when updating config
export interface ConfigWarning {
  setting: string
  message: string
  type: 'restart' | 'security' | 'info'
}

// ConfigUpdateResponse is the response from config update API
export interface ConfigUpdateResponse {
  config: configTypes
  warnings?: ConfigWarning[]
  requiresRestart: boolean
  changedSettings?: string[]
  restartSettings?: string[]
}

export type configTypes = {
  // Global settings
  global?: {
    debug?: boolean
    database?: {
      username?: string
      dbname?: string
    }
  }
  // Server settings
  server?: {
    host?: string
    port?: number
  }
  // Provision settings
  provision?: {
    portPool?: {
      from?: number
      to?: number
    }
    useSudo?: boolean
    keepUserPasswords?: boolean
    cloneAccessAddresses?: string
    containerConfig?: { [key: string]: string }
  }
  // Database container settings
  databaseContainer?: {
    dockerImage?: string
    dockerPath?: string
  }
  databaseConfigs?: {
    configs?: {
      [key: string]: string
    }
  }
  // Cloning settings
  cloning?: {
    maxIdleMinutes?: number
    accessHost?: string
  }
  // Retrieval settings
  retrieval?: {
    refresh?: {
      timetable?: string
      skipStartRefresh?: boolean
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
  // Pool Manager settings
  poolManager?: {
    mountDir?: string
    selectedPool?: string
  }
  // Observer settings
  observer?: {
    replacementRules?: { [key: string]: string }
  }
  // Diagnostic settings
  diagnostic?: {
    logsRetentionDays?: number
  }
  // Embedded UI settings
  embeddedUI?: {
    enabled?: boolean
    dockerImage?: string
    host?: string
    port?: number
  }
  // Platform settings
  platform?: {
    url?: string
    orgKey?: string
    projectName?: string
    enablePersonalTokens?: boolean
    enableTelemetry?: boolean
  }
  // Webhooks settings
  webhooks?: {
    hooks?: WebhookHook[]
  }
}

export const formatConfig = (config: configTypes) => {
  const dockerImage = config.databaseContainer?.dockerImage
  return {
    // Global settings
    debug: config.global?.debug,
    globalDbUsername: config.global?.database?.username,
    globalDbName: config.global?.database?.dbname,

    // Server settings
    serverHost: config.server?.host,
    serverPort: config.server?.port,

    // Provision settings
    portPoolFrom: config.provision?.portPool?.from,
    portPoolTo: config.provision?.portPool?.to,
    useSudo: config.provision?.useSudo,
    keepUserPasswords: config.provision?.keepUserPasswords,
    cloneAccessAddresses: config.provision?.cloneAccessAddresses,
    containerConfig: config.provision?.containerConfig,

    // Docker image settings
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

    // Cloning settings
    maxIdleMinutes: config.cloning?.maxIdleMinutes,
    accessHost: config.cloning?.accessHost,

    // Retrieval settings
    timetable: config.retrieval?.refresh?.timetable,
    skipStartRefresh: config.retrieval?.refresh?.skipStartRefresh,
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

    // Pool Manager settings
    poolMountDir: config.poolManager?.mountDir,
    poolSelectedPool: config.poolManager?.selectedPool,

    // Observer settings
    observerReplacementRules: config.observer?.replacementRules,

    // Diagnostic settings
    logsRetentionDays: config.diagnostic?.logsRetentionDays,

    // Embedded UI settings
    embeddedUIEnabled: config.embeddedUI?.enabled,
    embeddedUIDockerImage: config.embeddedUI?.dockerImage,
    embeddedUIHost: config.embeddedUI?.host,
    embeddedUIPort: config.embeddedUI?.port,

    // Platform settings
    platformUrl: config.platform?.url,
    platformProjectName: config.platform?.projectName,
    platformEnablePersonalToken: config.platform?.enablePersonalTokens,
    platformEnableTelemetry: config.platform?.enableTelemetry,

    // Webhooks settings
    webhooksHooks: config.webhooks?.hooks,
  }
}

export type Config = ReturnType<typeof formatConfig>
