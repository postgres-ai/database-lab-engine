/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import 'es6-promise/auto';
import 'whatwg-fetch';

function encodeData(data) {
  return Object.keys(data).map(function (key) {
    return [key, data[key]].map(encodeURIComponent).join('=');
  }).join('&');
}

class Api {
  constructor(setting) {
    this.server = setting.server;
    this.apiServer = setting.apiServer
  }

  get(url, query, options) {
    let params = '';

    if (query) {
      params = `?${encodeData(query)}`;
    }

    if (options) {
      options.Prefer = 'count=none';
    }

    let fetchOptions = {
      ...options,
      method: 'get',
      credentials: 'include'
    };

    return fetch(`${url}${params}`, fetchOptions);
  }

  post(url, data, options = {}) {
    let headers = options.headers || {};

    let fetchOptions = {
      ...options,
      method: 'post',
      credentials: 'include',
      headers: {
        ...headers,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    };

    return fetch(url, fetchOptions);
  }

  patch(url, data, options = {}) {
    let headers = options.headers || {};

    let fetchOptions = {
      ...options,
      method: 'PATCH',
      credentials: 'include',
      headers: {
        ...headers,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    };

    return fetch(url, fetchOptions);
  }

  delete(url, data, options = {}) {
    let headers = options.headers || {};

    let fetchOptions = {
      ...options,
      method: 'DELETE',
      credentials: 'include',
      headers: {
        ...headers,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(data)
    };

    return fetch(url, fetchOptions);
  }

  login(login, password) {
    let headers = {};

    return this.post(`${this.apiServer}/rpc/login`, {
      email: login,
      password: password
    }, {
      headers: headers
    });
  }

  getUserProfile(token) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.get(`${this.apiServer}/user_get`, {}, {
      headers: headers
    });
  }

  getAccessTokens(token, orgId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId !== null && orgId !== 0) {
      params.org_id = `eq.${orgId}`;
    }

    return this.get(`${this.apiServer}/api_tokens`, params, {
      headers: headers
    });
  }

  getAccessToken(token, name, expires, orgId, isPersonal) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/api_token_create`, {
      name: name,
      expires: expires,
      org_id: orgId,
      is_personal: isPersonal
    }, {
      headers: headers
    });
  }

  revokeAccessToken(token, id) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/api_token_revoke`, { id: id }, {
      headers: headers
    });
  }

  getCheckupReports(token, orgId, projectId, reportId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId !== null && orgId !== 0) {
      params.org_id = `eq.${orgId}`;
    }

    if (projectId !== null && projectId !== 0) {
      params.project_id = `eq.${projectId}`;
    }

    if (typeof reportId !== 'undefined' && reportId !== 0) {
      params.id = `eq.${reportId}`;
    }

    return this.get(`${this.apiServer}/checkup_reports`, params, {
      headers: headers
    });
  }

  getCheckupReportFiles(token, reportId, type, orderBy, orderDirection) {
    let headers = {
      Authorization: 'Bearer ' + token
    };
    let params = {
      checkup_report_id: `eq.${reportId}`
    };

    if (type) {
      params.type = `eq.${type}`;
    }

    if (orderBy && orderDirection) {
      params.order = `${orderBy}.${orderDirection}`;
    }

    return this.get(`${this.apiServer}/checkup_report_files`, params, {
      headers: headers
    });
  }

  getCheckupReportFile(token, projectId, reportId, fileId, type) {
    let headers = {
      Authorization: 'Bearer ' + token
    };
    let params = {
      project_id: `eq.${projectId}`,
      checkup_report_id: `eq.${reportId}`,
      type: `eq.${type}`
    };

    if (fileId === parseInt(fileId, 10)) {
      params.id = `eq.${fileId}`;
    } else {
      params.filename = `eq.${fileId}`;
    }

    return this.get(`${this.apiServer}/checkup_report_file_data`, params, {
      headers: headers
    });
  }

  getProjects(token, orgId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId) {
      params.org_id = `eq.${orgId}`;
    }

    return this.get(`${this.apiServer}/projects`, params, {
      headers: headers
    });
  }

  getJoeSessions(token, orgId, projectId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId !== null && orgId !== 0) {
      params.org_id = `eq.${orgId}`;
    }

    if (projectId !== null && projectId !== 0) {
      params.project_id = `eq.${projectId}`;
    }

    return this.get(`${this.apiServer}/joe_sessions`, params, {
      headers: headers
    });
  }

  getJoeSessionCommands(token,
    { orgId, session, fingerprint, command, project,
      author, startAt, limit, lastId, search, isFavorite }) {
    const params = {};
    const headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId && orgId !== 0) {
      params['org_id'] = `eq.${orgId}`;
    }

    if (session && session !== 0) {
      params['joe_session_id'] = `eq.${session}`;
    }

    if (fingerprint) {
      params.fingerprint = `ilike.${fingerprint}*`;
    }

    if (command) {
      params.command = `eq.${command}`;
    }

    if (startAt) {
      params.created_at = `gt.${startAt}`;
    }

    if (limit) {
      params.limit = limit;
    }

    if (lastId) {
      // backend order by id.desc
      params.id = `lt.${lastId}`;
    }

    if (project) {
      // backend order by id.desc
      params.project_name = `ilike.${project}*`;
    }

    if (author) {
      params.or = `(username.ilike.${author}*,` +
        `useremail.ilike.${author}*,` +
        `slack_username.ilike.${author}*)`;
    }

    if (search) {
      let searchText = encodeURIComponent(search);
      params.tsv = `fts(simple).${searchText}`;
    }

    if (isFavorite) {
      params.is_favorite = `gt.0`;
    }


    return this.get(`${this.apiServer}/joe_session_commands`, params, {
      headers: headers
    });
  }

  getJoeSessionCommand(token, orgId, commandId) {
    let params = { org_id: `eq.${orgId}` };
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (!!commandId && commandId !== 0) {
      params.id = `eq.${commandId}`;
    }

    return this.get(`${this.apiServer}/joe_session_commands`, params, {
      headers: headers
    });
  }

  getOrgs(token, orgId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId) {
      params.id = `eq.${orgId}`;
    }

    return this.get(`${this.apiServer}/orgs`, params, {
      headers: headers
    });
  }

  getOrgUsers(token, orgId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId) {
      params.id = `eq.${orgId}`;
    }

    return this.get(`${this.apiServer}/org_users`, params, {
      headers: headers
    });
  }

  updateOrg(token, orgId, orgData) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token,
      prefer: 'return=representation'
    };

    if (orgData.name) {
      params.name = orgData.name;
    }

    if (orgData.alias) {
      params.alias = orgData.alias;
    }

    if (typeof orgData.users_autojoin !== 'undefined') {
      params.users_autojoin = orgData.users_autojoin;
    }

    if (typeof orgData.onboarding_text !== 'undefined') {
      params.onboarding_text = orgData.onboarding_text;
    }

    if (typeof orgData.oauth_allow_google !== 'undefined') {
      params.oauth_allow_google = orgData.oauth_allow_google;
    }

    if (typeof orgData.oauth_allow_linkedin !== 'undefined') {
      params.oauth_allow_linkedin = orgData.oauth_allow_linkedin;
    }

    if (typeof orgData.oauth_allow_github !== 'undefined') {
      params.oauth_allow_github = orgData.oauth_allow_github;
    }

    if (typeof orgData.oauth_allow_gitlab !== 'undefined') {
      params.oauth_allow_gitlab = orgData.oauth_allow_gitlab;
    }

    return this.patch(`${this.apiServer}/orgs?id=eq.` + orgId, params, {
      headers: headers
    });
  }

  createOrg(token, orgData) {
    let params = {
      name: orgData.name,
      alias: orgData.alias
    };
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgData.email_domain_autojoin) {
      params.org_domain = orgData.email_domain_autojoin;
    }

    if (typeof orgData.users_autojoin !== 'undefined') {
      params.users_autojoin = orgData.users_autojoin;
    }

    return this.post(`${this.apiServer}/rpc/user_create_org`, params, {
      headers: headers
    });
  }

  addOrgDomain(token, orgId, domain) {
    let params = {
      org_id: orgId,
      domain_name: domain
    };
    let headers = {
      Authorization: 'Bearer ' + token,
      prefer: 'return=representation'
    };

    return this.post(`${this.apiServer}/org_domains`, params, {
      headers: headers
    });
  }

  deleteOrgDomain(token, domainId) {
    let headers = {
      Authorization: 'Bearer ' + token,
      prefer: 'return=representation'
    };

    return this.delete(`${this.apiServer}/org_domains?id=eq.${domainId}`, {}, {
      headers: headers
    });
  }

  inviteUser(token, orgId, email) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/user_invite_org_user`, {
      org_id: orgId,
      email: email
    }, {
      headers: headers
    });
  }

  useDemoData(token) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/use_demo_data`, {}, {
      headers: headers
    });
  }

  getDbLabInstances(token, orgId, projectId, instanceId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId !== null && orgId !== 0) {
      params.org_id = `eq.${orgId}`;
    }
    if (projectId !== null && projectId !== 0) {
      params.project_id = `eq.${projectId}`;
    }
    if (typeof instanceId !== 'undefined' && instanceId !== 0) {
      params.id = `eq.${instanceId}`;
    }

    return this.get(`${this.apiServer}/dblab_instances`, params, {
      headers: headers
    });
  }

  addDbLabInstance(token, instanceData) {
    let headers = {
      Authorization: 'Bearer ' + token
    };
    let params = {
      url: instanceData.url,
      org_id: instanceData.orgId,
      token: instanceData.instanceToken,
      project: instanceData.project,
      project_label: instanceData.projectLabel,
      use_tunnel: instanceData.useTunnel,
      ssh_server_url: instanceData.sshServerUrl
    };

    return this.post(`${this.apiServer}/rpc/dblab_instance_create`, params, {
      headers: headers
    });
  }

  editDbLabInstance(token, instanceData) {
    let headers = {
      Authorization: 'Bearer ' + token,
    }
    let params = {
      instance_id: Number(instanceData.instanceId),
      project_name: instanceData.project,
      project_label: instanceData.projectLabel,
      use_tunnel: instanceData.useTunnel,
      ssh_server_url: instanceData.sshServerUrl,
      verify_token: instanceData.instanceToken,
    }

    return this.post(`${this.apiServer}/rpc/dblab_instance_edit`, params, {
      headers: headers,
    })
  }

  destroyDbLabInstance(token, instanceId) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/dblab_instance_destroy`, {
      instance_id: instanceId
    }, {
      headers: headers
    });
  }

  getDbLabInstanceStatus(token, instanceId) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/dblab_instance_status_refresh`, {
      instance_id: instanceId
    }, {
      headers: headers
    });
  }

  checkDbLabInstanceUrl(token, url, verifyToken, useTunnel) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/dblab_instance_status`, {
      url: url,
      verify_token: verifyToken,
      use_tunnel: useTunnel
    }, {
      headers: headers
    });
  }

  getJoeInstances(token, orgId, projectId, instanceId) {
    let params = {};
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (orgId !== null && orgId !== 0) {
      params.org_id = `eq.${orgId}`;
    }
    if (typeof projectId !== 'undefined' && projectId !== 0) {
      params.project_id = `eq.${projectId}`;
    }
    if (typeof instanceId !== 'undefined' && instanceId !== 0) {
      params.id = `eq.${instanceId}`;
    }

    return this.get(`${this.apiServer}/joe_instances`, params, {
      headers: headers
    });
  }

  getJoeInstanceChannels(token, instanceId) {
    let params = {
      instance_id: instanceId
    };
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.get(`${this.apiServer}/rpc/joe_instance_channels_get`, params, {
      headers: headers
    });
  }

  sendJoeInstanceCommand(token, instanceId, channelId, command, sessionId) {
    let params = {
      instance_id: instanceId,
      channel_id: channelId,
      command: command
    };
    let headers = {
      Authorization: 'Bearer ' + token
    };

    if (sessionId !== null && sessionId !== 0) {
      params.session_id = sessionId;
    }

    return this.post(`${this.apiServer}/rpc/joe_command_send`, params, {
      headers: headers
    });
  }

  getJoeInstanceMessages(token, channelId, sessionId) {
    let params = {
      channel_id: `eq.${channelId}`,
      session_id: `eq.${sessionId}`
    };
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.get(`${this.apiServer}/joe_messages`, params, {
      headers: headers
    });
  }

  getJoeMessageArtifacts(token, messageId) {
    let params = {
      message_id: `eq.${messageId}`
    };
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.get(`${this.apiServer}/joe_message_artifacts`, params, {
      headers: headers
    });
  }

  addJoeInstance(token, instanceData) {
    let headers = {
      Authorization: 'Bearer ' + token
    };
    let params = {
      url: instanceData.url,
      org_id: instanceData.orgId,
      token: instanceData.verifyToken,
      project: instanceData.project,
      use_tunnel: instanceData.useTunnel,
      dry_run: instanceData.dryRun
    };

    if (instanceData.useTunnel && instanceData.sshServerUrl) {
      params.ssh_server_url = instanceData.sshServerUrl;
    }

    return this.post(`${this.apiServer}/rpc/joe_instance_create`, params, {
      headers: headers
    });
  }

  destroyJoeInstance(token, instanceId) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/joe_instance_destroy`, {
      instance_id: instanceId
    }, {
      headers: headers
    });
  }

  deleteJoeSessions(token, ids) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/joe_session_delete`, {
      ids: ids
    }, {
      headers: headers
    });
  }

  deleteJoeCommands(token, ids) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/joe_command_delete`, { ids },
      { headers });
  }


  deleteCheckupReports(token, ids) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/checkup_report_delete`, {
      ids: ids
    }, {
      headers: headers
    });
  }

  joeCommandFavorite(token, commandId, favorite) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/joe_command_favorite`, {
      command_id: parseInt(commandId, 10),
      favorite
    }, { headers });
  }

  getSharedUrlData(uuid) {
    return this.get(`${this.apiServer}/rpc/shared_url_get_data`, { uuid }, {});
  }

  getSharedUrl(token, org_id, object_type, object_id) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/shared_url_get`, {
      org_id,
      object_type,
      object_id
    }, { headers });
  }

  addSharedUrl(token, urlParams) {
    let headers = {
      Authorization: 'Bearer ' + token
    };
    let params = {
      org_id: urlParams.orgId,
      url: urlParams.url,
      object_type: urlParams.objectType,
      object_id: urlParams.objectId
    };

    if (urlParams.uuid) {
      params['uuid'] = urlParams.uuid;
    }

    return this.post(`${this.apiServer}/rpc/shared_url_add`, params, { headers });
  }

  removeSharedUrl(token, org_id, id) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(`${this.apiServer}/rpc/shared_url_remove`, { org_id, id }, { headers });
  }

  subscribeBilling(token, org_id, payment_method_id) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(
      `${this.apiServer}/rpc/billing_subscribe`,
      { org_id, payment_method_id },
      { headers }
    );
  }

  getBillingDataUsage(token, orgId) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.get(
      `${this.apiServer}/billing_data_usage`,
      { org_id: `eq.${orgId}` },
      { headers }
    );
  }

  getDbLabSessions(token, { orgId, projectId, instanceId, limit, lastId }) {
    let headers = {
      Authorization: 'Bearer ' + token,
      Prefer: 'count=exact'
    };

    let params = {
      org_id: `eq.${orgId}`
    };

    if (typeof projectId !== 'undefined' && projectId) {
      params.project_id = `eq.$(projectId)`;
    }

    if (typeof instanceId !== 'undefined' && instanceId) {
      params.instance_id = `eq.$(instanceId)`;
    }

    if (lastId) {
      params.id = `lt.${lastId}`;
    }

    if (limit) {
      params.limit = limit;
    }

    return this.get(`${this.apiServer}/dblab_sessions`, params, {
      headers: headers
    });
  }

  getDbLabSession(token, sessionId) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    let params = {
      id: `eq.${sessionId}`
    };

    return this.get(`${this.apiServer}/dblab_sessions`, params, {
      headers: headers
    });
  }

  getDbLabSessionLogs(token, { sessionId, limit, lastId }) {
    let headers = {
      Authorization: 'Bearer ' + token,
      Prefer: 'count=exact'
    };

    let params = {
      dblab_session_id: `eq.${sessionId}`
    };

    if (lastId) {
      params.id = `lt.${lastId}`;
    }

    if (limit) {
      params.limit = limit;
    }

    return this.get(`${this.apiServer}/dblab_session_logs`, params, {
      headers: headers
    });
  }

  getDbLabSessionArtifacts(token, sessionId) {
    let headers = {
      Authorization: 'Bearer ' + token,
      Prefer: 'count=exact'
    };

    let params = {
      dblab_session_id: `eq.${sessionId}`
    };

    return this.get(`${this.apiServer}/dblab_session_artifacts`, params, {
      headers: headers
    });
  }

  getDbLabSessionArtifact(token, sessionId, artifactType) {
    let headers = {
      Authorization: 'Bearer ' + token,
      Prefer: 'count=exact'
    };

    let params = {
      dblab_session_id: `eq.${sessionId}`,
      artifact_type: `eq.${artifactType}`
    };

    return this.get(`${this.apiServer}/dblab_session_artifacts_data`, params, {
      headers: headers
    });
  }

  updateOrgUser(token, org_id, user_id, role_id) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(
      `${this.apiServer}/rpc/user_update_org_user`,
      { org_id, user_id, role_id },
      { headers }
    );
  }

  deleteOrgUser(token, org_id, user_id) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(
      `${this.apiServer}/rpc/user_delete_org_user`,
      { org_id, user_id },
      { headers }
    );
  }

  getAuditLog(token, { orgId, lastId, limit }) {
    let headers = {
      Authorization: 'Bearer ' + token,
      Prefer: 'count=exact'
    };

    let params = {
      org_id: `eq.${orgId}`
    };

    if (lastId) {
      params.id = `lt.${lastId}`;
    }

    if (limit) {
      params.limit = limit;
    }

    return this.get(`${this.apiServer}/audit_log`, params, {
      headers: headers
    });
  }

  sendUserCode(token) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(
      `${this.apiServer}/rpc/user_send_code`,
      {},
      { headers }
    );
  }

  confirmUserEmail(token, verification_code) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(
      `${this.apiServer}/rpc/user_confirm_email`,
      { verification_code },
      { headers }
    );
  }

  confirmTosAgreement(token) {
    let headers = {
      Authorization: 'Bearer ' + token
    };

    return this.post(
      `${this.apiServer}/rpc/user_confirm_tos_agreement`,
      {},
      { headers }
    );
  }
}

export default Api;
