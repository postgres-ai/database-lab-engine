export const ROUTES = {
  name: 'Database Lab Engine',
  path: '/',

  AUTH: {
    name: 'Auth',
    path: '/auth',
  },

  INSTANCE: {
    path: `/instance`,
    name: 'Instance',

    SNAPSHOTS: {
      path: `/instance/snapshots`,

      SNAPSHOTS: {
        name: 'Snapshots',
        path: `/instance/snapshots`,
      },

      SNAPSHOT: {
        name: 'Snapshot',
        createPath: (snapshotId = ':snapshotId') =>
          `/instance/snapshots/${snapshotId}`,
      },
    },
    BRANCHES: {
      path: `/instance/branches`,

      BRANCHES: {
        name: 'Branches',
        path: `/instance/branches`,
      },

      BRANCH: {
        name: 'Branches',
        createPath: (branchId = ':branchId') =>
          `/instance/branches/${branchId}`,
      },
    },

    CLONES: {
      path: `/instance/clones`,

      CREATE: {
        name: 'Create clone',
        path: `/instance/clones/create`,
      },

      CLONES: {
        name: 'Clones',
        path: `/instance/clones`,
      },

      CLONE: {
        name: 'Clone',
        createPath: (cloneId = ':cloneId') => `/instance/clones/${cloneId}`,
      },
    },
  },
}
