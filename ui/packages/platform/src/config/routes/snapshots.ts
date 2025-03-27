export const ORG_SNAPSHOTS = {
  createPath: (args?: { org: string; instanceId: string }) => {
    const { org = ':org', instanceId = ':instanceId' } = args ?? {}

    return `/${org}/instances/${instanceId}/snapshots`
  },
  ADD: {
    createPath: (args?: { org: string; instanceId: string }) => {
      const { org = ':org', instanceId = ':instanceId' } = args ?? {}

      return `/${org}/instances/${instanceId}/snapshots/add`
    },
  },

  SNAPSHOT: {
    createPath: (args?: {
      org: string
      instanceId: string
      snapshotId: string
    }) => {
      const {
        org = ':org',
        instanceId = ':instanceId',
        snapshotId = ':snapshotId',
      } = args ?? {}

      return `/${org}/instances/${instanceId}/snapshots/${snapshotId}`
    },
  },
}

export const PROJECT_SNAPSHOTS = {
  createPath: (args?: { org: string; project: string; instanceId: string }) => {
    const {
      org = ':org',
      project = ':project',
      instanceId = ':instanceId',
    } = args ?? {}

    return `/${org}/${project}/instances/${instanceId}/snapshots`
  },
  ADD: {
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

      return `/${org}/${project}/instances/${instanceId}/snapshots/add`
    },
  },

  SNAPSHOT: {
    createPath: (args?: {
      org: string
      project: string
      instanceId: string
      snapshotId: string
    }) => {
      const {
        org = ':org',
        project = ':project',
        instanceId = ':instanceId',
        snapshotId = ':snapshotId',
      } = args ?? {}

      return `/${org}/${project}/instances/${instanceId}/snapshots/${snapshotId}`
    },
  },
}
