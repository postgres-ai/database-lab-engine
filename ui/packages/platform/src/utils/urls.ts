/* eslint-disable import/no-anonymous-default-export */
/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  ProjectDataType,
  getOrgAliasById,
  getProjectAliasById,
} from './aliases'

export interface PropsType {
  org?: string | number | null
  orgId?: number | null
  project?: string | null
  projectId?: string | number | null
}

export default {
  isRequestedPath: function (path: string) {
    return window.location.pathname.startsWith('/' + path)
  },

  getRequestParam: function (paramName: string) {
    const params = window.location.search.substring(1).split('&')

    for (let i = 0; i < params.length; i++) {
      const pair = params[i].split('=')
      if (pair[0] === paramName) {
        return pair[1]
      }
    }

    return false
  },

  getBasePath: function (props: PropsType, data: ProjectDataType) {
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
  },

  linkDbLabInstances: function (props: PropsType) {
    const basePath = this.getBasePath(props)

    return basePath + '/instances'
  },

  linkClusters: function (props: PropsType) {
    const basePath = this.getBasePath(props)

    return basePath + '/pg'
  },

  linkDbLabInstance: function (props: PropsType, instanceId: string) {
    const basePath = this.getBasePath(props)

    return basePath + '/instances/' + instanceId
  },

  linkDbLabInstanceAdd: function (props: PropsType, creationType?: string) {
    const basePath = this.getBasePath(props)

    return (
      basePath + '/instances' + (creationType ? '/' + creationType : '')
    )
  },

  linkClusterInstanceAdd: function (props: PropsType, creationType?: string) {
    const basePath = this.getBasePath(props)

    return (
      basePath + '/pg' + (creationType ? '/' + creationType : '')
    )
  },

  linkDbLabClone: function (
    props: PropsType,
    instanceId: string,
    cloneId: string | number | boolean,
  ) {
    const basePath = this.getBasePath(props)

    return (
      basePath +
      '/instances/' +
      instanceId +
      '/clones/' +
      encodeURIComponent(cloneId)
    )
  },

  linkDbLabCloneAdd: function (props: PropsType, instanceId: string) {
    const basePath = this.getBasePath(props)

    return basePath + '/instances/' + instanceId + '/clones/add'
  },

  linkDbLabInstanceEditProject: function (props: PropsType, instanceId: string) {
    const basePath = this.getBasePath(props)

    return `${basePath}/instances/edit/${instanceId}`
  },


  linkReport: function (props: PropsType, reportId: string, type?: string) {
    const basePath = this.getBasePath(props)
    let url = basePath + '/reports/' + reportId

    if (type) {
      url = url + '/' + type
    }

    return url
  },

  linkReports: function (props: PropsType) {
    const basePath = this.getBasePath(props)
    return basePath + '/reports'
  },

  linkReportFile: function (
    props: PropsType,
    reportId: string,
    fileId: number,
    type: string,
  ) {
    const basePath = this.getBasePath(props)

    return basePath + '/reports/' + reportId + '/files/' + fileId + '/' + type
  },

  linkCheckupAgentAdd: function (props: PropsType) {
    const basePath = this.getBasePath(props)
    return basePath + '/checkup-config'
  },

  linkJoeInstances: function (props: PropsType) {
    const basePath = this.getBasePath(props)

    return basePath + '/joe-instances'
  },

  linkJoeInstance: function (props: PropsType, instanceId: string | number) {
    const basePath = this.getBasePath(props)

    return basePath + '/joe-instances/' + instanceId
  },

  linkJoeInstanceAdd: function (props: PropsType) {
    const basePath = this.getBasePath(props)

    return basePath + '/joe-instances/add'
  },

  linkBilling: function (props: PropsType) {
    const basePath = this.getBasePath(props)

    return basePath + '/billing'
  },

  linkReportQuery: function (
    props: PropsType,
    reportId: string,
    fileName: string,
  ) {
    const basePath = this.getBasePath(props)

    return basePath + '/reports/' + reportId + '/files/' + fileName + '/sql?raw'
  },

  linkReportQueryFull: function (
    props: PropsType,
    reportId: string,
    fileName: string,
  ) {
    return (
      process.env.PUBLIC_URL + this.linkReportQuery(props, reportId, fileName)
    )
  },

  linkAccessTokens: function (props: { org: string }) {
    const org = props.org ? props.org : null

    return '/' + org + '/tokens'
  },

  isSharedUrl() {
    return window.location.href.indexOf('/shared/') !== -1
  },

  linkShared(uuid: string | null) {
    return (
      window.location.protocol +
      '//' +
      window.location.host +
      process.env.PUBLIC_URL +
      '/shared/' +
      uuid
    )
  },
}
