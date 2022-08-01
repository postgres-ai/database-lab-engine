export const ROUTES = {
  name: 'Database Lab',
  path: '/',

  AUTH: {
    name: 'Auth',
    path: '/auth',
  },

  INSTANCE: {
    path: `/instance`,
    name: 'Instance',

    CLONES: {
      path: `/instance/clones`,

      CREATE: {
        name: 'Create clone',
        path: `/instance/clones/create`,
      },

      CLONE: {
        name: 'Clone',
        createPath: (cloneId = ':cloneId') => `/instance/clones/${cloneId}`,
      },
    },
  },
}
