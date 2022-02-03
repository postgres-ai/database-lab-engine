/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import {
  Table, TableBody, TableCell, TableHead, TableRow,
  IconButton
} from '@material-ui/core';
import DeleteIcon from '@material-ui/icons/Delete';
import Select from '@material-ui/core/Select';
import MenuItem from '@material-ui/core/MenuItem';
import ExitIcon from '@material-ui/icons/ExitToApp';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { styles } from '@postgres.ai/shared/styles/styles';

import Actions from '../actions/actions';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsoleButton from './ConsoleButton';
import ConsolePageTitle from './ConsolePageTitle';
import Error from './Error';
import Store from '../stores/store';
import Warning from './Warning';
import messages from '../assets/messages';


const getStyles = theme => ({
  root: {
    ...styles.root,
    display: 'flex',
    flexDirection: 'column',
    paddingBottom: '20px'
  },
  container: {
    display: 'flex',
    flexWrap: 'wrap'
  },
  textField: {
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
    width: '80%'
  },
  actionCell: {
    textAlign: 'right',
    padding: 0,
    paddingRight: 16
  },
  iconButton: {
    margin: '-12px',
    marginLeft: 5
  },
  inTableProgress: {
    width: '15px!important',
    height: '15px!important',
    marginLeft: 5,
    verticalAlign: 'middle'
  },
  roleSelector: {
    'height': 24,
    'width': 190,
    '& svg': {
      top: 5,
      right: 3
    },
    '& .MuiSelect-select': {
      padding: 8,
      paddingRight: 20
    }

  },
  roleSelectorItem: {
    fontSize: 14
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

const membersTitle = 'Members';

class OrgSettings extends Component {
  componentDidMount() {
    const that = this;
    const { orgId, orgPermissions } = this.props;
    const hasListMembersPermission = !orgPermissions || orgPermissions.settingsMemberList;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const orgUsers = this.data && this.data.orgUsers ? this.data.orgUsers :
        null;

      that.setState({ data: this.data });

      if (!hasListMembersPermission) {
        return;
      }

      if (orgUsers.isRefreshed) {
        that.setState({
          changes: {}
        });
      }

      if (orgUsers.isUpdated || orgUsers.isDeleted) {
        // Updated users list after deleting or updating.
        Actions.getOrgUsers(auth.token, orgId);
      }

      if (auth && auth.token && orgUsers &&
        !orgUsers.isProcessing && !orgUsers.error && !that.state) {
        // Initial loading of users.
        Actions.getOrgUsers(auth.token, orgId);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  buttonAddHandler = () => {
    const org = this.props.org ? this.props.org : null;

    this.props.history.push('/' + org + '/members/add');
  }

  deleteHandler = (event, u) => {
    const { orgId, env } = this.props;
    /* eslint no-alert: 0 */
    let userName = [];
    if (u.first_name) {
      userName.push(u.first_name);
    }
    if (u.last_name) {
      userName.push(u.last_name);
    }
    userName = userName.join(' ');
    if (!userName) {
      userName = 'with email ' + u.email;
    } else {
      userName = userName + ' (' + u.email + ')';
    }
    let message = 'Are you sure you want to remove ' +
      userName + ' from the organization?';

    if (u.id === env.data.info.id) {
      message = 'Are you sure you want to leave the organization?';
    }


    if (window.confirm(message) === true) {
      Actions.deleteOrgUser(this.state.data.auth.token, orgId, u.id);
    }
  }

  changeRoleHandler = (userId, userRoleId) => {
    const { orgId } = this.props;

    Actions.updateOrgUser(this.state.data.auth.token, orgId,
      userId, userRoleId);

    this.setState({
      changes: {
        [userId]: userRoleId
      }
    });
  }

  roleSelector(u) {
    const that = this;
    const { classes, orgPermissions } = this.props;
    const roles = this.state.data && this.state.data.orgUsers &&
      this.state.data.orgUsers.data && this.state.data.orgUsers.data.roles ?
      this.state.data.orgUsers.data.roles : [];

    if (orgPermissions.settingsMemberUpdate && !u.is_owner) {
      return (
        <span>
          <Select
            value={that.state && that.state.changes && that.state.changes[u.id] ?
              that.state.changes[u.id] : u.role_id}
            onChange={(event) => {
              if (u.role_id !== event.target.value) {
                that.changeRoleHandler(u.id, event.target.value);
              }
            }}
            className={classes.roleSelector}
            variant='outlined'
          >
            {roles.map(r => {
              return (
                <MenuItem value={r.id} className={classes.roleSelectorItem}>
                  {r.name}
                </MenuItem>
              );
            })}
          </Select>
        </span>
      );
    }

    // Just output role name.
    let role = '-';
    for (let i in roles) {
      if (roles[i].id === u.role_id) {
        role = roles[i].name;
      }
    }

    return (
      <span>{role}{u.is_owner ? ' (Owner)' : ''}</span>
    );
  }

  render() {
    const { classes, orgPermissions, orgId, env } = this.props;
    const data = this.state && this.state.data ? this.state.data.orgUsers : null;
    const userProfile = this.state && this.state.data ? this.state.data.userProfile : null;

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: membersTitle }]}
      />
    );

    const hasAddMemberPermission = !orgPermissions || orgPermissions.settingsMemberAdd;
    const hasListMembersPermission = !orgPermissions || orgPermissions.settingsMemberList;

    const actions = [(
      <ConsoleButton
        disabled={ !hasAddMemberPermission }
        title={ hasAddMemberPermission ? 'Add a new member' : messages.noPermission }
        variant='contained'
        color='primary'
        onClick={ this.buttonAddHandler }
      >
        Add
      </ConsoleButton>
    )];

    const pageTitle = (
      <ConsolePageTitle
        title={ membersTitle }
        actions={ actions }
      />
    );

    if (!data || (hasListMembersPermission &&
      (data.orgId !== orgId) ||
      (data.isProcessing && data.isRefresh === false))) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <PageSpinner />
        </div>
      );
    }

    if (data.error) {
      return (
        <div className={classes.root}>
          <Error
            message={data.errorMessage}
            code={data.errorCode}
          />
        </div>
      );
    }

    // If user does not have "ListMembersPermission" we will fill the list only
    // with his data without making getOrgUsers request.
    let users = [];
    if (hasListMembersPermission) {
      if (data && data.data && data.data.users && data.data.users.length > 0) {
        users = data.data.users;
      }
    } else if (userProfile && userProfile.data && userProfile.data.info) {
      users = [userProfile.data.info];
    }

    return (
      <div className={classes.root}>
        { breadcrumbs }

        { pageTitle }

        { !hasListMembersPermission &&
          <Warning>You do not have permission to view the full list of members</Warning>
        }

        { users.length > 0 ? (
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell>Email</TableCell>
                  <TableCell>Role</TableCell>
                  <TableCell>First name</TableCell>
                  <TableCell>Last name</TableCell>
                  <TableCell/>
                </TableRow>
              </TableHead>
              <TableBody>
                {users.map(u => {
                  return (
                    <TableRow
                      hover
                      className={classes.row}
                      key={u.id}
                    >
                      <TableCell className={classes.cell}>{u.email}</TableCell>
                      <TableCell className={classes.cell}>
                        {this.roleSelector(u)}
                        {(u.id === data.updateUserId && data.isUpdating) && (
                          <Spinner size='lg' className={classes.inTableProgress} />
                        )}
                      </TableCell>
                      <TableCell className={classes.cell}>{u.first_name}</TableCell>
                      <TableCell className={classes.cell}>{u.last_name}</TableCell>
                      <TableCell className={classes.actionCell}>
                        {(!u.is_owner && data.isDeleting && data.deleteUserId === u.id) && (
                          <Spinner size='lg' className={classes.inTableProgress} />
                        )}
                        {!u.is_owner && u.id !== env.data.info.id && <IconButton
                          aria-label='delete' title='Delete user from the organization'
                          disabled={data.isDeleting}
                          onClick={event => this.deleteHandler(event, u)}
                        >
                          <DeleteIcon/>
                        </IconButton>}
                        {u.id === env.data.info.id && <IconButton
                          aria-label='delete' title='Leave the organization'
                          disabled={data.isDeleting || u.is_owner}
                          onClick={event => this.deleteHandler(event, u)}
                        >
                          <ExitIcon/>
                        </IconButton>}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>) : 'Members not found'
        }

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

OrgSettings.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(OrgSettings);
