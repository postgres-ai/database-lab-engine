import { ORG_CLONES, PROJECT_CLONES } from './clones'
import { ORG_BRANCHES, PROJECT_BRANCHES } from './branches'
import { ORG_SNAPSHOTS, PROJECT_SNAPSHOTS } from './snapshots'

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
    LOGS: {
      createPath: (args?: { org: string; instanceId: string }) => {
        const { org = ':org', instanceId = ':instanceId' } = args ?? {}
        return `/${org}/instances/${instanceId}/logs`
      },
    },
    CONFIGURATION: {
      createPath: (args?: { org: string; instanceId: string }) => {
        const { org = ':org', instanceId = ':instanceId' } = args ?? {}
        return `/${org}/instances/${instanceId}/configuration`
      },
    },
    CLONES: ORG_CLONES,
    BRANCHES: ORG_BRANCHES,
    SNAPSHOTS: ORG_SNAPSHOTS,
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
    BRANCHES: PROJECT_BRANCHES,
    SNAPSHOTS: PROJECT_SNAPSHOTS,
    LOGS: {
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
        return `/${org}/${project}/instances/${instanceId}/logs`
      },
    },
    CONFIGURATION: {
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
        return `/${org}/${project}/instances/${instanceId}/configuration`
      },
    },
  },
}
