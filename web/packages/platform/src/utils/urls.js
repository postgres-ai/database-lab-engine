/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import Aliases from '../utils/aliases';

export default {
  isRequestedPath: function (path) {
    return window.location.pathname.startsWith('/' + path);
  },

  getRequestParam: function (paramName) {
    let params = window.location.search.substring(1).split('&');

    for (let i = 0; i < params.length; i++) {
      let pair = params[i].split('=');
      if (pair[0] === paramName) {
        return pair[1];
      }
    }

    return false;
  },

  getBasePath: function (props, data) {
    let org = props.org ? props.org : null;
    let project = props.project ? props.project : null;

    if (!org && props.orgId && data) {
      org = Aliases.getOrgAliasById(data, props.orgId);
    }

    if (!project && props.projectId && data) {
      project = Aliases.getProjectAliasById(data, props.projectId);
    }

    if (org && project) {
      return '/' + org + '/' + project;
    }

    if (org && !project) {
      return '/' + org;
    }

    return '';
  },

  linkDbLabInstances: function (props) {
    const basePath = this.getBasePath(props);

    return basePath + '/instances';
  },

  linkDbLabInstance: function (props, instanceId) {
    const basePath = this.getBasePath(props);

    return basePath + '/instances/' + instanceId;
  },

  linkDbLabInstanceAdd: function (props) {
    const basePath = this.getBasePath(props);

    return basePath + '/instances/add';
  },

  linkDbLabClone: function (props, instanceId, cloneId) {
    const basePath = this.getBasePath(props);

    return basePath + '/instances/' + instanceId + '/clones/' + encodeURIComponent(cloneId);
  },

  linkDbLabCloneAdd: function (props, instanceId) {
    const basePath = this.getBasePath(props);

    return basePath + '/instances/' + instanceId + '/clones/add';
  },

  linkReport: function (props, reportId, type) {
    const basePath = this.getBasePath(props);
    let url = basePath + '/reports/' + reportId;

    if (type) {
      url = url + '/' + type;
    }

    return url;
  },

  linkReports: function (props) {
    const basePath = this.getBasePath(props);
    return basePath + '/reports';
  },

  linkReportFile: function (props, reportId, fileId, type) {
    const basePath = this.getBasePath(props);

    return basePath + '/reports/' + reportId + '/files/' + fileId + '/' +
      type;
  },

  linkCheckupAgentAdd: function (props) {
    const basePath = this.getBasePath(props);
    return basePath + '/checkup-config';
  },

  linkJoeInstances: function (props) {
    const basePath = this.getBasePath(props);

    return basePath + '/joe-instances';
  },

  linkJoeInstance: function (props, instanceId) {
    const basePath = this.getBasePath(props);

    return basePath + '/joe-instances/' + instanceId;
  },

  linkJoeInstanceAdd: function (props) {
    const basePath = this.getBasePath(props);

    return basePath + '/joe-instances/add';
  },

  linkBilling: function (props) {
    const basePath = this.getBasePath(props);

    return basePath + '/billing';
  },

  linkReportQuery: function (props, reportId, fileName) {
    const basePath = this.getBasePath(props);

    return basePath + '/reports/' + reportId + '/files/' + fileName + '/sql?raw';
  },

  linkReportQueryFull: function (props, reportId, fileName) {
    return process.env.PUBLIC_URL +
      this.linkReportQuery(props, reportId, fileName);
  },

  linkAccessTokens: function (props) {
    let org = props.org ? props.org : null;

    return '/' + org + '/tokens';
  },

  isSharedUrl() {
    return window.location.href.indexOf('/shared/') !== -1;
  },

  linkShared(uuid) {
    return window.location.protocol + '//' + window.location.host +
      process.env.PUBLIC_URL + '/shared/' + uuid;
  }
};
