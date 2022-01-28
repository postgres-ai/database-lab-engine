/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { snakeToCamel } from '../utils/utils';

const adminRoleId = 3;

export default {

  userIsOwner: function (userId, orgId, data) {
    for (let i in data.orgs) {
      if (data.orgs.hasOwnProperty(i) && data.orgs[i].id === orgId &&
        data.orgs[i].owner_user_id === userId) {
        return true;
      }
    }

    return false;
  },

  isAdmin: function (orgData) {
    return orgData && orgData.role && orgData.role.id === adminRoleId;
  },

  /*
  Permissions defined on the server and converted to camel case from snake case programmatically.

  Available permissions listed for convenience (may be outdated):
    Guest:
    - settings_organization_list
    - settings_token_create_personal
    - settings_token_list_owned
    - settings_token_revoke_owned
    - settings_profile_view
    - settings_projects_view
    - dblab_instance_list
    - dblab_instance_view
    - dblab_clone_create
    - dblab_clone_status
    - dblab_clone_update
    - dblab_clone_reset
    - dblab_clone_destroy
    - dblab_snapshot_list
    - joe_assistant_post_command
    - joe_history_view
    - checkup_report_view

    User:
    - Guest
    - settings_member_list

    Owner: (TODO: Rename to Admin)
    - User
    - settings_organization_create
    - settings_organization_update
    - settings_token_create_impersonal
    - settings_token_list_any
    - settings_token_revoke_any
    - settings_member_add
    - settings_member_delete
    - dblab_instance_create
    - dblab_instance_delete
    - joe_instance_create
    - joe_instance_delete
    - joe_assistant_create_session
    - joe_assistant_post_message
    - joe_assistant_post_artifact
    - joe_history_delete
    - checkup_report_configure
    - checkup_report_delete
  */
  getPermissions: function (orgData) {
    const permissions = {};

    if (!orgData || !orgData.role || !orgData.role.permissions) {
      return permissions;
    }

    orgData.role.permissions.forEach(p => {
      const pCamel = snakeToCamel(p);
      if (!pCamel) {
        return;
      }

      permissions[pCamel] = true;
    });

    return permissions;
  }
};
