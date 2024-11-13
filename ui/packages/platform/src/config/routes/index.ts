/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { ORG_INSTANCES, PROJECT_INSTANCES } from './instances'

export const ROUTES = {
  ROOT: {
    name: 'Organizations',
    path: '/',
  },

  PROFILE: {
    path: '/profile',
  },

  CREATE_ORG: {
    path: '/addorg',
  },

  ORG: {
    TOKENS: {
      createPath: (args?: { org: string }) => {
        const { org = ':org' } = args ?? {}
        return `/${org}/tokens`
      }
    },

    JOE_INSTANCES: {
      JOE_INSTANCE: {
        createPath: ({
          org = ':org',
          id = ':id',
        }: { org?: string; id?: string } = {}) => `/${org}/joe-instances/${id}`,
      },
    },

    INSTANCES: ORG_INSTANCES,

    PROJECT: {
      JOE_INSTANCES: {
        JOE_INSTANCE: {
          createPath: ({
            org = ':org',
            project = ':project',
            id = ':id',
          }: { org?: string; project?: string; id?: string } = {}) =>
            `/${org}/${project}/joe-instances/${id}`,
        },
      },

      INSTANCES: PROJECT_INSTANCES,

      ASSISTANT: {
        createPath: ({
            org = ':org',
            id,
          }: { org?: string; id?: string } = {}) =>
            id ? `/${org}/assistant/${id}` : `/${org}/assistant`,
      }
    },
  },
}
