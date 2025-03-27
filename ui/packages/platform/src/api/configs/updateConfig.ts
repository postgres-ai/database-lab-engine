import {
  postUniqueCustomOptions,
  postUniqueDatabases,
} from '@postgres.ai/shared/pages/Instance/Configuration/utils'
import { Config } from '@postgres.ai/shared/types/api/entities/config'
import { request } from 'helpers/request'

export const updateConfig = async (req: Config, instanceId: string) => {
  const response = await request('/rpc/dblab_api_call', {
    method: 'POST',
    body: JSON.stringify({
      instance_id: instanceId,
      action: '/admin/config',
      method: 'post',
      data: {
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
          spec: {
            logicalDump: {
              options: {
                databases: postUniqueDatabases(req.databases),
                customOptions: postUniqueCustomOptions(req.pgDumpCustomOptions),
                parallelJobs: req.dumpParallelJobs,
                ignoreErrors: req.dumpIgnoreErrors,
                source: {
                  connection: {
                    dbname: req.dbname,
                    host: req.host,
                    port: req.port,
                    username: req.username,
                    password: req.password,
                  },
                },
              },
            },
            logicalRestore: {
              options: {
                customOptions: postUniqueCustomOptions(
                  req.pgRestoreCustomOptions,
                ),
                parallelJobs: req.restoreParallelJobs,
                ignoreErrors: req.restoreIgnoreErrors,
              },
            },
          },
        },
      },
    }),
  })

  return {
    response: response.ok ? response : null,
    error: response.ok ? null : response,
  }
}
