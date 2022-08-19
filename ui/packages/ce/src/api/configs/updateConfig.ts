import { postUniqueDatabases } from '@postgres.ai/shared/pages/Configuration/utils'
import { Config } from '@postgres.ai/shared/types/api/entities/config'
import { request } from 'helpers/request'

export const updateConfig = async (req: Config) => {
  const response = await request('/admin/config', {
    method: 'POST',
    body: JSON.stringify({
      global: {
        debug: req.debug,
      },
      databaseContainer: {
        dockerImage: req.dockerImage,
      },
      databaseConfigs: {
        configs: {
          shared_buffers: req.sharedBuffers,
          shared_preload_libraries: req.sharedPreloadLibraries,
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
              parallelJobs: req.pg_dump,
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
        },
      },
    }),
  })

  return {
    response: response.ok ? response : null,
    error: response.ok ? null : response,
  }
}
