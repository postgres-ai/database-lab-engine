/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

module.exports = {
  // Find project alias by project id from userProfile.orgs data.
  getProjectAliasById: function (data, projectId) {
    if (!data || !projectId) {
      return null;
    }

    for (let org in data) {
      if (data.hasOwnProperty(org)) {
        for (let proj in data[org].projects) {
          if (data[org].projects.hasOwnProperty(proj)) {
            if (data[org].projects[proj].id === projectId) {
              return proj;
            }
          }
        }
      }
    }

    return null;
  },

  // Find project alias by project id from userProfile.orgs data.
  getProjectNameById: function (data, projectId) {
    if (!data || !projectId) {
      return null;
    }

    for (let org in data) {
      if (data.hasOwnProperty(org)) {
        for (let proj in data[org].projects) {
          if (data[org].projects.hasOwnProperty(proj)) {
            if (data[org].projects[proj].id === projectId) {
              return data[org].projects[proj].name;
            }
          }
        }
      }
    }

    return null;
  },

  // Find org alias by org id from userProfile.orgs data.
  getOrgAliasById: function (data, orgId) {
    if (!data || !orgId) {
      return null;
    }

    for (let org in data) {
      if (data[org].id === orgId) {
        return org;
      }
    }

    return null;
  },

  // Prepare alias.
  formatAlias: function (alias) {
    let newAlias = alias.replace(/[ :\/\?#\[\]@!$&()*+,;=.~"'']/g, '-');
    newAlias = newAlias.toLowerCase();
    return newAlias;
  },

  getBasePath: function (props, data) {
    let org = props.org ? props.org : null;
    let project = props.project ? props.project : null;

    if (!org && props.orgId && data) {
      org = this.getOrgAliasById(data, props.orgId);
    }

    if (!project && props.projectId && data) {
      project = this.getProjectAliasById(data, props.projectId);
    }

    if (org && project) {
      return '/' + org + '/' + project;
    }

    if (org && !project) {
      return '/' + org;
    }

    return '';
  }
};
