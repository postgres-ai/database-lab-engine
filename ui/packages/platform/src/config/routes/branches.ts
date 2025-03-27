export const ORG_BRANCHES = {
  createPath: (args?: { org: string; instanceId: string }) => {
    const { org = ':org', instanceId = ':instanceId' } = args ?? {}

    return `/${org}/instances/${instanceId}/branches`
  },
  ADD: {
    createPath: (args?: { org: string; instanceId: string }) => {
      const { org = ':org', instanceId = ':instanceId' } = args ?? {}

      return `/${org}/instances/${instanceId}/branches/add`
    },
  },

  BRANCH: {
    createPath: (args?: {
      org: string
      instanceId: string
      branchId: string
    }) => {
      const {
        org = ':org',
        instanceId = ':instanceId',
        branchId = ':branchId',
      } = args ?? {}

      return `/${org}/instances/${instanceId}/branches/${branchId}`
    },
  },
}

export const PROJECT_BRANCHES = {
  createPath: (args?: { org: string; project: string; instanceId: string }) => {
    const {
      org = ':org',
      project = ':project',
      instanceId = ':instanceId',
    } = args ?? {}

    return `/${org}/${project}/instances/${instanceId}/branches`
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

      return `/${org}/${project}/instances/${instanceId}/branches/add`
    },
  },

  BRANCH: {
    createPath: (args?: {
      org: string
      project: string
      instanceId: string
      branchId: string
    }) => {
      const {
        org = ':org',
        project = ':project',
        instanceId = ':instanceId',
        branchId = ':branchId',
      } = args ?? {}

      return `/${org}/${project}/instances/${instanceId}/branches/${branchId}`
    },
  },
}
