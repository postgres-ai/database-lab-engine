/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import Reflux from 'reflux';
import jwtDecode from 'jwt-decode';
import md5 from 'md5';
import { v4 as uuidv4 } from 'uuid';

import { emoji } from 'config/emoji';
import { localStorage } from 'helpers/localStorage';

import settings from '../utils/settings';
import Actions from '../actions/actions';
import format from '../utils/format';

import { preformatJoeMessage } from './preformatJoeMessage';
import { createWebSocket } from 'utils/webSockets';

const WS_MAX_RETRY_CONNECTION_COUNT = 5;

const joeInstance = {
  channels: {},
  currentChannelId: null,
  channelsError: null,
  isChannelsProcessing: false,
  isChannelsProcessed: false,

  data: [],
  isProcessing: false,
  isProcessed: false,
  error: null,
  errorMessage: null,

  sessionId: null,
  commandError: null,
  isCommandProcessing: false,
  isCommandProcessed: false,

  messages: {},
  messageError: null,
  isMessagesProcessing: false,
  isMessagesProcessed: false
};

const storeItem = {
  isProcessing: false,
  isProcessed: false,
  data: null,
  error: null
};

const initialState = {
  app: {
    isProcessing: false,
    data: null,
    error: null,
    needUpdate: false
  },
  auth: {
    isProcessing: false,
    userId: null,
    token: null,
    error: null
  },
  dashboard: {
    profileUpdateInitAfterDemo: false
  },
  userProfile: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  inviteUser: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  useDemoData: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  orgProfile: {
    isProcessing: false,
    isProcessed: false,
    orgDomains: {},
    data: null,
    error: null,
    isUpdating: false
  },
  orgUsers: storeItem,
  userTokens: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  tokenRequest: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  reports: {
    projectId: 0,
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  report: {
    reportId: null,
    isProcessing: false,
    isProcessed: false,
    isDownloading: false,
    isDeleting: false,
    isDeleted: false,
    data: null,
    error: null
  },
  reportFile: {
    files: {},
    fileId: null,
    isProcessing: false,
    error: null
  },
  sessions: {
    projectId: 0,
    isProcessing: false,
    isProcessed: false,
    isDeleting: false,
    isDeleted: false,
    data: null,
    error: null
  },
  commands: {
    projectId: 0,
    isProcessing: false,
    isProcessed: false,
    isHistoryExists: [],
    data: null,
    error: null
  },
  command: {
    projectId: 0,
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  projects: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  dbLabInstances: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  newDbLabInstance: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  dbLabInstanceSnapshots: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  dbLabInstanceStatus: {
    instances: {},
    instanceId: null,
    isProcessing: false,
    error: null
  },
  dbLabCloneStatus: {
    isProcessing: false,
    clones: {},
    cloneId: null,
    instanceId: null,
    error: null
  },
  notification: {
    message: null,
    type: null,
    duration: null
  },
  joeInstances: {
    isProcessing: false,
    isProcessed: false,
    data: null,
    error: null
  },
  joeInstance: {
    instances: {}
  },
  externalVisualization: {
    isProcessing: false,
    isProcessed: false,
    error: null,
    url: null,
    plan: null,
    type: null
  },
  newJoeInstance: storeItem,
  sharedUrlData: storeItem,
  shareUrl: storeItem,
  sharedUrls: {
    command: {}
  },
  billing: {
    ...storeItem,
    isSubscriptionProcessing: false,
    isSubscriptionProcessed: false,
    paymentMethodId: ''
  },
  dbLabInstance: storeItem,
  dbLabSessions: {
    ...storeItem,
    isSessionExists: []
  },
  dbLabSession: {
    ...storeItem,
    isLogDownloading: false,
    logs: {}
  },
  auditLog: storeItem
};

const Store = Reflux.createStore({
  listenables: [Actions],

  data: Object.assign({}, initialState),

  getInitialState: function () {
    return Object.assign({}, initialState);
  },

  getError: function (data) {
    let result = '';

    if (data.message === 'JWT expired' ||
      data.details === 'Deprecated JWT value') {
      Actions.signOut();
    }

    if (data.code || data.details || data.hint) {
      if (data.details) {
        result = data.details;
      }

      if (data.hint) {
        result = result + ' ' + data.hint;
      }

      if (result !== '') {
        return result;
      }

      if (data.message) {
        return data.message;
      }

      return true;
    }

    return null;
  },


  onShowNotification(message, type, duration = 3000) {
    this.data.notification.message = message;
    this.data.notification.type = type;
    this.data.notification.duration = duration;
    this.trigger(this.data);
  },

  onHideNotification() {
    this.data.notification.message = null;
    this.data.notification.type = null;
    this.data.notification.duration = null;
    this.trigger(this.data);
  },

  onRefresh() {
    this.data.inviteUser = {
      isProcessing: false,
      isProcessed: false,
      data: null,
      error: null
    };

    this.trigger(this.data);
  },

  onSignOut() {
    this.data.auth.userId = null;
    this.data.auth.token = null;
    this.data.auth.error = null;
    this.data.auth.errorMessage = null;
    localStorage.removeAuthToken();
    window.location = settings.signinUrl;
  },


  onDoAuthFailed: function (error) {
    this.data.auth.isProcessing = false;
    this.data.auth.isProcessed = true;
    this.data.auth.error = true;
    this.data.auth.errorMessage = error.message;
    localStorage.removeAuthToken();
    this.trigger(this.data);
  },

  onDoAuthProgressed: function () {
    this.data.auth.isProcessing = true;
    this.trigger(this.data);
  },

  onDoAuthCompleted: function (data) {
    this.data.auth.isProcessing = false;
    this.data.auth.isProcessed = true;
    this.data.auth.errorMessage = this.getError(data);
    this.data.auth.error = !!this.data.auth.errorMessage;

    if (typeof data.token !== 'undefined') {
      let tokenData = null;

      try {
        tokenData = jwtDecode(data.token);
      } catch (e) {
        console.log(e);
      }

      if (tokenData && typeof tokenData.id !== 'undefined') {
        this.data = this.getInitialState();
        this.data.auth.token = data.token;
        localStorage.setAuthToken(data.token);
        this.data.auth.userId = parseInt(tokenData.id, 10);
        Actions.getUserProfile(this.data.auth.token);
      } else {
        localStorage.removeAuthToken();
        this.data.auth.token = null;
      }
    }

    this.trigger(this.data);
  },


  onGetUserProfileFailed: function (error) {
    this.data.userProfile.isProcessing = false;
    this.data.userProfile.error = true;
    this.data.userProfile.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetUserProfileProgressed: function () {
    this.data.userProfile.isProcessing = true;
    this.trigger(this.data);
  },

  onGetUserProfileCompleted: function (data) {
    this.data.userProfile.isProcessing = false;
    this.data.userProfile.errorMessage = this.getError(data);
    this.data.userProfile.error = !!this.data.userProfile.errorMessage;

    if (!this.data.userProfile.error && data.length > 0) {
      this.data.userProfile.data = data[0];
      this.data.userProfile.isProcessed = true;

      if (window.intercomSettings) {
        let name = null;

        if (data[0].info.email.split('@').length > 0) {
          name = data[0].info.email.split('@')[0];
        }

        if (data[0].info.first_name) {
          name = data[0].info.first_name;
          if (data[0].info.last_name) {
            name = name + ' ' + data[0].info.last_name;
          }
        }
        window.intercomSettings.name = name;
        window.intercomSettings.email = data[0].info.email;
        window.intercomSettings.user_id = data[0].info.id;
        window.intercomSettings.created_at = data[0].info.created_at;

        if (data[0].orgs) {
          window.intercomSettings.companies = [];

          for (let i in data[0].orgs) {
            if (data[0].orgs.hasOwnProperty(i)) {
              window.intercomSettings.companies.push({
                company_id: data[0].orgs[i].id,
                name: data[0].orgs[i].name
              });
            }
          }
        }

        if (window.Intercom) {
          window.Intercom('update');
        }
      }
    }

    this.trigger(this.data);
  },


  onGetOrgsFailed: function (error) {
    this.data.orgProfile.isProcessing = false;
    this.data.orgProfile.error = true;
    this.data.orgProfile.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetOrgsProgressed: function (data) {
    this.data.orgProfile.isProcessing = true;
    this.data.orgProfile.orgId = data.orgId;

    this.trigger(this.data);
  },

  onGetOrgsCompleted: function (data) {
    this.data.orgProfile.isProcessing = false;
    this.data.orgProfile.errorMessage = this.getError(data.data);
    this.data.orgProfile.error = this.data.orgProfile.errorMessage;

    if (!this.data.orgProfile.error) {
      if (data.data.length > 0) {
        let users = [];

        this.data.orgProfile.prevAlias = null;
        this.data.orgProfile.data = data.data[0];
        for (let i in data.data[0].users) {
          if (data.data[0].users.hasOwnProperty(i)) {
            users.push(data.data[0].users[i]);
          }
        }

        this.data.orgProfile.data.users = users;
        this.data.orgProfile.orgId = data.orgId;
        this.data.orgProfile.isProcessed = true;
      } else {
        this.data.orgProfile.error = true;
        this.data.orgProfile.errorMessage =
          'You do not have permission to view this page.';
        this.data.orgProfile.errorCode = 403;
      }
    }

    this.trigger(this.data);
  },


  onGetOrgUsersFailed: function (error) {
    this.data.orgUsers.isProcessing = false;
    this.data.orgUsers.error = true;
    this.data.orgUsers.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetOrgUsersProgressed: function (orgId) {
    this.data.orgUsers.isProcessing = true;
    this.data.orgUsers.isRefreshed = false;
    this.data.orgUsers.isRefresh = false;
    if (this.data.orgUsers.orgId === orgId) {
      this.data.orgUsers.isRefresh = true;
    }
    this.data.orgUsers.orgId = orgId;
    this.data.orgUsers.isUpdating = false;
    this.data.orgUsers.isUpdated = false;
    this.data.orgUsers.updateUserId = null;
    this.data.orgUsers.isDeleting = false;
    this.data.orgUsers.isDeleted = false;
    this.data.orgUsers.deleteUserId = null;

    this.trigger(this.data);
  },

  onGetOrgUsersCompleted: function (orgId, data) {
    this.data.orgUsers.isProcessing = false;
    this.data.orgUsers.errorMessage = this.getError(data);
    this.data.orgUsers.error = !!this.data.orgUsers.errorMessage;

    if (this.data.orgUsers.isRefresh) {
      this.data.orgUsers.isRefreshed = true;
    }

    if (!this.data.orgUsers.error) {
      if (data.length > 0) {
        let users = [];
        let roles = [];

        this.data.orgUsers.data = data[0];
        for (let i in data[0].users) {
          if (data[0].users.hasOwnProperty(i)) {
            users.push(data[0].users[i]);
          }
        }

        for (let i in data[0].roles) {
          if (data[0].roles.hasOwnProperty(i)) {
            roles.push(data[0].roles[i]);
          }
        }

        this.data.orgUsers.data.users = users;
        this.data.orgUsers.data.roles = roles;
        this.data.orgUsers.orgId = orgId;
        this.data.orgUsers.isProcessed = true;
      } else {
        this.data.orgUsers.error = true;
        this.data.orgUsers.errorMessage =
          'You do not have permission to view this page.';
        this.data.orgUsers.errorCode = 403;
      }
    }

    this.trigger(this.data);
  },


  onUpdateOrgFailed: function (error) {
    this.data.orgProfile.isUpdating = false;
    this.data.orgProfile.updateError = true;
    this.data.orgProfile.updateErrorMessage = error.message;
    this.trigger(this.data);
  },

  onUpdateOrgProgressed: function (data) {
    this.data.orgProfile.updateErrorFields = null;
    this.data.orgProfile.isUpdating = true;

    if (data && data.name) {
      this.data.orgProfile.data.updateName = data.name;
    }

    if (data && data.alias) {
      this.data.orgProfile.data.updateAlias = data.alias;
    }

    if (data && data.users_autojoin) {
      this.data.orgProfile.data.updateUsersAutojoin = data['users_autojoin'];
    }

    this.trigger(this.data);
  },

  onUpdateOrgCompleted: function (data) {
    this.data.orgProfile.isUpdating = false;
    this.data.orgProfile.updateErrorMessage = this.getError(data);
    this.data.orgProfile.updateError = !!this.data.orgProfile.updateErrorMessage;

    if (!this.data.orgProfile.updateError && data.length > 0) {
      this.data.orgProfile.prevAlias = this.data.orgProfile.data?.alias;
      this.data.orgProfile.data = data[0];
      Actions.getUserProfile(this.data.auth.token);
      Actions.getOrgs(this.data.auth.token, this.data.orgProfile.orgId);
      this.data.orgProfile.updateErrorFields = null;
      Actions.showNotification('Organizations settings successfully saved.', 'success');
    } else {
      this.data.orgProfile.updateErrorFields = [];

      if (this.data.orgProfile.updateErrorMessage.indexOf('u_orgs_name') !==
        -1) {
        this.data.orgProfile.updateErrorMessage =
          'Organization with same name already exists.';
        this.data.orgProfile.updateErrorFields.push('name');
        this.data.orgProfile.updateError = true;
      }

      if (this.data.orgProfile.updateErrorMessage.indexOf('u_orgs_alias') !==
        -1) {
        this.data.orgProfile.updateErrorMessage =
          'Organization with the same alias already exists.';
        this.data.orgProfile.updateErrorFields.push('alias');
        this.data.orgProfile.updateError = true;
      }
    }

    this.trigger(this.data);
  },


  onCreateOrgFailed: function (error) {
    this.data.orgProfile.isUpdating = false;
    this.data.orgProfile.updateError = true;
    this.data.orgProfile.updateErrorMessage = error.message;
    this.trigger(this.data);
  },

  onCreateOrgProgressed: function (data) {
    this.data.orgProfile.updateErrorFields = null;
    this.data.orgProfile.isUpdating = true;
    this.data.orgProfile.data = {};

    if (data && data.name) {
      this.data.orgProfile.data.updateName = data.name;
    }

    if (data && data.alias) {
      this.data.orgProfile.data.updateAlias = data.alias;
    }

    this.trigger(this.data);
  },

  onCreateOrgCompleted: function (data) {
    this.data.orgProfile.isUpdating = false;
    this.data.orgProfile.updateErrorMessage = this.getError(data);
    this.data.orgProfile.updateError = !!this.data.orgProfile.updateErrorMessage;

    if (!this.data.orgProfile.updateError && data) {
      this.data.orgProfile.prevAlias = this.data.orgProfile.data.alias;
      this.data.orgProfile.data = data;
      Actions.getUserProfile(this.data.auth.token);
      Actions.getOrgs(this.data.auth.token, this.data.orgProfile.orgId);
      this.data.orgProfile.updateErrorFields = null;
      window.location.pathname = this.data.orgProfile.data.alias
    } else {
      this.data.orgProfile.updateErrorFields = [];

      if (this.data.orgProfile.updateErrorMessage.indexOf('u_orgs_name') !==
        -1) {
        this.data.orgProfile.updateErrorMessage =
          'Organization with same name already exists.';
        this.data.orgProfile.updateErrorFields.push('name');
      }

      if (this.data.orgProfile.updateErrorMessage.indexOf('u_orgs_alias') !==
        -1) {
        this.data.orgProfile.updateErrorMessage =
          'Organization with same alias already exists.';
        this.data.orgProfile.updateErrorFields.push('alias');
      }
    }

    this.trigger(this.data);
  },


  onInviteUserFailed: function (error) {
    this.data.inviteUser.isUpdating = false;
    this.data.inviteUser.isProcessed = false;
    this.data.inviteUser.updateError = true;
    this.data.inviteUser.updateErrorMessage = error.message;
    this.trigger(this.data);
  },

  onInviteUserProgressed: function (data) {
    this.data.inviteUser.updateErrorFields = null;
    this.data.inviteUser.isUpdating = true;
    this.data.inviteUser.data = {};

    if (data && data.email) {
      this.data.inviteUser.data.email = data.email;
    }

    this.trigger(this.data);
  },

  onInviteUserCompleted: function (data) {
    this.data.inviteUser.isUpdating = false;
    this.data.inviteUser.errorMessage = this.getError(data);
    this.data.inviteUser.error = !!this.data.inviteUser.errorMessage;

    if (!this.data.inviteUser.updateError && data) {
      this.data.inviteUser.prevEmail = this.data.inviteUser.data.alias;
      this.data.inviteUser.data = data;
      this.data.inviteUser.isProcessed = true;
      Actions.getOrgUsers(this.data.auth.token, this.data.orgProfile.orgId);
    }

    this.trigger(this.data);
  },


  onUseDemoDataFailed: function (error) {
    this.data.useDemoData.isProcessing = false;
    this.data.useDemoData.isProcessed = false;
    this.data.useDemoData.error = true;
    this.data.useDemoData.errorMessage = error.message;
    this.trigger(this.data);
  },

  onUseDemoDataProgressed: function () {
    this.data.useDemoData.updateErrorFields = null;
    this.data.useDemoData.isProcessing = true;
    this.data.useDemoData.data = {};
    this.trigger(this.data);
  },

  onUseDemoDataCompleted: function (data) {
    this.data.useDemoData.isProcessing = false;
    this.data.useDemoData.errorMessage = this.getError(data);
    this.data.useDemoData.error = !!this.data.useDemoData.errorMessage;

    if (this.data.useDemoData.error) {
      Actions.showNotification(this.data.useDemoData.errorMessage, 'error');
    } else {
      this.data.useDemoData.data = data;
      this.data.useDemoData.isProcessed = true;

      Actions.showNotification('Successfully joined the Demo organization.', 'success');
    }

    this.trigger(this.data);
  },


  onGetAccessTokensFailed: function (error) {
    this.data.userTokens.isProcessing = false;
    this.data.userTokens.error = true;
    this.data.userTokens.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetAccessTokensProgressed: function () {
    this.data.userTokens.isProcessing = true;
    this.trigger(this.data);
  },

  onGetAccessTokensCompleted: function (data) {
    this.data.userTokens.isProcessing = false;
    this.data.userTokens.errorMessage = this.getError(data.data);
    this.data.userTokens.error = !!this.data.userTokens.errorMessage;

    if (!this.data.userTokens.error) {
      this.data.userTokens.data = data.data;
      this.data.userTokens.orgId = data.orgId;

      this.data.userTokens.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetAccessTokenFailed: function (error) {
    this.data.tokenRequest.isProcessing = false;
    this.data.tokenRequest.error = true;
    this.data.tokenRequest.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetAccessTokenProgressed: function () {
    this.data.tokenRequest = {
      isProcessing: true,
      isProcessed: false,
      data: null,
      error: null
    };
    this.trigger(this.data);
  },

  onGetAccessTokenCompleted: function (data) {
    this.data.tokenRequest.isProcessing = false;
    this.data.tokenRequest.errorMessage = this.getError(data.data);
    this.data.tokenRequest.error = !!this.data.tokenRequest.errorMessage;

    if (!this.data.tokenRequest.error && data.data && data.orgId) {
      this.data.tokenRequest.data = data.data;
      this.data.tokenRequest.isProcessed = true;
      Actions.getAccessTokens(this.data.auth.token, data.orgId);
    } else if (this.data.tokenRequest.errorMessage.indexOf('timestamp') !==
      -1) {
      this.data.tokenRequest.errorMessage = 'Check the expiration date.';
    }

    this.trigger(this.data);
  },


  onHideGeneratedAccessToken: function () {
    this.data.tokenRequest.isProcessed = false;
    this.data.tokenRequest.data = null;
    this.trigger(this.data);
  },


  onRevokeAccessTokenFailed: function (error) {
    console.log('Revoking token', error);
  },

  onRevokeAccessTokenProgressed: function (id) {
    for (let i in this.data.userTokens.data) {
      if (this.data.userTokens.data[i].id === id) {
        this.data.userTokens.data[i].revoking = true;
      }
    }
  },

  onRevokeAccessTokenCompleted: function (data) {
    this.data.userTokens.revokeErrorMessage = this.getError(data.data);
    this.data.userTokens.revokeError =
      !!this.data.userTokens.revokeErrorMessage;

    Actions.hideGeneratedAccessToken();
    if (data && data.orgId) {
      Actions.getAccessTokens(this.data.auth.token, data.orgId);
    }

    if (this.data.userTokens.revokeError) {
      Actions.showNotification(this.data.userTokens.revokeErrorMessage, 'error');
    } else {
      Actions.showNotification('Token revoked.', 'success');
    }
  },


  onGetCheckupReportsFailed: function (error) {
    this.data.reports.isProcessing = false;
    this.data.reports.error = true;
    this.data.reports.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetCheckupReportsProgressed: function () {
    this.data.reports.isProcessing = true;
    this.trigger(this.data);
  },

  onGetCheckupReportsCompleted: function (data) {
    this.data.reports.isProcessing = false;
    this.data.reports.errorMessage = this.getError(data);
    this.data.reports.error = !!this.data.reports.errorMessage;

    if (!this.data.reports.error) {
      this.data.reports.data = data.data;
      this.data.reports.orgId = data.orgId;
      this.data.reports.projectId = data.projectId;
      this.data.reports.reportId = data.reportId;
      this.data.reports.isProcessed = true;
      if (!data.data.length && data.reportId) {
        this.data.reports.error = true;
        this.data.reports.errorMessage =
          'Specified report not found or you have no access.';
        this.data.reports.errorCode = 404;
      }
    }

    this.trigger(this.data);
  },


  onGetCheckupReportFilesFailed: function (error) {
    this.data.report.isProcessing = false;
    this.data.report.error = true;
    this.data.report.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetCheckupReportFilesProgressed: function () {
    this.data.report.isProcessing = true;
    this.data.report.data = null;
    this.data.report.reportId = null;
    this.trigger(this.data);
  },

  onGetCheckupReportFilesCompleted: function (data) {
    this.data.report.isProcessing = false;
    this.data.report.errorMessage = this.getError(data);
    this.data.report.error = !!this.data.report.errorMessage;

    if (!this.data.report.error) {
      this.data.report.reportId = data.reportId;
      this.data.report.type = data.type;
      this.data.report.data = data.data;
      this.data.report.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetCheckupReportFileFailed: function (error) {
    this.data.reportFile.isProcessing = false;
    this.data.reportFile.error = true;
    this.data.reportFile.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetCheckupReportFileProgressed: function () {
    this.data.reportFile.isProcessing = true;
    this.trigger(this.data);
  },

  onGetCheckupReportFileCompleted: function (id, data) {
    this.data.reportFile.isProcessing = false;
    this.data.reportFile.errorMessage = this.getError(data);
    this.data.reportFile.error = !!this.data.reportFile.errorMessage;


    if (!this.data.reportFile.error && data.length > 0) {
      let fileId = data[0].type + '_' + id;

      this.data.reportFile.files[fileId] = data[0];

      if (data[0].type === 'md') {
        let mdData = this.data.reportFile.files[fileId].data;
        // Replace [ not in links.
        mdData = mdData.split(/\[(?!.*\]\(.*\))/).join('\\[');
        // Replace ] not in links.
        mdData = mdData.split(/\](?!\(.*\))/).join('\\]');
        // Replace icons.
        mdData = mdData.split(':warning:').join(emoji.warning);
        mdData = mdData.split(':information_source:').join(emoji.large_blue_diamond);
        this.data.reportFile.files[fileId].data = mdData;
        this.data.reportFile.files[fileId].isProcessed = true;
      }
    } else {
      this.data.reportFile.error = true;
      this.data.reportFile.errorMessage = 'Specified file not found or you have no access.';
      this.data.reportFile.errorCode = 404;
    }

    this.trigger(this.data);
  },


  onGetJoeSessionsFailed: function (error) {
    this.data.sessions.isProcessing = false;
    this.data.sessions.error = true;
    this.data.sessions.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetJoeSessionsProgressed: function () {
    this.data.sessions.isProcessing = true;
    this.trigger(this.data);
  },

  onGetJoeSessionsCompleted: function (data) {
    this.data.sessions.isProcessing = false;
    this.data.sessions.errorMessage = this.getError(data);
    this.data.sessions.error = !!this.data.sessions.errorMessage;

    if (!this.data.sessions.error) {
      this.data.sessions.data = data.data;
      this.data.sessions.orgId = data.orgId;
      this.data.sessions.projectId = data.projectId;
      this.data.sessions.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetJoeSessionCommandsFailed: function (error) {
    this.data.commands.isProcessing = false;
    this.data.commands.error = true;
    this.data.commands.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetJoeSessionCommandsProgressed: function (params) {
    if (!params.lastId) {
      this.data.commands.data = [];
    }
    this.data.commands.isProcessing = true;
    this.data.commands.isProcessed = false;
    this.trigger(this.data);
  },

  onGetJoeSessionCommandsCompleted: function (data, params) {
    this.data.commands.isProcessing = false;
    this.data.commands.errorMessage = this.getError(data);
    this.data.commands.error = !!this.data.commands.errorMessage;
    this.data.commands.isProcessed = true;

    if (!this.data.commands.error) {
      if (params.limit) {
        if (data.length < params.limit) {
          this.data.commands.isComplete = true;
        } else {
          this.data.commands.isComplete = false;
        }
      }

      if (params.lastId && this.data.commands.data && params.limit) {
        this.data.commands.data = [...this.data.commands.data, ...data];
      } else {
        this.data.commands.data = data;
      }

      this.data.commands.isHistoryExists[params.orgId] =
        this.data.commands.isHistoryExists[params.orgId] || data.length > 0;
      this.data.commands.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetJoeSessionCommandFailed: function (error) {
    this.data.command.isProcessing = false;
    this.data.command.error = true;
    this.data.command.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetJoeSessionCommandProgressed: function () {
    this.data.command.isProcessing = true;
    this.trigger(this.data);
  },

  onGetJoeSessionCommandCompleted: function (data) {
    this.data.command.isProcessing = false;
    this.data.command.errorMessage = this.getError(data);
    this.data.command.error = !!this.data.command.errorMessage;

    if (!this.data.command.error && data && data.id) {
      const command = {
        projectId: data.project_id,
        sessionId: data.joe_session_id,
        projectName: data.project_name,
        commandId: data.id,
        command: data.command,
        query: data.query,
        response: data.response,
        planText: data.plan_text,
        planExecText: data.plan_execution_text,
        planExecJson: data.plan_execution_json,
        stats: data.stats,
        error: data.error,
        queryLocks: data.query_locks || '',
        slackUid: data.slack_uid,
        slackUsername: data.slack_username,
        slackChannel: data.slack_channel,
        slackTs: data.slack_ts,
        createdAt: data.created_at,
        createdFormatted: data.created_formatted,
        useremail: data.useremail,
        username: data.username
      };

      this.data.command.data = command;
      this.data.command.isProcessed = true;
    } else {
      this.data.command.error = true;
      this.data.command.errorMessage = 'Specified command not found or you have no access.';
      this.data.command.errorCode = 404;
    }

    this.trigger(this.data);
  },


  onGetProjectsFailed: function (error) {
    this.data.projects.isProcessing = false;
    this.data.projects.error = true;
    this.data.projects.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetProjectsProgressed: function () {
    this.data.projects.isProcessing = true;
    this.data.projects.orgId = null;
    this.trigger(this.data);
  },

  onGetProjectsCompleted: function (data) {
    this.data.projects.isProcessing = false;
    this.data.projects.errorMessage = this.getError(data.data);
    this.data.projects.error = !!this.data.projects.errorMessage;

    if (!this.data.projects.error) {
      if (data.orgId) {
        this.data.projects.orgId = data.orgId;
      }

      this.data.projects.data = data.data;
      this.data.projects.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onSetReportsProject(orgId, projectId) {
    this.data.reports.orgId = orgId;
    this.data.reports.projectId = projectId;
    Actions.getCheckupReports(this.data.auth.token, orgId, projectId);
    this.trigger(this.data);
  },

  onSetSessionsProject(orgId, projectId) {
    this.data.sessions.orgId = orgId;
    this.data.sessions.projectId = projectId;
    Actions.getJoeSessions(this.data.auth.token, orgId, projectId);
    this.trigger(this.data);
  },

  onSetDbLabInstancesProject(orgId, projectId) {
    this.data.dbLabInstances.orgId = orgId;
    this.data.dbLabInstances.projectId = projectId;
    Actions.getDbLabInstances(this.data.auth.token, orgId, projectId);
    this.trigger(this.data);
  },


  onGetDbLabInstancesFailed: function (error) {
    this.data.dbLabInstances.isProcessing = false;
    this.data.dbLabInstances.error = true;
    this.data.dbLabInstances.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetDbLabInstancesProgressed: function () {
    this.data.dbLabInstances.isProcessing = true;
    this.trigger(this.data);
  },

  onGetDbLabInstancesCompleted: function (data) {
    this.data.dbLabInstances.isProcessing = false;
    this.data.dbLabInstances.errorMessage = this.getError(data);
    this.data.dbLabInstances.error = !!this.data.dbLabInstances.errorMessage;

    if (!this.data.dbLabInstances.error) {
      let dblabs = {};

      for (let i in data.data) {
        if (data.data.hasOwnProperty(i)) {
          dblabs[data.data[i].id] = data.data[i];
        }
      }

      this.data.dbLabInstances.data = dblabs;
      this.data.dbLabInstances.orgId = data.orgId;
      this.data.dbLabInstances.projectId = data.projectId;
      this.data.dbLabInstances.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onAddDbLabInstanceFailed: function (error) {
    this.data.newDbLabInstance.isUpdating = false;
    this.data.newDbLabInstance.isProcessed = false;
    this.data.newDbLabInstance.updateError = true;
    this.data.newDbLabInstance.updateErrorMessage = error.message;
    this.trigger(this.data);
  },

  onAddDbLabInstanceProgressed: function (data) {
    this.data.newDbLabInstance.updateErrorFields = null;
    this.data.newDbLabInstance.isUpdating = true;
    this.data.newDbLabInstance.data = {};

    if (data && data.email) {
      this.data.newDbLabInstance.data.email = data.email;
    }

    this.trigger(this.data);
  },

  onAddDbLabInstanceCompleted: function (data) {
    this.data.newDbLabInstance.isUpdating = false;
    this.data.newDbLabInstance.errorMessage = this.getError(data.data);
    this.data.newDbLabInstance.error = !!this.data.newDbLabInstance.errorMessage;

    if (!this.data.newDbLabInstance.error && data.data) {
      this.data.newDbLabInstance.data = data.data;
      this.data.newDbLabInstance.isProcessed = true;
      // Update orgs and projects.
      Actions.getUserProfile(this.data.auth.token);
      Actions.getDbLabInstances(this.data.auth.token, data.orgId, data.data
        .project_id);
    }

    this.trigger(this.data);
  },

  onEditDbLabInstanceFailed: function (error) {
    this.data.newDbLabInstance.isUpdating = false;
    this.data.newDbLabInstance.isProcessed = false;
    this.data.newDbLabInstance.updateError = true;
    this.data.newDbLabInstance.updateErrorMessage = error.message;
    this.trigger(this.data);
  },

  onEditDbLabInstanceProgressed: function (data) {
    this.data.newDbLabInstance.updateErrorFields = null
    this.data.newDbLabInstance.isUpdating = true
    this.data.newDbLabInstance.data = {}

    if (data && data.email) {
      this.data.newDbLabInstance.data.email = data.email
    }
      
    this.trigger(this.data)
  },

  onEditDbLabInstanceCompleted: function (data) {
    this.data.newDbLabInstance.isUpdating = false;
    this.data.newDbLabInstance.errorMessage = this.getError(data.data);
    this.data.newDbLabInstance.error = !!this.data.newDbLabInstance.errorMessage;

    if (!this.data.newDbLabInstance.error && data.data) {
      this.data.newDbLabInstance.data = data.data;
      this.data.newDbLabInstance.isProcessed = true;
      // Update orgs and projects.
      Actions.getUserProfile(this.data.auth.token);
      Actions.getDbLabInstances(this.data.auth.token, data.orgId, data.data
        .project_id);
      
      if (window.location.href.indexOf('edit') > -1) {
        let url = window.location.href.split('/edit')[0]
        window.location.href = url
      }
    }
    this.trigger(this.data);
  },


  onCheckDbLabInstanceUrlFailed: function (error) {
    this.data.newDbLabInstance.isChecked = false;
    this.data.newDbLabInstance.isCheckProcessed = true;
    this.data.newDbLabInstance.isChecking = false;
    this.data.newDbLabInstance.checkingError = true;
    this.data.newDbLabInstance.checkingErrorMessage = error.message;
    this.trigger(this.data);
  },

  onCheckDbLabInstanceUrlProgressed: function () {
    this.data.newDbLabInstance.isChecked = false;
    this.data.newDbLabInstance.isCheckProcessed = false;
    this.data.newDbLabInstance.isChecking = true;
    this.trigger(this.data);
  },

  onCheckDbLabInstanceUrlCompleted: function (data) {
    this.data.newDbLabInstance.isChecking = false;
    this.data.newDbLabInstance.checkingErrorMessage = this.getError(data);
    this.data.newDbLabInstance.checkingError = !!this.data.newDbLabInstance
      .checkingErrorMessage;
    this.data.newDbLabInstance.isCheckProcessed = true;

    if (!this.data.newDbLabInstance.checkingError && data) {
      this.data.newDbLabInstance.isChecked = true;
    }

    this.trigger(this.data);
  },


  onDestroyDbLabInstanceFailed: function (error) {
    for (let i in this.data.dbLabInstances.data) {
      if (this.data.dbLabInstances.data.hasOwnProperty(i)) {
        this.data.dbLabInstances.data[i].isProcessing = false;
      }
    }

    this.data.dbLabInstances.isProcessingItem = false;
    this.data.dbLabInstances.itemError = true;
    this.data.dbLabInstances.itemErrorMessage = error.message;
    this.trigger(this.data);
  },

  onDestroyDbLabInstanceProgressed: function (data) {
    this.data.dbLabInstances.data[data.instanceId].isProcessing = true;
    this.data.newDbLabInstance.isProcessingItem = true;
    this.data.dbLabInstances.itemError = false;
    this.data.dbLabInstances.itemErrorMessage = null;
    this.trigger(this.data);
  },

  onDestroyDbLabInstanceCompleted: function (data) {
    this.data.dbLabInstances.isProcessingItem = false;
    this.data.dbLabInstances.itemErrorMessage = this.getError(data.data);
    this.data.dbLabInstances.itemError = !!this.data.dbLabInstances.itemErrorMessage;

    if (!this.data.dbLabInstances.itemError && data) {
      this.data.dbLabInstances.data[data.instanceId].isProcessing = false;
      Actions.getDbLabInstances(this.data.auth.token,
        this.data.dbLabInstances.orgId, this.data.dbLabInstances.projectId
      );
    }

    this.trigger(this.data);
  },


  onResetNewDbLabInstance: function () {
    this.data.newDbLabInstance = {
      isProcessing: false,
      isProcessed: false,
      isChecked: null,
      isCheckProcessed: false,
      data: null,
      error: null
    };
  },

  onGetDbLabInstanceStatusFailed: function (data, error) {
    this.data.dbLabInstanceStatus.isProcessing = false;
    this.data.dbLabInstanceStatus.isProcessed = true;
    this.data.dbLabInstanceStatus.error = true;
    this.data.dbLabInstanceStatus.errorMessage = error.message;

    if (data && data.instanceId) {
      this.data.dbLabInstanceStatus.instanceId = data.instanceId;

      if (!this.data.dbLabInstanceStatus.instances[data.instanceId]) {
        this.data.dbLabInstanceStatus.instances[data.instanceId] = {};
      }

      this.data.dbLabInstanceStatus.instances[data.instanceId]._isError =
        true;
      this.data.dbLabInstanceStatus.instances[data.instanceId]._isProcessed =
        false;
    }

    this.trigger(this.data);
  },

  onGetDbLabInstanceStatusProgressed: function (data) {
    this.data.dbLabInstanceStatus.isProcessing = true;
    this.data.dbLabInstanceStatus.isProcessed = false;
    this.data.dbLabInstanceStatus.error = false;
    this.data.dbLabInstanceStatus.data = {};

    if (data && data.instanceId) {
      this.data.dbLabInstanceStatus.instanceId = data.instanceId;

      if (!this.data.dbLabInstanceStatus.instances[data.instanceId]) {
        this.data.dbLabInstanceStatus.instances[data.instanceId] = {};
      }

      this.data.dbLabInstanceStatus.instances[data.instanceId]._isError =
        false;
      this.data.dbLabInstanceStatus.instances[data.instanceId]._isProcessed =
        false;
    }

    this.trigger(this.data);
  },

  onGetDbLabInstanceStatusCompleted: function (data) {
    this.data.dbLabInstanceStatus.isProcessing = false;
    this.data.dbLabInstanceStatus.errorMessage = this.getError(data.data);
    this.data.dbLabInstanceStatus.error = !!this.data.dbLabInstanceStatus
      .errorMessage;

    if (!this.data.dbLabInstanceStatus.error && data.data && data.instanceId) {
      this.data.dbLabInstanceStatus.instanceId = data.instanceId;

      if (!this.data.dbLabInstanceStatus.instances[data.instanceId]) {
        this.data.dbLabInstanceStatus.instances[data.instanceId] = {};
      }

      this.data.dbLabInstanceStatus.instances[data.instanceId] = data.data;

      if (this.data.dbLabInstances.data &&
        this.data.dbLabInstances.data[data.instanceId]) {
        this.data.dbLabInstances.data[data.instanceId].state = data.data.state;
      }

      this.data.dbLabInstanceStatus.isProcessed = true;
      this.data.dbLabInstanceStatus.instances[data.instanceId]._isProcessed =
        true;
      this.data.dbLabInstanceStatus.instances[data.instanceId]._isError =
        false;
    }

    this.trigger(this.data);
  },

  onDownloadReportJsonFilesProgressed: function () {
    this.data.report.isDownloading = true;
    this.trigger(this.data);
  },

  onDownloadReportJsonFilesCompleted: function () {
    this.data.report.isDownloading = false;
    this.trigger(this.data);
  },


  onAddOrgDomainFailed: function (error) {
    this.data.orgProfile.orgDomains.isProcessing = false;
    this.data.orgProfile.orgDomains.isProcessed = false;
    this.data.orgProfile.orgDomains.error = true;
    this.data.orgProfile.orgDomains.errorMessage = error.message;
    Actions.showNotification(this.data.orgProfile.orgDomains.errorMessage, 'error');
    this.trigger(this.data);
  },

  onAddOrgDomainProgressed: function () {
    this.data.orgProfile.orgDomains.isProcessing = true;
    this.data.orgProfile.orgDomains.isProcessed = false;
    this.trigger(this.data);
  },

  onAddOrgDomainCompleted: function (data) {
    this.data.orgProfile.orgDomains.isProcessing = false;
    this.data.orgProfile.orgDomains.isProcessed = true;
    this.data.orgProfile.orgDomains.errorMessage = this.getError(data);
    this.data.orgProfile.orgDomains.error =
      !!this.data.orgProfile.orgDomains.errorMessage;
    Actions.showNotification('Domain successfully added. To confirm it, ' +
      'please contact the Support team.', 'success');
    Actions.getOrgs(this.data.auth.token, data.orgId);
  },


  onDeleteOrgDomainFailed: function (error) {
    this.data.orgProfile.orgDomains.isDeleting = false;
    this.data.orgProfile.orgDomains.isDeleted = false;
    this.data.orgProfile.orgDomains.error = true;
    this.data.orgProfile.orgDomains.errorMessage = error.message;
    Actions.showNotification(this.data.orgProfile.orgDomains.errorMessage, 'error');
    this.trigger(this.data);
  },

  onDeleteOrgDomainProgressed: function (data) {
    this.data.orgProfile.orgDomains.domainId = data.domainId;
    this.data.orgProfile.orgDomains.isDeleting = true;
    this.data.orgProfile.orgDomains.isDeleted = false;
    this.trigger(this.data);
  },

  onDeleteOrgDomainCompleted: function (data) {
    this.data.orgProfile.orgDomains.isDeleting = false;
    this.data.orgProfile.orgDomains.isDeleted = true;
    this.data.orgProfile.orgDomains.errorMessage = this.getError(data);
    this.data.orgProfile.orgDomains.error =
      !!this.data.orgProfile.orgDomains.errorMessage;
    this.data.orgProfile.orgDomains.domainId = null;
    Actions.getOrgs(this.data.auth.token, data.orgId);
    Actions.showNotification('Domain removed.', 'success');
  },


  onSetJoeInstancesProject(orgId, projectId) {
    this.data.joeInstances.orgId = orgId;
    this.data.joeInstances.projectId = projectId;
    Actions.getJoeInstances(this.data.auth.token, orgId, projectId);
    this.trigger(this.data);
  },


  onGetJoeInstancesFailed: function (error) {
    this.data.joeInstances.isProcessing = false;
    this.data.joeInstances.error = true;
    this.data.joeInstances.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetJoeInstancesProgressed: function () {
    this.data.joeInstances.isProcessing = true;
    this.trigger(this.data);
  },

  onGetJoeInstancesCompleted: function (orgId, projectId, data) {
    this.data.joeInstances.isProcessing = false;
    this.data.joeInstances.errorMessage = this.getError(data);
    this.data.joeInstances.error = !!this.data.joeInstances.errorMessage;

    if (!this.data.joeInstances.error) {
      let instances = {};

      for (let i in data) {
        if (data.hasOwnProperty(i)) {
          instances[data[i].id] = data[i];
        }
      }

      this.data.joeInstances.data = instances;
      this.data.joeInstances.orgId = orgId;
      this.data.joeInstances.projectId = projectId;
      this.data.joeInstances.isProcessed = true;
    }

    this.trigger(this.data);
  },


  getJoeInstance(instanceId) {
    if (!this.data.joeInstance.instances[instanceId]) {
      this.data.joeInstance.instances[instanceId] =
      Object.assign({}, joeInstance);
    }

    return this.data.joeInstance.instances[instanceId];
  },

  getJoeInstanceChannelMessages(instanceId, channelId) {
    const instance = this.getJoeInstance(instanceId);
    const messages = instance.messages;

    if (!messages[channelId]) {
      messages[channelId] = {};
    }

    return messages[channelId];
  },


  getJoeMessageArtifacts(instanceId, channelId, messageId) {
    const messages = this.getJoeInstanceChannelMessages(instanceId, channelId);

    if (messages.hasOwnProperty(messageId)) {
      if (!messages[messageId].hasOwnProperty('artifacts')) {
        messages[messageId].artifacts = {};
      }
    } else {
      return null;
    }

    return messages[messageId].artifacts;
  },


  onGetJoeInstanceChannelsFailed: function (instanceId, error) {
    const instance = this.getJoeInstance(instanceId);
    instance.isChannelsProcessing = false;
    instance.channelsError = true;
    instance.channelsErrorMessage = error.message;
    this.trigger(this.data);
  },

  onGetJoeInstanceChannelsProgressed: function (instanceId) {
    const instance = this.getJoeInstance(instanceId);
    instance.isChannelsProcessing = true;
    instance.channelsErrorMessage = null;
    instance.channelsError = false;
    this.trigger(this.data);
  },

  onGetJoeInstanceChannelsCompleted: function (instanceId, data) {
    const instance = this.getJoeInstance(instanceId);
    instance.isChannelsProcessing = false;
    instance.channelsErrorMessage = this.getError(data);
    instance.channelsError = !!instance.channelsErrorMessage;

    if (!instance.channelsError) {
      instance.channels = {};

      for (let i = data.length - 1; i >= 0; i--) {
        instance.channels[data[i].channel_id] = {
          channelId: data[i].channel_id
        };
      }

      instance.isChannelsProcessed = true;
    }

    this.trigger(this.data);
  },


  onSendJoeInstanceCommandFailed: function (instanceId, error) {
    const instance = this.getJoeInstance(instanceId);
    instance.isCommandProcessing = false;
    instance.commandError = true;
    instance.commandErrorMessage = error.message;
    this.trigger(this.data);
  },

  onSendJoeInstanceCommandProgressed: function (instanceId) {
    const instance = this.getJoeInstance(instanceId);
    instance.isCommandProcessing = true;
    instance.commandErrorMessage = null;
    instance.commandError = false;
    this.trigger(this.data);
  },

  setJoeInstanceMessage(messages, updatedMessage) {
    const messageId = updatedMessage.id ? updatedMessage.id : updatedMessage.message_id;

    const message = messages[messageId];

    // TODO(Anton): React should process this. Messages update flow refactor needed.
    const isMessageChanged =
      !message ||
      message.status !== updatedMessage.status ||
      message.delivery_status !== updatedMessage.delivery_status ||
      message.message !== updatedMessage.message ||
      message.created_at !== updatedMessage.created_at;

    messages[messageId] = updatedMessage;

    if (isMessageChanged) {
      const actualMessage = messages[messageId];
      // Update message only if status changed. New one added or artifacts added.
      actualMessage.formattedMessage = preformatJoeMessage(actualMessage.message);
      actualMessage.formattedTime = format.formatTimestamp(actualMessage.created_at);
    }

    return isMessageChanged;
  },

  onSendJoeInstanceCommandCompleted: function (instanceId, data) {
    const instance = this.getJoeInstance(instanceId);
    let changed = false;
    instance.isCommandProcessing = false;
    instance.commandErrorMessage = this.getError(data);
    instance.commandError = !!instance.commandErrorMessage;

    if (data && data.session_id && data.message_id && data.channel_id) {
      instance.channels[data.channel_id].sessionId = data.session_id;
      const messages = this.getJoeInstanceChannelMessages(instanceId, data.channel_id);
      changed = this.setJoeInstanceMessage(messages, data);

      if (!instance.channels[data.channel_id].wsOpen) {
        Actions.getJoeInstanceMessages(this.data.auth.token, instanceId,
          data.channel_id, data.session_id);
        try {
          Actions.initJoeWebSocketConnection(
            instanceId,
            data.channel_id,
            'joe-' + data.session_id + '-' + md5(data.channel_id));
        } catch (e) {
          instance.wsErrorMessage =
            'Command send, but cannot start persistent network connection. ' +
            'Execute any command to try to start the connection again.';
        }
      }

      instance.isCommandProcessed = true;
    }

    if (changed) {
      this.trigger(this.data);
    }
  },

  onJoeWebSocketMessage: function (instanceId, channelId, event) {
    const messages = this.getJoeInstanceChannelMessages(instanceId, channelId);

    if (event.data) {
      let data;
      try {
        data = JSON.parse(event.data);
      } catch (e) {
        console.log('Cannot parse WS message');
      }

      if (data && data.payload && data.payload.id) {
        console.log('WebSocket message payload', data);
        if (this.setJoeInstanceMessage(messages, data.payload)) {
          this.trigger(this.data);
        }
      }
    }
  },

  onCloseJoeWebSocketConnection: function (instanceId) {
    const instance = this.getJoeInstance(instanceId);

    instance.wsClose = true;
    if (instance.ws) {
      instance.ws.close();
    }
  },

  restoreJoeWebSocketConnection: function (instanceId, channelId, wsChannelId) {
    const instance = this.getJoeInstance(instanceId);

    if (!instance.wsClose && instance.channels[channelId].wsOpen) {
      instance.channels[channelId].wsRetryConnectionCount =
        instance.channels[channelId].wsRetryConnectionCount ?
          instance.channels[channelId].wsRetryConnectionCount + 1 : 1;

      console.log('Try restore connection', instance.wsRetryConnectionCount ?
        instance.wsRetryConnectionCount : 1);

      if (instance.channels[channelId].wsRetryConnectionCount >
          WS_MAX_RETRY_CONNECTION_COUNT) {
        instance.channels[channelId].wsErrorMessage =
          'There are issues with a persistent network connection. ' +
          'Cannot restore connection. ' +
          'Execute any command to try to restore the connection again.';
        instance.channels[channelId].wsOpen = false;
        instance.channels[channelId].wsFailed = true;
        return;
      }

      Actions.initJoeWebSocketConnection(instanceId, channelId, wsChannelId);
    } else {
      instance.channels[channelId].wsOpen = false;
    }

    this.trigger(this.data);
  },

  onInitJoeWebSocketConnection: function (instanceId, channelId, wsChannelId) {
    const that = this;
    const instance = this.getJoeInstance(instanceId);
    let ws = createWebSocket(settings.wsServer,
      wsChannelId + '/' + this.data.auth.token);

    instance.ws = ws;
    ws.onopen = function () {
      console.log('WebSocket connection opened.', wsChannelId);
      instance.channels[channelId].wsOpen = true;
      instance.channels[channelId].wsError = null;
      instance.channels[channelId].wsErrorMessage = null;
      instance.channels[channelId].wsRetryConnectionCount = null;
      instance.channels[channelId].wsFailed = null;
      ws.send('?');
      that.trigger(that.data);
    };

    ws.onmessage = function (event) {
      console.log('WebSocket message payload', event);
      that.onJoeWebSocketMessage(instanceId, channelId, event);
    };

    ws.onerror = function (error) {
      console.log('WebSocket error.', error);

      if (instance.channels[channelId].wsOpen) {
        that.restoreJoeWebSocketConnection(instanceId, channelId, wsChannelId);
      } else if (!instance.channels[channelId].wsFailed) {
        instance.channels[channelId].wsError = error;
        instance.channels[channelId].wsErrorMessage =
          'There are issues with a persistent network connection. ' +
          'Cannot start connection. ' +
          'Execute any command to try to start the connection again.';
      }

      that.trigger(that.data);
    };

    ws.onclose = function (event) {
      console.log('WebSocket closed.', event);
      console.log('WebSocket connection closed with code: ' + event.code +
        ' reason: ' + event.reason);

      if (event.wasClean) {
        console.log('WebSocket connection closed clear.');
      } else {
        console.log('WebSocket connection failed.');
      }

      that.restoreJoeWebSocketConnection(instanceId, channelId, wsChannelId);
    };
  },


  onGetJoeInstanceMessagesFailed: function (instanceId, channelId, error) {
    let instance = this.getJoeInstance(instanceId);
    instance.isMessagesProcessing = false;
    instance.commandError = true;
    instance.commandErrorMessage = error.message;
    this.trigger(this.data);
  },

  onGetJoeInstanceMessagesProgressed: function (instanceId) {
    this.getJoeInstance(instanceId).isMessagesProcessing = true;
    this.trigger(this.data);
  },

  onGetJoeInstanceMessagesCompleted: function (instanceId, channelId, data) {
    let instance = this.getJoeInstance(instanceId);
    const messages = this.getJoeInstanceChannelMessages(instanceId, channelId);
    let lastId = instance.clear && instance.clear[channelId] ?
      instance.clear[channelId] : 0;
    let changed = false;

    instance.isMessagesProcessing = false;
    instance.messagesErrorMessage = this.getError(data);
    instance.messagesError = !!instance.messagesErrorMessage;

    if (data) {
      for (let i in data) {
        if (data.hasOwnProperty(i)) {
          // Update only if message updated in database.
          if (data[i].id > lastId && (!messages[data[i].id] ||
            messages[data[i].id].updated_at !== data[i].updated_at)) {
            changed = this.setJoeInstanceMessage(messages, data[i]) || changed;
          }
        }
      }

      instance.isMessagesProcessed = false;
    }

    if (changed) {
      this.trigger(this.data);
    }
  },

  onGetJoeMessageArtifactsFailed: function (instanceId, channelId, error) {
    let instance = this.getJoeInstance(instanceId);
    instance.isProcessingMessageArtifactsId = false;
    instance.isProcessingMessageArtifactsError = error;
    if (instance.isProcessingMessageArtifactsId) {
      let artifacts = this.getJoeMessageArtifacts(
        instanceId,
        channelId,
        instance.isProcessingMessageArtifactsId);
      if (!artifacts) {
        return;
      }
      artifacts.isProcessing = false;
      artifacts.error = true;
    }
    this.trigger(this.data);
  },

  onGetJoeMessageArtifactsProgressed: function (instanceId, channelId, messageId) {
    let instance = this.getJoeInstance(instanceId);
    instance.isProcessingMessageArtifactsId = messageId;
    instance.isProcessingMessageArtifactsError = null;
    let artifacts = this.getJoeMessageArtifacts(instanceId, channelId, messageId);

    if (artifacts) {
      artifacts.isProcessing = true;
    }

    this.trigger(this.data);
  },

  onGetJoeMessageArtifactsCompleted: function (instanceId, channelId, messageId, data) {
    let instance = this.getJoeInstance(instanceId);
    instance.isProcessingMessageArtifactsErrorMessage =
      this.getError(data);
    instance.isProcessingMessageArtifactsError =
      !!instance.isProcessingMessageArtifactsErrorMessage;

    let artifacts = this.getJoeMessageArtifacts(
      instanceId,
      channelId,
      instance.isProcessingMessageArtifactsId
    );

    if (data && artifacts) {
      artifacts['files'] = {};

      for (let i = data.length - 1; i >= 0; i--) {
        artifacts['files'][data[i].id] = data[i];
      }

      artifacts.isProcessing = false;
    }

    instance.isProcessingMessageArtifactsId = null;
    instance.isMessagesProcessed = false;
    this.trigger(this.data);
  },


  onResetNewJoeInstance: function () {
    this.data.newJoeInstance = Object.assign({}, storeItem);
    this.data.newJoeInstance.isChecked = null;
    this.data.newJoeInstance.isCheckProcessed = false;
  },

  onCheckJoeInstanceUrlFailed: function (error) {
    this.data.newJoeInstance.isChecked = false;
    this.data.newJoeInstance.isCheckProcessed = true;
    this.data.newJoeInstance.isChecking = false;
    this.data.newJoeInstance.checkingError = true;
    this.data.newJoeInstance.checkingErrorMessage = error.message;
    this.trigger(this.data);
  },

  onCheckJoeInstanceUrlProgressed: function () {
    this.data.newJoeInstance.isChecked = false;
    this.data.newJoeInstance.isCheckProcessed = false;
    this.data.newJoeInstance.isChecking = true;
    this.trigger(this.data);
  },

  onCheckJoeInstanceUrlCompleted: function (data) {
    this.data.newJoeInstance.isChecking = false;
    this.data.newJoeInstance.checkingErrorMessage = this.getError(data);
    this.data.newJoeInstance.checkingError = !!this.data.newJoeInstance
      .checkingErrorMessage;
    this.data.newJoeInstance.isCheckProcessed = true;

    if (!this.data.newJoeInstance.checkingError && data) {
      this.data.newJoeInstance.isChecked = data.verified;
    }

    this.trigger(this.data);
  },


  onAddJoeInstanceFailed: function (error) {
    this.data.newJoeInstance.isUpdating = false;
    this.data.newJoeInstance.isProcessed = false;
    this.data.newJoeInstance.updateError = true;
    this.data.newJoeInstance.updateErrorMessage = error.message;
    this.trigger(this.data);
  },

  onAddJoeInstanceProgressed: function (data) {
    this.data.newJoeInstance.updateErrorFields = null;
    this.data.newJoeInstance.isUpdating = true;
    this.data.newJoeInstance.data = {};

    if (data && data.email) {
      this.data.newJoeInstance.data.email = data.email;
    }

    this.trigger(this.data);
  },

  onAddJoeInstanceCompleted: function (orgId, data) {
    this.data.newJoeInstance.isUpdating = false;
    this.data.newJoeInstance.errorMessage = data && data.details ?
      data.details : null;
    this.data.newJoeInstance.error = !!this.data.newJoeInstance.errorMessage;

    if (!this.data.newJoeInstance.error && data && orgId) {
      this.data.newJoeInstance.data = data;
      this.data.newJoeInstance.isProcessed = true;

      // Update orgs and projects.
      Actions.getUserProfile(this.data.auth.token);
      Actions.getJoeInstances(this.data.auth.token, orgId, data.project_id);
    }

    this.trigger(this.data);
  },


  onDestroyJoeInstanceFailed: function (error) {
    for (let i in this.data.joeInstances.data) {
      if (this.data.joeInstances.data.hasOwnProperty(i)) {
        this.data.joeInstances.data[i].isProcessing = false;
      }
    }

    this.data.joeInstances.isProcessingItem = false;
    this.data.joeInstances.itemError = true;
    this.data.joeInstances.itemErrorMessage = error.message;
    this.trigger(this.data);
  },

  onDestroyJoeInstanceProgressed: function (instanceId) {
    this.data.joeInstances.data[instanceId].isProcessing = true;
    this.data.newDbLabInstance.isProcessingItem = true;
    this.data.joeInstances.itemError = false;
    this.data.joeInstances.itemErrorMessage = null;
    this.trigger(this.data);
  },

  onDestroyJoeInstanceCompleted: function (instanceId, data) {
    this.data.joeInstances.isProcessingItem = false;
    this.data.joeInstances.itemErrorMessage = this.getError(data);
    this.data.joeInstances.itemError = !!this.data.joeInstances.itemErrorMessage;

    if (!this.data.joeInstances.itemError && data && instanceId) {
      this.data.joeInstances.data[instanceId].isProcessing = false;
      Actions.getJoeInstances(this.data.auth.token,
        this.data.joeInstances.orgId, this.data.joeInstances.projectId
      );
    }
  },


  onGetExternalVisualizationDataFailed: function (error) {
    this.data.externalVisualization.isProcessing = false;
    this.data.externalVisualization.isProcessed = false;
    this.data.externalVisualization.error = error;
    this.trigger(this.data);
  },

  onGetExternalVisualizationDataProgressed: function (type, plan, query) {
    this.data.externalVisualization.isProcessing = true;
    this.data.externalVisualization.isProcessed = false;
    this.data.externalVisualization.error = null;
    this.data.externalVisualization.type = type;
    this.data.externalVisualization.plan = plan;
    this.data.externalVisualization.query = query;
    this.trigger(this.data);
  },

  onGetExternalVisualizationDataCompleted: function (type, plan, query, url) {
    this.data.externalVisualization.isProcessing = false;
    this.data.externalVisualization.isProcessed = true;
    this.data.externalVisualization.error = null;
    this.data.externalVisualization.type = type;
    this.data.externalVisualization.plan = plan;
    this.data.externalVisualization.query = query;
    this.data.externalVisualization.url = url;
    this.trigger(this.data);
  },

  onCloseExternalVisualization: function () {
    this.data.externalVisualization.isProcessing = false;
    this.data.externalVisualization.isProcessed = true;
    this.data.externalVisualization.error = null;
    this.data.externalVisualization.type = null;
    this.data.externalVisualization.plan = null;
    this.data.externalVisualization.query = null;
    this.data.externalVisualization.url = null;
    this.trigger(this.data);
  },


  onDeleteJoeSessionsFailed: function (error) {
    this.data.sessions.isDeleting = false;
    this.data.sessions.isDeleted = false;
    this.data.sessions.deleteError = this.getError(error);
    this.trigger(this.data);
  },

  onDeleteJoeSessionsProgressed: function () {
    this.data.sessions.isDeleting = true;
    this.data.sessions.isDeleted = false;
    this.data.sessions.deleteError = null;
    this.trigger(this.data);
  },

  onDeleteJoeSessionsCompleted: function (ids, data) {
    this.data.sessions.isDeleting = false;
    this.data.sessions.isDeleted = (ids.length === data.sessions);
    this.data.sessions.deleteError = null;

    if (this.data.sessions.isDeleted) {
      Actions.showNotification(
        `${data.sessions} session(s), ${data.commands} ` +
        ` command(s), and ${data.messages} message(s) with artifacts successfully removed.`,
        'success',
        5000
      );

      for (let i in ids) {
        if (!ids.hasOwnProperty(i)) {
          continue;
        }

        for (let j in this.data.sessions.data) {
          if (this.data.sessions.data[j].id === ids[i]) {
            delete this.data.sessions.data[j];
          }
        }
      }
    } else {
      Actions.getJoeSessions(this.data.auth.token, this.data.sessions.orgId,
        this.data.sessions.projectId);
    }

    this.trigger(this.data);
  },


  onDeleteJoeCommandsFailed: function (error) {
    this.data.commands.isDeleting = false;
    this.data.commands.isDeleted = false;
    this.data.commands.deleteError = this.getError(error);
    this.trigger(this.data);
  },

  onDeleteJoeCommandsProgressed: function () {
    this.data.commands.isDeleting = true;
    this.data.commands.isDeleted = false;
    this.data.commands.deleteError = null;
    this.trigger(this.data);
  },

  onDeleteJoeCommandsCompleted: function (ids, data) {
    this.data.commands.isDeleting = false;
    this.data.commands.isDeleted = (ids.length === data.commands);
    this.data.commands.deleteError = null;

    if (this.data.commands.isDeleted) {
      Actions.showNotification(
        `${data.commands} command(s) removed.`,
        'success',
        5000
      );

      // Delete from list successfully removed items.
      for (let i in ids) {
        if (!ids.hasOwnProperty(i)) {
          continue;
        }

        for (let j in this.data.commands.data) {
          if (this.data.commands.data[j].id === ids[i]) {
            delete this.data.commands.data[j];
          }
        }
      }
    }

    this.trigger(this.data);
  },


  onDeleteCheckupReportsFailed: function (error) {
    this.data.reports.isDeleting = false;
    this.data.reports.isDeleted = false;
    this.data.reports.deleteError = this.getError(error);
    this.trigger(this.data);
  },

  onDeleteCheckupReportsProgressed: function () {
    this.data.reports.isDeleting = true;
    this.data.reports.isDeleted = false;
    this.data.reports.deleteError = null;
    this.trigger(this.data);
  },

  onDeleteCheckupReportsCompleted: function (ids, data) {
    this.data.reports.isDeleting = false;
    this.data.reports.isDeleted = (ids.length === data.reports);
    this.data.reports.deleteError = null;

    if (this.data.reports.isDeleted) {
      Actions.showNotification(
        `${data.reports} report(s) and ` +
        `${data.jsons + data.texts} file(s) successfully removed.`,
        'success',
        5000
      );

      for (let i in ids) {
        if (!ids.hasOwnProperty(i)) {
          continue;
        }

        for (let j in this.data.reports.data) {
          if (this.data.reports.data[j].id === ids[i]) {
            delete this.data.reports.data[j];
          }
        }
      }
    } else {
      Actions.getCheckupReports(this.data.auth.token, this.data.reports.orgId,
        this.data.reports.projectId);
    }

    this.trigger(this.data);
  },

  onJoeCommandFavoriteCompleted: function (data, commandId, favorite) {
    let errorMessage = this.getError(data);
    if (errorMessage) {
      Actions.showNotification(errorMessage, 'error');
      return;
    }

    for (let c in this.data.commands.data) {
      if (this.data.commands.data[c].id === commandId) {
        this.data.commands.data[c].is_favorite = favorite;
      }
    }

    if (favorite) {
      Actions.showNotification(
        'Command added to your favorites.',
        'success',
        5000
      );
    } else {
      Actions.showNotification(
        'Command removed from your favorites.',
        'success',
        5000
      );
    }

    this.trigger(this.data);
  },


  onClearJoeInstanceChannelMessages: function (instanceId, channelId) {
    let instance = this.getJoeInstance(instanceId);
    const messages = instance.messages;
    const msgIds = Object.keys(messages[channelId] ? messages[channelId] : {});
    let maxMessageId = Math.max(...msgIds);

    instance.clear = {};
    instance.clear[channelId] = maxMessageId;
    messages[channelId] = {};
    this.trigger(this.data);
  },

  onGetSharedUrlDataFailed: function (error) {
    this.data.sharedUrlData.isProcessing = false;
    this.data.sharedUrlData.isProcessed = false;
    this.data.sharedUrlData.error = this.getError(error);
    this.trigger(this.data);
  },

  onGetSharedUrlDataProgressed: function () {
    this.data.sharedUrlData.isProcessing = true;
    this.data.sharedUrlData.isProcessed = false;
    this.data.sharedUrlData.error = null;
    this.trigger(this.data);
  },

  onGetSharedUrlDataCompleted: function (uuid, data) {
    this.data.sharedUrlData.isProcessing = false;
    this.data.sharedUrlData.isProcessed = true;
    this.data.sharedUrlData.errorMessage = this.getError(data);
    this.data.sharedUrlData.error = !!this.data.sharedUrlData.errorMessage;

    if (!this.data.sharedUrlData.error) {
      this.data.sharedUrlData.data = data;

      if (data.url && data.url.object_type) {
        switch (data.url.object_type) {
        case 'command':
          this.onGetJoeSessionCommandCompleted(data.url_data);
          this.data.command.projectId = data.url_data.project_id;
          break;
        default:
          this.data.sharedUrlData.errorMessage = 'Unknown page type.';
          this.data.sharedUrlData.error = true;
          break;
        }
      }
    }

    this.trigger(this.data);
  },

  onAddSharedUrlProgressed() {
    this.data.shareUrl.isAdding = true;

    this.trigger(this.data);
  },

  onAddSharedUrlCompleted(params, data) {
    this.data.shareUrl.isAdding = false;
    this.data.shareUrl.data = data;

    if (!this.data.sharedUrls[params.objectType]) {
      this.data.sharedUrls[params.objectType] = {};
    }

    this.data.sharedUrls[params.objectType][params.objectId] = data;

    this.trigger(this.data);
  },

  onRemoveSharedUrlProgressed() {
    this.data.shareUrl.isRemoving = true;

    this.trigger(this.data);
  },

  onRemoveSharedUrlCompleted(orgId, objectType, objectId) {
    this.data.shareUrl.isRemoving = false;

    if (this.data.sharedUrls[objectType] &&
      this.data.sharedUrls[objectType][objectId]) {
      this.data.sharedUrls[objectType][objectId] = {
        object_type: objectType,
        object_id: objectId
      };
    }

    this.trigger(this.data);
  },

  onShowShareUrlDialog: function (orgId, objectType, objectId, remark) {
    this.data.shareUrl.open = true;
    this.data.shareUrl.objectType = objectType;
    this.data.shareUrl.objectId = objectId;
    this.data.shareUrl.uuid = uuidv4();
    this.data.shareUrl.remark = remark;
    this.data.shareUrl.orgId = orgId;

    Actions.getSharedUrl(this.data.auth.token, orgId, objectType, objectId);

    this.trigger(this.data);
  },

  onCloseShareUrlDialog: function (close, save, share) {
    if (save) {
      if (this.data.shareUrl.data && this.data.shareUrl.data.uuid && !share) {
        Actions.removeSharedUrl(this.data.auth.token, this.data.shareUrl.orgId,
          this.data.shareUrl.objectType, this.data.shareUrl.objectId,
          this.data.shareUrl.data.uuid);
        this.data.shareUrl.data = null;
        this.data.shareUrl.uuid = uuidv4();
      }

      if ((!this.data.shareUrl.data ||
        (this.data.shareUrl.data && !this.data.shareUrl.data.uuid))
         && share) {
        Actions.addSharedUrl(
          this.data.auth.token,
          {
            orgId: this.data.shareUrl.orgId,
            url: window.location.href,
            objectType: this.data.shareUrl.objectType,
            objectId: this.data.shareUrl.objectId,
            uuid: this.data.shareUrl.uuid
          }
        );
      }
    }

    if (close) {
      this.data.shareUrl.open = false;
      this.data.shareUrl.objectType = null;
      this.data.shareUrl.objectId = null;
      this.data.shareUrl.orgId = null;
      this.data.shareUrl.data = null;
      this.data.shareUrl.uuid = uuidv4();
    }

    this.trigger(this.data);
  },

  onGetSharedUrlFailed: function (error) {
    this.data.shareUrl.isProcessing = false;
    this.data.shareUrl.isProcessed = false;
    this.data.shareUrl.error = this.getError(error);
    this.trigger(this.data);
  },

  onGetSharedUrlProgressed: function () {
    this.data.shareUrl.isProcessing = true;
    this.data.shareUrl.isProcessed = false;
    this.data.shareUrl.error = null;
    this.trigger(this.data);
  },

  onGetSharedUrlCompleted: function (orgId, objectType, objectId, data) {
    this.data.shareUrl.isProcessing = false;
    this.data.shareUrl.isProcessed = true;
    this.data.shareUrl.errorMessage = this.getError(data);
    this.data.shareUrl.error = !!this.data.shareUrl.errorMessage;

    if (!this.data.sharedUrls[objectType]) {
      this.data.sharedUrls[objectType] = {};
    }

    if (!this.data.shareUrl.error) {
      this.data.shareUrl.data = data;
      this.data.sharedUrls[objectType][objectId] = data;
    } else {
      this.data.sharedUrls[objectType][objectId] = {
        object_type: objectType,
        object_id: objectId
      };
    }

    this.trigger(this.data);
  },

  onSubscribeBillingFailed: function (error) {
    this.data.shareUrl.isSubscriptionProcessing = false;
    this.data.shareUrl.isSubscriptionProcessed = false;
    this.data.shareUrl.error = this.getError(error);
    this.trigger(this.data);
  },

  onSubscribeBillingProgressed: function () {
    this.data.billing.isSubscriptionProcessing = true;
    this.data.billing.isSubscriptionProcessed = false;
    this.data.billing.error = null;
    this.trigger(this.data);
  },

  onSubscribeBillingCompleted: function (orgId, paymentMethodId, data) {
    this.data.billing.isSubscriptionProcessing = false;
    this.data.billing.isSubscriptionProcessed = true;
    this.data.billing.subscriptionErrorMessage = this.getError(data);
    this.data.billing.subscriptionError = !!this.data.billing.subscriptionErrorMessage;

    if (!this.data.billing.subscriptionError && data.result && data.result !== 'fail') {
      Actions.getUserProfile(this.data.auth.token);
      setTimeout(() => {
        Actions.getUserProfile(this.data.auth.token);
      }, 5000);
    } 
    this.trigger(this.data);
  },


  onGetBillingDataUsageFailed: function (error) {
    this.data.billing.isProcessing = false;
    this.data.billing.isProcessed = false;
    this.data.billing.error = this.getError(error);
    this.trigger(this.data);
  },

  onGetBillingDataUsageProgressed: function (orgId) {
    this.data.billing.isProcessing = true;
    this.data.billing.isProcessed = false;
    this.data.billing.error = null;
    this.data.billing.orgId = orgId;
    this.trigger(this.data);
  },

  onGetBillingDataUsageCompleted: function (orgId, data) {
    this.data.billing.isProcessing = false;
    this.data.billing.isProcessed = true;
    this.data.billing.errorMessage = this.getError(data);
    this.data.billing.error = !!this.data.billing.errorMessage;
    this.data.billing.orgId = orgId;

    if (!this.data.billing.error && data.length === 1) {
      this.data.billing.data = data[0];
    }

    this.trigger(this.data);
  },

  onSetSubscriptionError: function (message) {
    this.data.billing.subscriptionErrorMessage = message;
    this.data.billing.subscriptionError = !!this.data.billing.subscriptionErrorMessage;

    this.trigger(this.data);
  },


  onGetJoeInstanceFailed: function (orgId, projectId, instanceId, error) {
    const instance = this.getJoeInstance(instanceId);
    instance.isProcessing = false;
    instance.isProcessed = true;
    instance.channelsError = true;
    instance.channelsErrorMessage = this.getError(error);
    this.trigger(this.data);
  },

  onGetJoeInstanceProgressed: function (orgId, projectId, instanceId) {
    const instance = this.getJoeInstance(instanceId);
    instance.isProcessing = true;
    instance.isProcessed = false;
    instance.errorMessage = null;
    instance.error = false;
    this.trigger(this.data);
  },

  onGetJoeInstanceCompleted: function (orgId, projectId, instanceId, data) {
    const instance = this.getJoeInstance(instanceId);
    instance.isProcessing = false;
    instance.errorMessage = this.getError(data);
    instance.error = !!instance.errorMessage;

    if (!instance.error) {
      if (data.length) {
        instance.data = data[0];
        instance.isProcessed = true;
      } else {
        instance.error = true;
        instance.errorMessage = 'Specified instance not found or you have no access.';
        instance.errorCode = 404;
      }
    }

    instance.isChannelsProcessed = false;

    this.trigger(this.data);
  },


  onGetDbLabInstanceFailed: function (orgId, projectId, instanceId, error) {
    this.data.dbLabInstance.isProcessing = false;
    this.data.dbLabInstance.isProcessed = true;
    this.data.dbLabInstance.error = true;
    this.data.dbLabInstance.errorMessage = this.getError(error);

    this.trigger(this.data);
  },

  onGetDbLabInstanceProgressed: function () {
    this.data.dbLabInstance.isProcessing = true;
    this.data.dbLabInstance.isProcessed = false;
    this.data.dbLabInstance.error = false;
    this.data.dbLabInstanceStatus.isProcessing = false;
    this.data.dbLabInstanceStatus.isProcessed = false;
    this.data.dbLabInstanceStatus.error = false;

    this.trigger(this.data);
  },

  onGetDbLabInstanceCompleted: function (orgId, projectId, instanceId, data) {
    this.data.dbLabInstance.isProcessing = false;
    this.data.dbLabInstance.errorMessage = this.getError(data);
    this.data.dbLabInstance.error = !!this.data.dbLabInstance.errorMessage;

    if (!this.data.dbLabInstance.error) {
      if (data.length) {
        this.data.dbLabInstance.data = data[0];
        this.data.dbLabInstance.isProcessed = true;
      } else {
        this.data.dbLabInstance.error = true;
        this.data.dbLabInstance.errorMessage =
          'Specified instance not found or you have no access.';
        this.data.dbLabInstance.errorCode = 404;
      }
    }

    this.trigger(this.data);
  },


  onGetDbLabSessionsFailed: function (error) {
    this.data.dbLabSessions.isProcessing = false;
    this.data.dbLabSessions.error = true;
    this.data.dbLabSessions.errorMessage = error.message;
    this.trigger(this.data);
  },

  onGetDbLabSessionsProgressed: function (params) {
    if (!params.lastId) {
      this.data.dbLabSessions.data = [];
    }
    this.data.dbLabSessions.isProcessing = true;
    this.data.dbLabSessions.isProcessed = false;
    this.trigger(this.data);
  },

  onGetDbLabSessionsCompleted: function (params, data, count) {
    this.data.dbLabSessions.isProcessing = false;
    this.data.dbLabSessions.errorMessage = this.getError(data);
    this.data.dbLabSessions.error = !!this.data.dbLabSessions.errorMessage;
    this.data.dbLabSessions.isProcessed = true;

    if (!this.data.dbLabSessions.error) {
      if (params.lastId && this.data.dbLabSessions.data && params.limit) {
        this.data.dbLabSessions.data = [...this.data.dbLabSessions.data, ...data];
      } else {
        this.data.dbLabSessions.data = data;
      }

      if (!isNaN(count) && typeof count !== 'undefined') {
        this.data.dbLabSessions.isComplete = data.length >= count;
      }

      this.data.dbLabSessions.isProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetDbLabSessionFailed: function (sessionId, error) {
    this.data.dbLabSession.isProcessing = false;
    this.data.dbLabSession.isProcessed = true;
    this.data.dbLabSession.error = true;
    this.data.dbLabSession.errorMessage = this.getError(error);

    this.trigger(this.data);
  },

  onGetDbLabSessionProgressed: function () {
    this.data.dbLabSession.isProcessing = true;
    this.data.dbLabSession.isProcessed = false;
    this.data.dbLabSession.error = false;
    this.data.dbLabSession.artifacts = null;
    this.data.dbLabSession.artifactData = null;
    this.data.dbLabSession.logs = null;

    this.data.dbLabSession.isLogsProcessing = false;
    this.data.dbLabSession.isLogsProcessed = false;
    this.data.dbLabSession.logsError = false;
    this.data.dbLabSession.logsErrorMessage = false;

    this.trigger(this.data);
  },

  onGetDbLabSessionCompleted: function (sessionId, data) {
    this.data.dbLabSession.isProcessing = false;
    this.data.dbLabSession.errorMessage = this.getError(data);
    this.data.dbLabSession.error = !!this.data.dbLabSession.errorMessage;

    if (!this.data.dbLabSession.error) {
      if (data.length) {
        this.data.dbLabSession.data = data[0];
        this.data.dbLabSession.isProcessed = true;
      } else {
        this.data.dbLabSession.error = true;
        this.data.dbLabSession.errorMessage =
          'Specified session not found or you have no access.';
        this.data.dbLabSession.errorCode = 404;
      }
    }

    this.trigger(this.data);
  },


  onGetDbLabSessionLogsFailed: function (params, error) {
    this.data.dbLabSession.isLogsProcessing = false;
    this.data.dbLabSession.isLogsProcessed = true;
    this.data.dbLabSession.logsError = true;
    this.data.dbLabSession.logsErrorMessage = this.getError(error);

    this.trigger(this.data);
  },

  onGetDbLabSessionLogsProgressed: function (params) {
    if (!params.lastId) {
      this.data.dbLabSessions.data = [];
    }

    this.data.dbLabSession.isLogsProcessing = true;
    this.data.dbLabSession.isLogsProcessed = false;
    this.data.dbLabSession.error = false;

    this.trigger(this.data);
  },

  onGetDbLabSessionLogsCompleted: function (params, data, count) {
    this.data.dbLabSession.isLogsProcessing = false;
    this.data.dbLabSession.logsErrorMessage = this.getError(data);
    this.data.dbLabSession.logsError = !!this.data.dbLabSession.errorMessage;

    if (!this.data.dbLabSession.logsError) {
      if (params.lastId && this.data.dbLabSession.logs && params.limit) {
        this.data.dbLabSession.logs = [...this.data.dbLabSession.logs, ...data];
      } else {
        this.data.dbLabSession.logs = data;
      }

      if (!isNaN(count) && typeof count !== 'undefined') {
        this.data.dbLabSession.isLogsComplete = data.length >= count;
      }

      this.data.dbLabSession.isLogsProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetDbLabSessionArtifactsFailed: function (sessionId, error) {
    this.data.dbLabSession.isArtifactsProcessing = false;
    this.data.dbLabSession.isArtifactsProcessed = true;
    this.data.dbLabSession.artifactsError = true;
    this.data.dbLabSession.artifactsErrorMessage = this.getError(error);

    this.trigger(this.data);
  },

  onGetDbLabSessionArtifactsProgressed: function (sessionId) {
    if (!sessionId) {
      this.data.dbLabSessions.data = [];
    }

    this.data.dbLabSession.isArtifactsProcessing = true;
    this.data.dbLabSession.isArtifactsProcessed = false;
    this.data.dbLabSession.error = false;

    this.trigger(this.data);
  },

  onGetDbLabSessionArtifactsCompleted: function (sessionId, data) {
    this.data.dbLabSession.isArtifactsProcessing = false;
    this.data.dbLabSession.artifactsErrorMessage = this.getError(data);
    this.data.dbLabSession.artifactsError =
      !!this.data.dbLabSession.artifactsErrorMessage;

    if (!this.data.dbLabSession.artifactsError) {
      this.data.dbLabSession.artifacts = data;

      this.data.dbLabSession.isArtifactsProcessed = true;
    }

    this.trigger(this.data);
  },


  onGetDbLabSessionArtifactFailed: function (sessionId, artifactType, error) {
    if (!this.data.dbLabSession.artifactData) {
      this.data.dbLabSession.artifactData = {};
    }

    if (!this.data.dbLabSession.artifactData[artifactType]) {
      this.data.dbLabSession.artifactData[artifactType] = {};
    }

    this.data.dbLabSession.artifactData[artifactType].isProcessing = false;
    this.data.dbLabSession.artifactData[artifactType].isProcessed = true;
    this.data.dbLabSession.artifactData[artifactType].error = true;
    this.data.dbLabSession.artifactData[artifactType].errorMessage = this.getError(error);

    this.trigger(this.data);
  },

  onGetDbLabSessionArtifactProgressed: function (sessionId, artifactType) {
    if (!this.data.dbLabSession.artifactData) {
      this.data.dbLabSession.artifactData = {};
    }

    if (!this.data.dbLabSession.artifactData[artifactType]) {
      this.data.dbLabSession.artifactData[artifactType] = {};
    }

    this.data.dbLabSession.artifactData[artifactType].isProcessing = true;
    this.data.dbLabSession.artifactData[artifactType].isProcessed = false;
    this.data.dbLabSession.artifactData[artifactType].error = false;
    this.data.dbLabSession.artifactData[artifactType].errorMessage =
      this.getError(this.data.dbLabSession.artifactData[artifactType].error);

    this.trigger(this.data);
  },

  onGetDbLabSessionArtifactCompleted: function (sessionId, artifactType, data) {
    if (!this.data.dbLabSession.artifactData) {
      this.data.dbLabSession.artifactData = {};
    }

    if (!this.data.dbLabSession.artifactData[artifactType]) {
      this.data.dbLabSession.artifactData[artifactType] = {};
    }

    this.data.dbLabSession.artifactData[artifactType].isProcessing = false;
    this.data.dbLabSession.artifactData[artifactType].isProcessed = true;
    this.data.dbLabSession.artifactData[artifactType].errorMessage = this.getError(data);
    this.data.dbLabSession.artifactData[artifactType].error =
      !!this.data.dbLabSession.artifactData[artifactType].errorMessage;

    if (!this.data.dbLabSession.artifactData[artifactType].error && data.length) {
      this.data.dbLabSession.artifactData[artifactType].data =
        JSON.stringify(data[0].artifact_data);
    }

    console.log(this.data.dbLabSession.artifactData[artifactType]);

    this.trigger(this.data);
  },


  onUpdateOrgUserFailed: function (orgId, userId, role, error) {
    this.data.orgUsers.isUpdating = false;
    this.data.orgUsers.updateError = true;
    this.data.orgUsers.updateErrorMessage = error.message;
    this.data.orgUsers.updateUserId = null;
    this.trigger(this.data);
  },

  onUpdateOrgUserProgressed: function (orgId, userId) {
    this.data.orgUsers.updateErrorFields = null;
    this.data.orgUsers.isUpdating = true;
    this.data.orgUsers.updateUserId = userId;

    this.trigger(this.data);
  },

  onUpdateOrgUserCompleted: function (orgId, userId, role, data) {
    this.data.orgUsers.isUpdating = false;
    this.data.orgUsers.updateErrorMessage = this.getError(data);
    this.data.orgUsers.updateError = !!this.data.orgUsers.updateErrorMessage;

    if (!this.data.orgUsers.updateError) {
      Actions.showNotification('User role successfully updated.', 'success');
    } else {
      Actions.showNotification(this.data.orgUsers.updateErrorMessage, 'error');
    }

    this.data.orgUsers.isUpdated = this.data.orgUsers.updateUserId;
    this.data.orgUsers.updateUserId = null;

    this.trigger(this.data);
  },

  onDeleteOrgUserFailed: function (orgId, userId, error) {
    this.data.orgUsers.isDeleting = false;
    this.data.orgUsers.deleteError = true;
    this.data.orgUsers.deleteErrorMessage = error.message;
    this.data.orgUsers.deleteUserId = null;
    this.trigger(this.data);
  },

  onDeleteOrgUserProgressed: function (orgId, userId) {
    this.data.orgUsers.deleteErrorFields = null;
    this.data.orgUsers.isDeleting = true;
    this.data.orgUsers.deleteUserId = userId;

    this.trigger(this.data);
  },

  onDeleteOrgUserCompleted: function (orgId, userId, data) {
    this.data.orgUsers.isDeleting = false;
    this.data.orgUsers.deleteErrorMessage = this.getError(data);
    this.data.orgUsers.deleteError = !!this.data.orgUsers.deleteErrorMessage;

    if (!this.data.orgUsers.deleteError) {
      Actions.showNotification('User successfully deleted from the organization.', 'success');
    } else {
      Actions.showNotification(this.data.orgUsers.deleteErrorMessage, 'error');
    }

    this.data.orgUsers.isDeleted = this.data.orgUsers.deleteUserId;
    this.data.orgUsers.deleteUserId = null;

    if (!this.data.orgUsers.deleteError && this.data.userProfile.data.info.id === userId) {
      window.location = process.env.PUBLIC_URL + '/';
    }

    this.trigger(this.data);
  },


  onGetAuditLogFailed: function (params, error) {
    this.data.auditLog.isProcessing = false;
    this.data.auditLog.isProcessed = true;
    this.data.auditLog.error = true;
    this.data.auditLog.errorMessage = this.getError(error);

    this.trigger(this.data);
  },

  onGetAuditLogProgressed: function (params) {
    if (!params.lastId) {
      this.data.auditLog.data = null;
    }

    this.data.auditLog.isProcessing = true;
    this.data.auditLog.isProcessed = false;
    this.data.auditLog.error = false;

    this.trigger(this.data);
  },

  onGetAuditLogCompleted: function (params, data, count) {
    this.data.auditLog.isProcessing = false;
    this.data.auditLog.errorMessage = this.getError(data);
    this.data.auditLog.error = !!this.data.auditLog.errorMessage;

    this.data.auditLog.orgId = params.orgId;
    if (!this.data.auditLog.error) {
      if (params.lastId && this.data.auditLog.data && params.limit) {
        this.data.auditLog.data = [...this.data.auditLog.data, ...data];
      } else {
        this.data.auditLog.data = data;
      }

      if (!isNaN(count) && typeof count !== 'undefined') {
        this.data.auditLog.isComplete = data.length >= count;
      }

      this.data.auditLog.isLogsProcessed = true;
    }

    this.trigger(this.data);
  },

  onDownloadDblabSessionLogFailed: function (error) {
    if (error.details) {
      Actions.showNotification(error.details, 'error');
    } else {
      Actions.showNotification('Unknown error occurred', 'error');
    }
    this.data.dbLabSession.isLogDownloading = false;
    this.trigger(this.data);
  },

  onDownloadDblabSessionLogProgressed: function () {
    this.data.dbLabSession.isLogDownloading = true;
    this.trigger(this.data);
  },

  onDownloadDblabSessionLogCompleted: function () {
    this.data.dbLabSession.isLogDownloading = false;
    this.trigger(this.data);
  },


  onDownloadDblabSessionArtifactFailed: function (error) {
    if (error.details) {
      Actions.showNotification(error.details, 'error');
    } else {
      Actions.showNotification('Unknown error occurred', 'error');
    }
    this.data.dbLabSession.isArtifactDownloading = false;
    this.data.dbLabSession.downloadingArtifactType = null;
    this.trigger(this.data);
  },

  onDownloadDblabSessionArtifactProgressed: function (params) {
    this.data.dbLabSession.isArtifactDownloading = true;
    this.data.dbLabSession.downloadingArtifactType = params.artifactType;
    this.trigger(this.data);
  },

  onDownloadDblabSessionArtifactCompleted: function () {
    this.data.dbLabSession.isArtifactDownloading = false;
    this.data.dbLabSession.downloadingArtifactType = null;
    this.trigger(this.data);
  },


  onConfirmUserEmailFailed: function (error) {
    this.data.userProfile.isConfirmProcessing = false;
    this.data.userProfile.confirmError = true;
    this.data.userProfile.confirmErrorMessage = error.message;
    this.trigger(this.data);
  },

  onConfirmUserEmailProgressed: function () {
    this.data.userProfile.isConfirmProcessing = true;
    this.data.userProfile.isConfirmProcessed = false;
    this.trigger(this.data);
  },

  onConfirmUserEmailCompleted: function (data) {
    this.data.userProfile.isConfirmProcessing = false;
    this.data.userProfile.confirmErrorMessage = this.getError(data);
    this.data.userProfile.confirmError = !!this.data.userProfile.confirmErrorMessage;
    this.data.userProfile.isConfirmProcessed = true;

    if (Array.isArray(data)) {
      this.onGetUserProfileCompleted(data);

      Actions.showNotification('Email successfully confirmed', 'success');
    } else {
      Actions.showNotification(
        'Email not confirmed. Check verification code or resend code.',
        'error',
        10000
      );
    }

    this.trigger(this.data);
  },

  onSendUserCodeCompleted: function (data) {
    if (data && data.result === 'ok') {
      Actions.showNotification('A verification code was sent.', 'success', 10000);
    } else {
      Actions.showNotification('Error happened on sending code.', 'error', 10000);
    }
  },

  onConfirmTosAgreementFailed: function (error) {
    this.data.userProfile.isTosAgreementConfirmProcessing = false;
    this.data.userProfile.tosConfirmError = true;
    this.data.userProfile.tosConfirmErrorMessage = error.message;
    this.trigger(this.data);
  },

  onConfirmTosAgreementProgressed: function () {
    this.data.userProfile.isTosAgreementConfirmProcessing = true;
    this.data.userProfile.isTosAgreementConfirmProcessed = false;
    this.trigger(this.data);
  },

  onConfirmTosAgreementCompleted: function (data) {
    this.data.userProfile.isTosAgreementConfirmProcessing = false;
    this.data.userProfile.tosConfirmErrorMessage = this.getError(data);
    this.data.userProfile.tosConfirmError = !!this.data.userProfile.tosConfirmErrorMessage;
    this.data.userProfile.isTosAgreementConfirmProcessed = true;

    if (Array.isArray(data)) {
      this.onGetUserProfileCompleted(data);

      Actions.showNotification('Agreement successfully confirmed', 'success', 5000);
    } else {
      Actions.showNotification('Agreement not confirmed', 'error', 10000);
    }

    this.trigger(this.data);
  }
});

export default Store;
