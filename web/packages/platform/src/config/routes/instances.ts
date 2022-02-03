import { ORG_CLONES, PROJECT_CLONES } from './clones'

export const ORG_INSTANCES = {
  createPath: (args?: { org: string }) => {
    const { org = ':org' } = args ?? {}
    return `/${org}/instances`
  },

  INSTANCE: {
    createPath: (args?: { org: string; instanceId: string }) => {
      const { org = ':org', instanceId = ':instanceId' } = args ?? {}
      return `/${org}/instances/${instanceId}`
    },

    CLONES: ORG_CLONES,
  },
}

export const PROJECT_INSTANCES = {
  createPath: (args?: { org: string; project: string }) => {
    const { org = ':org', project = ':project' } = args ?? {}
    return `/${org}/${project}/instances`
  },

  INSTANCE: {
    createPath: (args?: {
      org: string
      project: string
      instanceId: string
    }) => {
      const {
        org = ':org',
        project = ':project',
        instanceId = ':instanceId',
      } = args ?? {}
      return `/${org}/${project}/instances/${instanceId}`
    },

    CLONES: PROJECT_CLONES,
  },
}
