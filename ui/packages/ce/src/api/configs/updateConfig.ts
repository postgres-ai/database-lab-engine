import {
  postUniqueCustomOptions,
  postUniqueDatabases,
} from '@postgres.ai/shared/pages/Instance/Configuration/utils'
import { formatTuningParamsToObj } from '@postgres.ai/shared/types/api/endpoints/testDbSource'
import { Config } from '@postgres.ai/shared/types/api/entities/config'
import { request } from 'helpers/request'

// physicalEnvsToObject turns the UI's array-of-{key,value} into the engine's
// envs map. Empty rows (no key) are dropped so the user can leave a blank
// trailing row in the editor without polluting the YAML.
const physicalEnvsToObject = (
  envs: Array<{ key: string; value: string }>,
): { [key: string]: string } => {
  const out: { [key: string]: string } = {}
  for (const { key, value } of envs) {
    if (!key) continue
    out[key] = value
  }
  return out
}

const buildLogicalSpec = (req: Config) => ({
  logicalDump: {
    options: {
      databases: postUniqueDatabases(req.databases),
      customOptions: postUniqueCustomOptions(req.pgDumpCustomOptions),
      ...(req.dumpParallelJobs && { parallelJobs: req.dumpParallelJobs }),
      ignoreErrors: req.dumpIgnoreErrors,
      source: {
        // Expert mode is discrete-fields-only. Clear any source connectionString
        // (e.g. written by `dblab local-install` for a URL carrying ?sslmode=…)
        // so it cannot shadow the host/port/dbname/username edited here — the
        // engine prefers a non-empty connectionString over the discrete fields.
        connectionString: '',
        connection: {
          dbname: req.dbname,
          host: req.host,
          ...(req.port && { port: req.port }),
          username: req.username,
          password: req.password,
        },
      },
    },
  },
  logicalRestore: {
    options: {
      customOptions: postUniqueCustomOptions(req.pgRestoreCustomOptions),
      ...(req.restoreParallelJobs && { parallelJobs: req.restoreParallelJobs }),
      ignoreErrors: req.restoreIgnoreErrors,
      configs: formatTuningParamsToObj(req.restoreConfigs),
    },
  },
})

const buildPhysicalSpec = (req: Config) => ({
  physicalRestore: {
    options: {
      ...(req.physicalTool && { tool: req.physicalTool }),
      ...(req.physicalDockerImage && {
        dockerImage: req.physicalDockerImage,
      }),
      sync: { enabled: req.physicalSyncEnabled },
      ...(req.physicalTool === 'walg' && {
        walg: { backupName: req.physicalWalgBackupName || 'LATEST' },
      }),
      ...(req.physicalTool === 'pgbackrest' && {
        pgbackrest: {
          stanza: req.physicalPgbackrestStanza,
          delta: req.physicalPgbackrestDelta,
        },
      }),
      envs: physicalEnvsToObject(req.physicalEnvs ?? []),
    },
  },
})

export const updateConfig = async (req: Config) => {
  const mode = req.retrievalMode || 'logical'
  const isPhysical = mode === 'physical'

  const response = await request('/admin/config', {
    method: 'POST',
    body: JSON.stringify({
      retrievalMode: mode,
      global: {
        debug: req.debug,
      },
      databaseContainer: {
        dockerImage: req.dockerPath,
      },
      databaseConfigs: {
        configs: {
          shared_buffers: req.sharedBuffers,
          shared_preload_libraries: req.sharedPreloadLibraries,
          ...(req.tuningParams as unknown as { [key: string]: string }),
        },
      },
      retrieval: {
        refresh: {
          timetable: req.timetable,
        },
        spec: isPhysical ? buildPhysicalSpec(req) : buildLogicalSpec(req),
      },
    }),
  })

  return {
    response: response.ok ? response : null,
    error: response.ok ? null : response,
  }
}
