/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export interface ProjectDataType {
  [data: string]: {
    id?: number
    projects: {
      [project: string]: {
        id: number
        name?: string
      }
    }
  }
}

// Find project alias by project id from userProfile.orgs data.
export const getProjectAliasById = function (
  data: ProjectDataType,
  projectId: number | string,
) {
  if (!data || !projectId) {
    return null
  }

  for (const org in data) {
    if (Object.prototype.hasOwnProperty.call(data, org)) {
      for (const proj in data[org].projects) {
        if (Object.prototype.hasOwnProperty.call(data[org].projects, proj)) {
          if (data[org].projects[proj].id === projectId) {
            return proj
          }
        }
      }
    }
  }

  return null
}

// Find project alias by project id from userProfile.orgs data.
export const getProjectNameById = function (
  data: ProjectDataType,
  projectId: number | string,
) {
  if (!data || !projectId) {
    return null
  }

  for (const org in data) {
    if (Object.prototype.hasOwnProperty.call(data, org)) {
      for (const proj in data[org].projects) {
        if (Object.prototype.hasOwnProperty.call(data[org].projects, proj)) {
          if (data[org].projects[proj].id === projectId) {
            return data[org].projects[proj].name
          }
        }
      }
    }
  }

  return null
}

// Find org alias by org id from userProfile.orgs data.
export const getOrgAliasById = function (
  data?: ProjectDataType,
  orgId?: number,
) {
  if (!data || !orgId) {
    return null
  }

  for (const org in data) {
    if (data[org].id === orgId) {
      return org
    }
  }

  return null
}

// Prepare alias.
export const formatAlias = function (alias: string) {
  let newAlias = alias.replace(/[ :/?#[\]@!$&()*+,;=.~"'']/g, '-')
  newAlias = newAlias.toLowerCase()
  return newAlias
}

export const getBasePath = function (
  props: {
    org: string
    project: string
    orgId: number
    projectId: number | string
  },
  data: ProjectDataType,
) {
  let org = props.org ? props.org : null
  let project = props.project ? props.project : null

  if (!org && props.orgId && data) {
    org = getOrgAliasById(data, props.orgId)
  }

  if (!project && props.projectId && data) {
    project = getProjectAliasById(data, props.projectId)
  }

  if (org && project) {
    return '/' + org + '/' + project
  }

  if (org && !project) {
    return '/' + org
  }

  return ''
}
