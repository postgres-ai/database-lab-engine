/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import Button from '@material-ui/core/Button'
import Grid from '@material-ui/core/Grid'
import TextField from '@material-ui/core/TextField'

import { styles } from '@postgres.ai/shared/styles/styles'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import ConsolePageTitle from '../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import Store from '../../stores/store'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { messages } from '../../assets/messages'
import { InviteFormProps } from 'components/AddMemberForm/AddMemberFormWrapper'
import { theme } from '@postgres.ai/shared/styles/theme'

interface InviteFormWithStylesProps extends InviteFormProps {
  classes: ClassesType
}

interface InviteFormState {
  email: string
  data: {
    auth: {
      token: string
    } | null
    inviteUser: {
      errorMessage: string
      updateErrorFields: string[]
      isUpdating: boolean
    } | null
    orgProfile: {
      orgId: number
      isProcessing: boolean
      isProcessed: boolean
      error: boolean
    } | null
  }
}

class InviteForm extends Component<InviteFormWithStylesProps, InviteFormState> {
  unsubscribe: () => void
  componentDidMount() {
    const that = this
    const { org, orgId } = this.props

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data })

      if (this.data.inviteUser.isProcessed && !this.data.inviteUser.error) {
        that.props.history.push('/' + org + '/members')
      }

      const auth: InviteFormState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const orgProfile: InviteFormState['data']['orgProfile'] =
        this.data && this.data.orgProfile ? this.data.orgProfile : null

      if (
        auth &&
        auth.token &&
        orgProfile &&
        orgProfile.orgId !== orgId &&
        !orgProfile.isProcessing &&
        !orgProfile.error
      ) {
        Actions.getOrgs(auth.token, orgId)
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value

    this.setState({
      email: value,
    })
  }

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.inviteUser : null

    if (auth && data && !data.isUpdating && this.state.email) {
      Actions.inviteUser(auth.token, orgId, this.state.email.trim())
    }
  }

  render() {
    const { classes, orgPermissions } = this.props

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: 'Add user' }]}
      />
    )

    const pageTitle = (
      <ConsolePageTitle
        title="Add a registered user"
        information="Use this form to add a registered users to your organization."
      />
    )

    if (orgPermissions && !orgPermissions.settingsMemberAdd) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}

          {pageTitle}

          <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
        </div>
      )
    }

    if (this.state && this.state.data && this.state.data.orgProfile?.error) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}

          {pageTitle}

          <ErrorWrapper />
        </div>
      )
    }

    if (
      !this.state ||
      !this.state.data ||
      !(this.state.data.orgProfile && this.state.data.orgProfile.isProcessed)
    ) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}

          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    const inviteData = this.state.data.inviteUser

    return (
      <div className={classes?.root}>
        {breadcrumbs}

        {pageTitle}

        <div>
          If the person is not registered yet, ask them to register first.
        </div>

        <div className={classes?.errorMessage}>
          {inviteData && inviteData.errorMessage ? (
            <div className={classes?.dense}>{inviteData.errorMessage}</div>
          ) : null}
        </div>

        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={8} className={classes?.container}>
            <Grid
              item
              xs
              container
              direction={theme.breakpoints.up('md') ? 'row' : 'column'}
              spacing={2}
            >
              <Grid item xs={12} sm={6}>
                <TextField
                  id="email"
                  label="Email"
                  variant="outlined"
                  value={this.state.email}
                  className={classes?.textField}
                  onChange={this.handleChange}
                  margin="normal"
                  error={
                    inviteData !== null &&
                    inviteData.updateErrorFields &&
                    inviteData.updateErrorFields.indexOf('email') !== -1
                  }
                  fullWidth
                  inputProps={{
                    name: 'email',
                    id: 'email',
                  }}
                  InputLabelProps={{
                    shrink: true,
                    style: styles.inputFieldLabel,
                  }}
                  FormHelperTextProps={{
                    style: styles.inputFieldHelper,
                  }}
                />

                <Button
                  variant="contained"
                  color="primary"
                  disabled={inviteData !== null && inviteData.isUpdating}
                  onClick={this.buttonHandler}
                  className={classes?.button}
                >
                  Add
                </Button>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
      </div>
    )
  }
}

export default InviteForm
