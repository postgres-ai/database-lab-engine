export const ORG_CLONES = {
  ADD: {
    createPath: (args?: {
      org: string
      instanceId: string
    }) => {
      const {
        org = ':org',
        instanceId = ':instanceId',
      } = args ?? {}

      return `/${org}/instances/${instanceId}/clones/add`
    },
  },

  CLONE: {
    createPath: (args?: {
      org: string
      instanceId: string
      cloneId: string
    }) => {
      const {
        org = ':org',
        instanceId = ':instanceId',
        cloneId = ':cloneId',
      } = args ?? {}

      return `/${org}/instances/${instanceId}/clones/${cloneId}`
    },
  },
}

export const PROJECT_CLONES = {
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

      return `/${org}/${project}/instances/${instanceId}/clones/add`
    },
  },

  CLONE: {
    createPath: (args?: {
      org: string
      project: string
      instanceId: string
      cloneId: string
    }) => {
      const {
        org = ':org',
        project = ':project',
        instanceId = ':instanceId',
        cloneId = ':cloneId',
      } = args ?? {}

      return `/${org}/${project}/instances/${instanceId}/clones/${cloneId}`
    },
  },
}
