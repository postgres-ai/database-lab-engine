/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  IconButton,
  Select,
  MenuItem,
} from '@material-ui/core'
import DeleteIcon from '@material-ui/icons/Delete'
import ExitIcon from '@material-ui/icons/ExitToApp'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from '../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import Store from '../../stores/store'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { messages } from '../../assets/messages'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { OrgSettingsProps } from 'components/OrgMembers/OrgMembersWrapper'

interface OrgSettingsWithStylesProps extends OrgSettingsProps {
  classes: ClassesType
}

interface UsersType {
  id: number
  role_id: number
  email: string
  first_name: string
  last_name: string
  is_owner: boolean
}

interface OrgSettingsState {
  filterValue: string
  changes: {
    [change: number]: number
  }
  data: {
    auth: {
      token: string
    } | null
    orgUsers: {
      data: {
        roles: { name: string; id: number }[]
        users: UsersType[]
      }
      orgId: number
      isRefresh: boolean
      isRefreshed: boolean
      isUpdated: boolean
      isDeleted: boolean
      deleteUserId: number
      updateUserId: number
      error: boolean
      errorMessage: string
      errorCode: number
      isProcessing: boolean
      isUpdating: boolean
      isDeleting: boolean
    } | null
    userProfile: {
      data: {
        info: UsersType
      }
    } | null
  } | null
}

const membersTitle = 'Members'

class OrgSettings extends Component<
  OrgSettingsWithStylesProps,
  OrgSettingsState
> {
  unsubscribe: () => void
  componentDidMount() {
    const that = this
    const { orgId, orgPermissions } = this.props
    const hasListMembersPermission =
      !orgPermissions || orgPermissions.settingsMemberList

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const orgUsers =
        this.data && this.data.orgUsers ? this.data.orgUsers : null

      that.setState({ data: this.data })

      if (!hasListMembersPermission) {
        return
      }

      if (orgUsers?.isRefreshed) {
        that.setState({
          changes: {},
        })
      }

      if (orgUsers?.isUpdated || orgUsers?.isDeleted) {
        // Updated users list after deleting or updating.
        Actions.getOrgUsers(auth?.token, orgId)
      }

      if (
        auth &&
        auth.token &&
        orgUsers &&
        !orgUsers.isProcessing &&
        !orgUsers.error &&
        !that.state
      ) {
        // Initial loading of users.
        Actions.getOrgUsers(auth.token, orgId)
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  buttonAddHandler = () => {
    const org = this.props.org ? this.props.org : null

    this.props.history.push('/' + org + '/members/add')
  }

  deleteHandler = (_: React.MouseEvent<HTMLButtonElement>, u: UsersType) => {
    const { orgId, env } = this.props
    /* eslint no-alert: 0 */
    let userName: string | string[] = []
    if (u.first_name) {
      userName.push(u.first_name)
    }
    if (u.last_name) {
      userName.push(u.last_name)
    }
    userName = userName.join(' ')
    if (!userName) {
      userName = 'with email ' + u.email
    } else {
      userName = userName + ' (' + u.email + ')'
    }
    let message =
      'Are you sure you want to remove ' + userName + ' from the organization?'

    if (u.id === env.data.info.id) {
      message = 'Are you sure you want to leave the organization?'
    }

    if (window.confirm(message) === true) {
      Actions.deleteOrgUser(this.state.data?.auth?.token, orgId, u.id)
    }
  }

  changeRoleHandler = (userId: number, userRoleId: number) => {
    const { orgId } = this.props

    Actions.updateOrgUser(
      this.state.data?.auth?.token,
      orgId,
      userId,
      userRoleId,
    )

    this.setState({
      changes: {
        [userId]: userRoleId,
      },
    })
  }

  roleSelector(u: UsersType) {
    const that = this
    const { classes, orgPermissions } = this.props
    const roles =
      this.state.data &&
      this.state.data.orgUsers &&
      this.state.data.orgUsers.data &&
      this.state.data.orgUsers.data.roles
        ? this.state.data.orgUsers.data.roles
        : []

    if (orgPermissions?.settingsMemberUpdate && !u.is_owner) {
      return (
        <span>
          <Select
            value={
              that.state && that.state.changes && that.state.changes[u.id]
                ? that.state.changes[u.id]
                : u.role_id
            }
            onChange={(event) => {
              if (u.role_id !== event.target.value) {
                that.changeRoleHandler(u.id, event.target.value as number)
              }
            }}
            className={classes.roleSelector}
            variant="outlined"
          >
            {roles.map((r) => {
              return (
                <MenuItem value={r.id} className={classes.roleSelectorItem}>
                  {r.name}
                </MenuItem>
              )
            })}
          </Select>
        </span>
      )
    }

    // Just output role name.
    let role = '-'
    for (let i in roles) {
      if (roles[i].id === u.role_id) {
        role = roles[i].name
      }
    }

    return (
      <span>
        {role}
        {u.is_owner ? ' (Owner)' : ''}
      </span>
    )
  }

  filterInputHandler = (event: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ filterValue: event.target.value })
  }

  render() {
    const { classes, orgPermissions, orgId, env } = this.props
    const data = this.state && this.state.data ? this.state.data.orgUsers : null
    const userProfile =
      this.state && this.state.data ? this.state.data.userProfile : null

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: membersTitle }]}
      />
    )

    const hasAddMemberPermission =
      !orgPermissions || orgPermissions.settingsMemberAdd
    const hasListMembersPermission =
      !orgPermissions || orgPermissions.settingsMemberList

    const actions = [
      <ConsoleButtonWrapper
        disabled={!hasAddMemberPermission}
        title={
          hasAddMemberPermission ? 'Add a new member' : messages.noPermission
        }
        variant="contained"
        color="primary"
        onClick={this.buttonAddHandler}
      >
        Add
      </ConsoleButtonWrapper>,
    ]

    let users: UsersType[] = []
    if (hasListMembersPermission) {
      if (data && data.data && data.data.users && data.data.users.length > 0) {
        users = data.data.users
      }
    } else if (userProfile && userProfile.data && userProfile.data.info) {
      users = [userProfile.data.info]
    }

    const filteredUsers = users?.filter((user) => {
      const fullName = (user.first_name || '') + ' ' + (user.last_name || '')
      return (
        fullName
          ?.toLowerCase()
          .indexOf((this.state.filterValue || '')?.toLowerCase()) !== -1
      )
    })

    const pageTitle = (
      <ConsolePageTitle
        title={membersTitle}
        actions={actions}
        filterProps={
          users && users.length > 0
            ? {
                filterValue: this.state.filterValue,
                filterHandler: this.filterInputHandler,
                placeholder: 'Search users by name',
              }
            : null
        }
      />
    )

    if (
      !data ||
      (hasListMembersPermission && data.orgId !== orgId) ||
      (data.isProcessing && data.isRefresh === false)
    ) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    if (data.error) {
      return (
        <div className={classes.root}>
          <ErrorWrapper message={data.errorMessage} code={data.errorCode} />
        </div>
      )
    }

    // If user does not have "ListMembersPermission" we will fill the list only
    // with his data without making getOrgUsers request.

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {!hasListMembersPermission && (
          <WarningWrapper>
            You do not have permission to view the full list of members
          </WarningWrapper>
        )}

        {filteredUsers && filteredUsers.length > 0 ? (
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell>Email</TableCell>
                  <TableCell>Role</TableCell>
                  <TableCell>First name</TableCell>
                  <TableCell>Last name</TableCell>
                  <TableCell />
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredUsers.map((u: UsersType) => {
                  return (
                    <TableRow hover className={classes.row} key={u.id}>
                      <TableCell className={classes.cell}>{u.email}</TableCell>
                      <TableCell className={classes.cell}>
                        {this.roleSelector(u)}
                        {u.id === data.updateUserId && data.isUpdating && (
                          <Spinner
                            size="lg"
                            className={classes.inTableProgress}
                          />
                        )}
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {u.first_name}
                      </TableCell>
                      <TableCell className={classes.cell}>
                        {u.last_name}
                      </TableCell>
                      <TableCell className={classes.actionCell}>
                        {!u.is_owner &&
                          data.isDeleting &&
                          data.deleteUserId === u.id && (
                            <Spinner
                              size="lg"
                              className={classes.inTableProgress}
                            />
                          )}
                        {!u.is_owner && u.id !== env.data.info.id && (
                          <IconButton
                            aria-label="delete"
                            title="Delete user from the organization"
                            disabled={data.isDeleting}
                            onClick={(event) => this.deleteHandler(event, u)}
                          >
                            <DeleteIcon />
                          </IconButton>
                        )}
                        {u.id === env.data.info.id && (
                          <IconButton
                            aria-label="delete"
                            title="Leave the organization"
                            disabled={data.isDeleting || u.is_owner}
                            onClick={(event) => this.deleteHandler(event, u)}
                          >
                            <ExitIcon />
                          </IconButton>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        ) : (
          'Members not found'
        )}

        <div className={classes.bottomSpace} />
      </div>
    )
  }
}

export default OrgSettings
