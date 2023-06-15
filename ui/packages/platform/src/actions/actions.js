/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import Reflux from 'reflux';

import { localStorage } from 'helpers/localStorage';

import Api from '../api/api';
import ExplainDepeszApi from '../api/explain/depesz';
import ExplainPev2Api from '../api/explain/pev2';
import settings from '../utils/settings';
import { visualizeTypes } from '../assets/visualizeTypes';

let api;
let explainDepeszApi;
let explainPev2Api;

settings.init(() => {
  api = new Api(settings);
  explainDepeszApi = new ExplainDepeszApi(settings);
  explainPev2Api = new ExplainPev2Api(settings);
});

// Timeout: 30 sec.
const REQUEST_TIMEOUT = 30000;
const ASYNC_ACTION = {
  asyncResult: true,
  children: ['progressed', 'completed', 'failed']
};

const Actions = Reflux.createActions([{
  auth: {},
  signOut: {},
  ASYNC_ACTION: ASYNC_ACTION,
  doAuth: ASYNC_ACTION,
  getUserProfile: ASYNC_ACTION,
  getAccessTokens: ASYNC_ACTION,
  getAccessToken: ASYNC_ACTION,
  hideGeneratedAccessToken: {},
  revokeAccessToken: ASYNC_ACTION,
  getCheckupReports: ASYNC_ACTION,
  getCheckupReportFiles: ASYNC_ACTION,
  getCheckupReportFile: ASYNC_ACTION,
  getJoeSessions: ASYNC_ACTION,
  getJoeSessionCommands: ASYNC_ACTION,
  getJoeSessionCommand: ASYNC_ACTION,
  getProjects: ASYNC_ACTION,
  getOrgs: ASYNC_ACTION,
  getOrgUsers: ASYNC_ACTION,
  updateOrg: ASYNC_ACTION,
  createOrg: ASYNC_ACTION,
  inviteUser: ASYNC_ACTION,
  useDemoData: ASYNC_ACTION,
  setReportsProject: {},
  setSessionsProject: {},
  setDbLabInstancesProject: {},
  refresh: {},
  getDbLabInstances: ASYNC_ACTION,
  addDbLabInstance: ASYNC_ACTION,
  editDbLabInstance: ASYNC_ACTION,
  destroyDbLabInstance: ASYNC_ACTION,
  resetNewDbLabInstance: {},
  reloadDblabInstance: ASYNC_ACTION,
  checkDbLabInstanceUrl: ASYNC_ACTION,
  downloadReportJsonFiles: ASYNC_ACTION,
  addOrgDomain: ASYNC_ACTION,
  deleteOrgDomain: ASYNC_ACTION,
  showNotification: {},
  hideNotification: {},
  setJoeInstancesProject: {},
  getJoeInstances: ASYNC_ACTION,
  getJoeInstanceChannels: ASYNC_ACTION,
  sendJoeInstanceCommand: ASYNC_ACTION,
  initJoeWebSocketConnection: {},
  getJoeInstanceMessages: ASYNC_ACTION,
  getJoeMessageArtifacts: ASYNC_ACTION,
  resetNewJoeInstance: {},
  checkJoeInstanceUrl: ASYNC_ACTION,
  addJoeInstance: ASYNC_ACTION,
  destroyJoeInstance: ASYNC_ACTION,
  closeJoeWebSocketConnection: {},
  getExternalVisualizationData: ASYNC_ACTION,
  closeExternalVisualization: {},
  deleteJoeSessions: ASYNC_ACTION,
  deleteJoeCommands: ASYNC_ACTION,
  deleteCheckupReports: ASYNC_ACTION,
  joeCommandFavorite: ASYNC_ACTION,
  clearJoeInstanceChannelMessages: {},
  getSharedUrlData: ASYNC_ACTION,
  getSharedUrl: ASYNC_ACTION,
  addSharedUrl: ASYNC_ACTION,
  removeSharedUrl: ASYNC_ACTION,
  showShareUrlDialog: {},
  closeShareUrlDialog: {},
  getBillingDataUsage: ASYNC_ACTION,
  subscribeBilling: ASYNC_ACTION,
  setSubscriptionError: {},
  getDbLabInstance: ASYNC_ACTION,
  getJoeInstance: ASYNC_ACTION,
  updateOrgUser: ASYNC_ACTION,
  deleteOrgUser: ASYNC_ACTION,
  getDbLabSessions: ASYNC_ACTION,
  getDbLabSession: ASYNC_ACTION,
  getDbLabSessionLogs: ASYNC_ACTION,
  getDbLabSessionArtifacts: ASYNC_ACTION,
  getDbLabSessionArtifact: ASYNC_ACTION,
  getAuditLog: ASYNC_ACTION,
  downloadDblabSessionLog: ASYNC_ACTION,
  downloadDblabSessionArtifact: ASYNC_ACTION,
  sendUserCode: ASYNC_ACTION,
  confirmUserEmail: ASYNC_ACTION,
  confirmTosAgreement: ASYNC_ACTION
}]);

function timeoutPromise(ms, promise) {
  return new Promise((resolve, reject) => {
    const timeoutId = setTimeout(function () {
      reject(new Error('timeout'));
    }, ms);

    promise
      .then(
        (res) => {
          clearTimeout(timeoutId);
          resolve(res);
        },
        (err) => {
          clearTimeout(timeoutId);
          reject(err);
        });
  });
}

function actionResult(promise, cb, errorCb) {
  timeoutPromise(REQUEST_TIMEOUT, promise)
    .then(result => {
      let count;
      try {
        let range = result.headers.get('content-range');
        if (range) {
          range = range.split('/');
          if (Array.isArray(range) && range.length) {
            range = range[range.length - 1];
            count = parseInt(range, 10);
          }
        }
      } catch (e) {
        console.log('Range is empty');
      }

      result.json()
        .then(json => {
          if (!json) {
            if (errorCb) {
              errorCb(new Error('wrong_reply'));
            } else {
              this.failed(new Error('wrong_reply'));
            }
          }

          if (cb) {
            cb(json, count);
          } else {
            this.completed(json, count);
          }
        })
        .catch(err => {
          console.error(err);

          if (errorCb) {
            errorCb(new Error('wrong_reply'));
          } else {
            this.failed(new Error('wrong_reply'));
          }
        });
    })
    .catch(err => {
      console.error(err);
      let actionErr = new Error('wrong_reply');

      if (err && err.message && err.message === 'timeout') {
        actionErr = new Error('failed_fetch');
      }

      if (errorCb) {
        errorCb(new Error(actionErr));
      } else {
        this.failed(actionErr);
      }
    });
}

Actions.doAuth.listen(function (email, password) {
  const action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  if (!email && !password) {
    // Token should be passed through /auth.html page.
    // Example: /auth-gate.html?token=some-token
    const token = localStorage.getAuthToken();

    if (token) {
      action.completed({ token: token });
    } else {
      action.failed(new Error('empty_request'));
    }

    return;
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.login(email, password))
    .then(result => {
      if (result.status !== 200) {
        action.failed(new Error('wrong_reply'));
      }

      result.json()
        .then(json => {
          if (json) {
            if (typeof json === 'object') {
              action.failed(new Error(json.message));
            } else {
              action.completed({ token: json });
            }
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getUserProfile.listen(function (token) {
  this.progressed();

  actionResult.bind(this)(
    api.getUserProfile(token),
    (json) => {
      this.completed(json);
    }
  );
});

Actions.getAccessTokens.listen(function (token, orgId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.getAccessTokens(token, orgId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json, orgId: orgId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getAccessToken.listen(function (token, name, expires, orgId, isPersonal) {
  let requestExpires = expires.split('-').join('') + 'T235959-0330';

  this.progressed();
  actionResult.bind(this)(
    api.getAccessToken(token, name, requestExpires, orgId, isPersonal),
    (json) => {
      this.completed({ orgId: orgId, data: json });
    });
});

Actions.revokeAccessToken.listen(function (token, orgId, id) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed(id);

  timeoutPromise(REQUEST_TIMEOUT, api.revokeAccessToken(token, id))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ orgId: orgId, data: json });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getCheckupReports.listen(function (token, orgId, projectId, reportId) {
  this.progressed();

  actionResult.bind(this)(
    api.getCheckupReports(token, orgId, projectId, reportId),
    (json) => {
      this.completed({
        data: json,
        orgId: orgId,
        projectId: projectId,
        reportId: reportId
      });
    }
  );
});

Actions.getCheckupReportFiles.listen(function (token, reportId, type, orderBy,
  orderDirection) {
  this.progressed();

  actionResult.bind(this)(
    api.getCheckupReportFiles(token, reportId, type, orderBy, orderDirection),
    (json) => {
      this.completed({ reportId: reportId, data: json, type: type });
    }
  );
});

Actions.getCheckupReportFile.listen(function (token, projectId, reportId, fileId, type) {
  this.progressed();

  actionResult.bind(this)(
    api.getCheckupReportFile(token, projectId, reportId, fileId, type),
    (json) => {
      this.completed(fileId, json);
    }
  );
});

Actions.getJoeSessions.listen(function (token, orgId, projectId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.getJoeSessions(token, orgId, projectId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json, orgId: orgId, projectId: projectId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getJoeSessionCommands.listen(function (token, params) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed(params);

  timeoutPromise(REQUEST_TIMEOUT, api.getJoeSessionCommands(token, params))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(json, params);
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getJoeSessionCommand.listen(function (token, orgId, commandId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.getJoeSessionCommand(token, orgId, commandId))

    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            const command = json[0];
            action.completed(command);
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getProjects.listen(function (token, orgId) {
  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  let action = this;

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.getProjects(token, orgId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json, orgId: orgId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getOrgs.listen(function (token, orgId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ orgId });

  timeoutPromise(REQUEST_TIMEOUT, api.getOrgs(token, orgId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json, orgId: orgId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getOrgUsers.listen(function (token, orgId) {
  this.progressed(orgId);

  actionResult.bind(this)(
    api.getOrgUsers(token, orgId),
    (json) => {
      this.completed(orgId, json);
    },
    (err) => {
      this.failed(orgId, err);
    }
  );
});

Actions.updateOrg.listen(function (token, orgId, orgData) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ orgId } + orgData);
  timeoutPromise(REQUEST_TIMEOUT, api.updateOrg(token, orgId, orgData))

    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(json);
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.createOrg.listen(function (token, orgData) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed(orgData);
  timeoutPromise(REQUEST_TIMEOUT, api.createOrg(token, orgData))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(json);
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.inviteUser.listen(function (token, orgId, email) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ orgId, email });
  timeoutPromise(REQUEST_TIMEOUT, api.inviteUser(token, orgId, email))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(json);
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.useDemoData.listen(function (token) {
  this.progressed();

  actionResult.bind(this)(
    api.useDemoData(token),
    (json) => {
      this.completed(json);
    },
    (err) => {
      this.failed(err);
    }
  );
});

Actions.getDbLabInstances.listen(function (token, orgId, projectId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();
  timeoutPromise(REQUEST_TIMEOUT, api.getDbLabInstances(token, orgId, projectId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json, orgId: orgId, projectId: projectId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.addDbLabInstance.listen(function (token, instanceData) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.addDbLabInstance(token, instanceData))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(
              { data: json, orgId: instanceData.orgId, project: instanceData.project });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.editDbLabInstance.listen(function (token, instanceData) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.editDbLabInstance(token, instanceData))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(
              { data: json, orgId: instanceData.orgId, project: instanceData.project });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.destroyDbLabInstance.listen(function (token, instanceId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ instanceId: instanceId });

  timeoutPromise(REQUEST_TIMEOUT, api.destroyDbLabInstance(token, instanceId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json, instanceId: instanceId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.reloadDblabInstance.listen(function (token, instanceId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ instanceId: instanceId });

  timeoutPromise(REQUEST_TIMEOUT, api.getInstance(token, instanceId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ data: json[0], instanceId: instanceId });
          } else {
            action.failed({ instanceId: instanceId }, new Error(
              'wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed({ instanceId: instanceId }, new Error(
            'wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed({ instanceId: instanceId }, new Error(
          'failed_fetch'));
      } else {
        action.failed({ instanceId: instanceId }, new Error(
          'wrong_reply'));
      }
    });
});

Actions.checkDbLabInstanceUrl.listen(function (token, url, verifyToken, useTunnel) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed();

  timeoutPromise(REQUEST_TIMEOUT, api.checkDbLabInstanceUrl(token, url, verifyToken, useTunnel))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed(json);
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.downloadBinaryFile = (action, token, fileUrl, storeParams = {}) => {
  let xhr = new XMLHttpRequest();

  action.progressed(storeParams);
  xhr.open('GET', fileUrl);
  xhr.setRequestHeader('authorization', 'Bearer ' + token);
  xhr.setRequestHeader('accept', 'application/octet-stream');
  xhr.responseType = 'arraybuffer';
  xhr.addEventListener('readystatechange', function () {
    if (this.readyState === 4) {
      let blob;
      let filename = '';
      let type = xhr.getResponseHeader('Content-Type');
      let disposition = xhr.getResponseHeader('Content-Disposition');
      let url = window.URL || window.webkitURL;

      if (disposition && disposition.indexOf('attachment') !== -1) {
        let filenameRegex = /filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/;
        let matches = filenameRegex.exec(disposition);

        if (matches !== null && matches[1]) {
          filename = matches[1].replace(/['"]/g, '');
        }
      }

      if (typeof File === 'function') {
        try {
          blob = new File([this.response], filename, { type: type });
        } catch (e) {
          /* IE 10 or less do not support constructor for File.
          In this case we will use Blob on next step. */
        }
      }

      if (this.status === 404) {
        try {
          const jsonBody = JSON.parse(new TextDecoder().decode(this.response));
          action.failed(jsonBody);
        } catch (e) {
          action.failed({});
        }
        return;
      }

      if (typeof blob === 'undefined') {
        blob = new Blob([this.response], { type: type });
      }

      action.completed();

      if (typeof window.navigator.msSaveBlob !== 'undefined') {
        window.navigator.msSaveBlob(blob, filename);

        return;
      }

      let downloadUrl = url.createObjectURL(blob);

      if (filename) {
        // use HTML5 a[download] attribute to specify filename.
        let a = document.createElement('a');
        // safari doesn't support this yet
        if (typeof a.download === 'undefined') {
          window.location = downloadUrl;
        } else {
          a.href = downloadUrl;
          a.download = filename;
          document.body.appendChild(a);
          a.click();
        }

        return;
      }

      window.location = downloadUrl;
    }
  });

  xhr.send();
};


Actions.downloadReportJsonFiles.listen(function (token, reportId) {
  let url = settings.apiServer +
    '/rpc/checkup_report_json_download?checkup_report_id=' + reportId;
  Actions.downloadBinaryFile(this, token, url);
});

Actions.deleteOrgDomain.listen(function (token, orgId, domainId) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ orgId, domainId });
  timeoutPromise(REQUEST_TIMEOUT, api.deleteOrgDomain(token, domainId))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ orgId, domainId });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.addOrgDomain.listen(function (token, orgId, domain) {
  let action = this;

  if (!api) {
    settings.init(function () {
      api = new Api(settings);
    });
  }

  action.progressed({ orgId, domain });
  timeoutPromise(REQUEST_TIMEOUT, api.addOrgDomain(token, orgId, domain))
    .then(result => {
      result.json()
        .then(json => {
          if (json) {
            action.completed({ orgId, domain });
          } else {
            action.failed(new Error('wrong_reply'));
          }
        })
        .catch(err => {
          console.error(err);
          action.failed(new Error('wrong_reply'));
        });
    })
    .catch(err => {
      console.error(err);
      if (err && err.message && err.message === 'timeout') {
        action.failed(new Error('failed_fetch'));
      } else {
        action.failed(new Error('wrong_reply'));
      }
    });
});

Actions.getJoeInstances.listen(function (token, orgId, projectId) {
  this.progressed();

  actionResult.bind(this)(
    api.getJoeInstances(token, orgId, projectId),
    (json) => {
      this.completed(orgId, projectId, json);
    });
});

Actions.getJoeInstanceChannels.listen(function (token, instanceId) {
  this.progressed(instanceId);
  actionResult.bind(this)(
    api.getJoeInstanceChannels(token, instanceId),
    (json) => {
      this.completed(instanceId, json);
    },
    (err) => {
      this.failed(instanceId, err);
    });
});

Actions.sendJoeInstanceCommand.listen(function (token, instanceId,
  channelId, command, sessionId) {
  this.progressed(instanceId);
  actionResult.bind(this)(
    api.sendJoeInstanceCommand(token, instanceId, channelId, command, sessionId),
    (json) => {
      json.message = command;
      this.completed(instanceId, json);
    },
    (err) => {
      this.failed(instanceId, err);
    }
  );
});

Actions.getJoeInstanceMessages.listen(function (token, instanceId,
  channelId, sessionId) {
  this.progressed(instanceId);
  actionResult.bind(this)(
    api.getJoeInstanceMessages(token, channelId, sessionId),
    (json) => {
      this.completed(instanceId, channelId, json);
    },
    (err) => {
      this.failed(instanceId, channelId, err);
    });
});

Actions.getJoeMessageArtifacts.listen(function (token, instanceId, channelId, messageId) {
  this.progressed(instanceId, channelId, messageId);
  actionResult.bind(this)(
    api.getJoeMessageArtifacts(token, messageId),
    (json) => {
      this.completed(instanceId, channelId, messageId, json);
    },
    (err) => {
      this.failed(instanceId, channelId, messageId, err);
    }
  );
});

Actions.checkJoeInstanceUrl.listen(function (token, instanceData) {
  this.progressed();

  instanceData.dryRun = true;

  actionResult.bind(this)(api.addJoeInstance(token, instanceData));
});

Actions.addJoeInstance.listen(function (token, instanceData) {
  this.progressed();

  instanceData.dryRun = false;

  actionResult.bind(this)(
    api.addJoeInstance(token, instanceData),
    (json) => {
      this.completed(instanceData.orgId, json);
    }
  );
});

Actions.destroyJoeInstance.listen(function (token, instanceId) {
  this.progressed(instanceId);

  actionResult.bind(this)(
    api.destroyJoeInstance(token, instanceId),
    (json) => {
      this.completed(instanceId, json);
    }
  );
});

Actions.getExternalVisualizationData.listen(function (type, plan, query) {
  if (type !== visualizeTypes.depesz && type !== visualizeTypes.pev2) {
    console.log('Unsupported visualization type.');
    return;
  }

  this.progressed(type, plan, query);

  if (type === visualizeTypes.pev2) {
    explainPev2Api.postPlan(plan.json, query)
      .then((result) => {
        result.json()
          .then((response) => {
            console.log(response.json);

            if (!response || !response.id) {
              this.failed(type);
              return;
            }

            const url = `${settings.explainPev2Server}#${response.id}`;
            this.completed(type, plan, query, url);
          }).catch((error) => {
            console.error('Error:', error);
            this.failed(type);
          });
      }).catch((error) => {
        console.error('Error:', error);
        this.failed(type);
      });
  } else {
    explainDepeszApi.postPlan(plan.text, query)
      .then((response) => {
        console.log(response);

        const xFinalUrl = response.headers.get('x-final-url');

        let url = response.url;
        if (response.url === settings.explainDepeszServer && !!xFinalUrl) {
          url = xFinalUrl;
        }

        this.completed(type, plan, query, url);
      }).catch((error) => {
        console.error('Error:', error);
        this.failed(type);
      });
  }
});

Actions.deleteJoeSessions.listen(function (token, ids) {
  this.progressed(ids);

  actionResult.bind(this)(
    api.deleteJoeSessions(token, ids),
    (json) => {
      this.completed(ids, json);
    }
  );
});

Actions.deleteJoeCommands.listen(function (token, ids) {
  this.progressed(ids);

  actionResult.bind(this)(
    api.deleteJoeCommands(token, ids),
    (json) => {
      this.completed(ids, json);
    }
  );
});


Actions.deleteCheckupReports.listen(function (token, ids) {
  this.progressed(ids);

  actionResult.bind(this)(
    api.deleteCheckupReports(token, ids),
    (json) => {
      this.completed(ids, json);
    }
  );
});

Actions.joeCommandFavorite.listen(function (token, commandId, favorite) {
  this.progressed(commandId, favorite);

  actionResult.bind(this)(
    api.joeCommandFavorite(token, commandId, favorite),
    (json) => {
      this.completed(json, commandId, favorite);
    }
  );
});

Actions.getSharedUrlData.listen(function (uuid) {
  this.progressed(uuid);

  actionResult.bind(this)(
    api.getSharedUrlData(uuid),
    (json) => {
      this.completed(uuid, json);
    }
  );
});

Actions.getSharedUrl.listen(function (token, orgId, objectType, objectId) {
  this.progressed(orgId, objectType, objectId);

  actionResult.bind(this)(
    api.getSharedUrl(token, orgId, objectType, objectId),
    (json) => {
      this.completed(orgId, objectType, objectId, json);
    }
  );
});

Actions.removeSharedUrl.listen(function (token, orgId, objectType, objectId, urlId) {
  this.progressed();

  actionResult.bind(this)(
    api.removeSharedUrl(token, orgId, urlId),
    (json) => {
      this.completed(orgId, objectType, objectId, urlId, json);
    }
  );
});

Actions.addSharedUrl.listen(function (token, params) {
  this.progressed(params);

  actionResult.bind(this)(
    api.addSharedUrl(token, params),
    (json) => {
      this.completed(params, json);
    }
  );
});

Actions.getBillingDataUsage.listen(function (token, orgId) {
  this.progressed(orgId);

  actionResult.bind(this)(
    api.getBillingDataUsage(token, orgId),
    (json) => {
      this.completed(orgId, json);
    }
  );
});

Actions.subscribeBilling.listen(function (token, orgId, paymentMethodId) {
  this.progressed(orgId, paymentMethodId);

  actionResult.bind(this)(
    api.subscribeBilling(token, orgId, paymentMethodId),
    (json) => {
      this.completed(orgId, paymentMethodId, json);
    }
  );
});

Actions.getDbLabInstance.listen(function (token, orgId, projectId, instanceId) {
  this.progressed(orgId, projectId, instanceId);

  actionResult.bind(this)(
    api.getDbLabInstances(token, orgId, projectId, instanceId),
    (json) => {
      this.completed(orgId, projectId, instanceId, json);
    },
    (err) => {
      this.failed(orgId, projectId, instanceId, err);
    }
  );
});

Actions.getJoeInstance.listen(function (token, orgId, projectId, instanceId) {
  this.progressed(orgId, projectId, instanceId);

  actionResult.bind(this)(
    api.getJoeInstances(token, orgId, projectId, instanceId),
    (json) => {
      this.completed(orgId, projectId, instanceId, json);
    },
    (err) => {
      this.failed(orgId, projectId, instanceId, err);
    }
  );
});

Actions.getDbLabSessions.listen(function (token, params) {
  this.progressed(params);

  actionResult.bind(this)(
    api.getDbLabSessions(token, params),
    (json, count) => {
      this.completed(params, json, count);
    },
    (err) => {
      this.failed(params, err);
    }
  );
});

Actions.getDbLabSession.listen(function (token, sessionId) {
  this.progressed(sessionId);

  actionResult.bind(this)(
    api.getDbLabSession(token, sessionId),
    (json) => {
      this.completed(sessionId, json);
    },
    (err) => {
      this.failed(sessionId, err);
    }
  );
});

Actions.getDbLabSessionLogs.listen(function (token, params) {
  this.progressed(params);

  actionResult.bind(this)(
    api.getDbLabSessionLogs(token, params),
    (json, count) => {
      this.completed(params, json, count);
    },
    (err) => {
      this.failed(params, err);
    }
  );
});

Actions.getDbLabSessionArtifacts.listen(function (token, sessionId) {
  this.progressed(sessionId);

  actionResult.bind(this)(
    api.getDbLabSessionArtifacts(token, sessionId),
    (json) => {
      this.completed(sessionId, json);
    },
    (err) => {
      this.failed(sessionId, err);
    }
  );
});

Actions.getDbLabSessionArtifact.listen(function (token, sessionId, artifactType) {
  this.progressed(sessionId, artifactType);

  actionResult.bind(this)(
    api.getDbLabSessionArtifact(token, sessionId, artifactType),
    (json) => {
      this.completed(sessionId, artifactType, json);
    },
    (err) => {
      this.failed(sessionId, artifactType, err);
    }
  );
});


Actions.updateOrgUser.listen(function (token, orgId, userId, role) {
  this.progressed(orgId, userId, role);

  actionResult.bind(this)(
    api.updateOrgUser(token, orgId, userId, role),
    (json) => {
      this.completed(orgId, userId, role, json);
    },
    (err) => {
      this.failed(orgId, userId, role, err);
    }
  );
});

Actions.deleteOrgUser.listen(function (token, orgId, userId) {
  this.progressed(orgId, userId);

  actionResult.bind(this)(
    api.deleteOrgUser(token, orgId, userId),
    (json) => {
      this.completed(orgId, userId, json);
    },
    (err) => {
      this.failed(orgId, userId, err);
    }
  );
});

Actions.getAuditLog.listen(function (token, params) {
  this.progressed(params);

  actionResult.bind(this)(
    api.getAuditLog(token, params),
    (json, count) => {
      this.completed(params, json, count);
    },
    (err) => {
      this.failed(params, err);
    }
  );
});

Actions.downloadDblabSessionLog.listen(function (token, sessionId) {
  let url = settings.apiServer +
    '/rpc/dblab_session_logs_download?dblab_session_id=' + sessionId;
  Actions.downloadBinaryFile(this, token, url);
});


Actions.downloadDblabSessionArtifact.listen(function (token, sessionId, artifactType) {
  let url = settings.apiServer +
    '/rpc/dblab_session_artifacts_download?' +
      'dblab_session_id=' + sessionId +
      '&artifact_type=' + artifactType;
  Actions.downloadBinaryFile(this, token, url, { artifactType });
});

Actions.sendUserCode.listen(function (token) {
  this.progressed();

  actionResult.bind(this)(
    api.sendUserCode(token),
    (json, count) => {
      this.completed(json, count);
    },
    (err) => {
      this.failed(err);
    }
  );
});

Actions.confirmUserEmail.listen(function (token, code) {
  this.progressed();

  actionResult.bind(this)(
    api.confirmUserEmail(token, code),
    (json) => {
      this.completed(json);
    },
    (err) => {
      this.failed(err);
    }
  );
});

Actions.confirmTosAgreement.listen(function (token) {
  this.progressed();

  actionResult.bind(this)(
    api.confirmTosAgreement(token),
    (json) => {
      this.completed(json);
    },
    (err) => {
      this.failed(err);
    }
  );
});

export default Actions;
