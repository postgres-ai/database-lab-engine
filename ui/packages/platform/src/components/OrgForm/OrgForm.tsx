/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import {
  Grid,
  Button,
  TextField,
  FormControlLabel,
  Checkbox,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  IconButton,
} from '@material-ui/core'
import DeleteIcon from '@material-ui/icons/Delete'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { styles } from '@postgres.ai/shared/styles/styles'
import { Link } from '@postgres.ai/shared/components/Link2'
import { ClassesType } from '@postgres.ai/platform/src/components/types'
import { theme } from '@postgres.ai/shared/styles/theme'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from '../ConsolePageTitle'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { messages } from '../../assets/messages'
import { formatAlias } from 'utils/aliases'
import { OrgFormProps } from 'components/OrgForm/OrgFormWrapper'

interface OrgFormWithStylesProps extends OrgFormProps {
  classes: ClassesType
}

interface DomainsType {
  [domain: string]: {
    confirmed: boolean
    id: number | null
  }
}

interface OrgFormState {
  id: number | null
  name: string
  alias: string
  onboardingText: string
  domain: string
  orgChanged: boolean
  usersAutojoin: boolean
  data: {
    auth: {
      token: string | null
    } | null
    orgProfile: {
      isUpdating: boolean
      error: boolean
      updateError: boolean
      errorMessage: string | undefined
      errorCode: number | undefined
      updateErrorMessage: string | null
      isProcessing: boolean
      orgId: number | null
      updateErrorFields: string[]
      orgDomains: {
        isProcessing: boolean
        isProcessed: boolean
        isDeleting: boolean
        domainId: number | null
      }
      data: {
        id: number | null
        name: string
        alias: string
        updateName: string
        updateAlias: string
        isProcessing: boolean
        updateUsersAutojoin: boolean
        onboarding_text: string
        users_autojoin: boolean
        domains: DomainsType | null
      } | null
    } | null
  } | null
}

class OrgForm extends Component<OrgFormWithStylesProps, OrgFormState> {
  state = {
    id: null,
    name: '',
    alias: '',
    onboardingText: '',
    domain: '',
    orgChanged: true,
    usersAutojoin: false,
    data: {
      auth: {
        token: null,
      },
      orgProfile: {
        isUpdating: false,
        isProcessing: false,
        error: false,
        updateError: false,
        errorMessage: undefined,
        errorCode: undefined,
        updateErrorMessage: null,
        updateErrorFields: [''],
        orgId: null,
        orgDomains: {
          isProcessing: false,
          isProcessed: false,
          isDeleting: false,
          domainId: null,
        },
        data: {
          id: null,
          name: '',
          alias: '',
          updateName: '',
          updateAlias: '',
          isProcessing: false,
          updateUsersAutojoin: false,
          onboarding_text: '',
          users_autojoin: false,
          domains: null,
        },
      },
    },
  }

  unsubscribe: () => void
  componentDidMount() {
    const { orgId, mode } = this.props
    const that = this

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const orgProfile =
        this.data && this.data.orgProfile ? this.data.orgProfile : null

      that.setState({ data: this.data })

      if (
        auth &&
        auth.token &&
        orgProfile &&
        orgProfile.orgId !== orgId &&
        !orgProfile.isProcessing
      ) {
        Actions.getOrgs(auth.token, orgId)
      }

      if (!that.isAdd(mode) && orgProfile && orgProfile.data) {
        if (orgProfile.orgDomains && orgProfile.orgDomains.isProcessed) {
          that.setState({
            domain: '',
          })
        }

        if (orgProfile.data.updateName && orgProfile.data.updateAlias) {
          that.setState({
            id: orgProfile.orgId,
            name: orgProfile.data.updateName,
            alias: orgProfile.data.updateAlias,
            usersAutojoin: orgProfile.data.updateUsersAutojoin,
            onboardingText: orgProfile.data['onboarding_text'],
            orgChanged: false,
          })
        } else {
          if (that.state.id && that.state.id === orgId) {
            return
          }

          if (orgProfile.isUpdating || orgProfile.updateError) {
            return
          }

          that.setState({
            id: orgProfile.data.id,
            name: orgProfile.data.name,
            alias: orgProfile.data.alias,
            usersAutojoin: orgProfile.data['users_autojoin'],
            onboardingText: orgProfile.data['onboarding_text'],
            orgChanged: false,
          })
        }
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.orgProfile : null

    if (
      auth &&
      data &&
      !data.isUpdating &&
      (this.state.name || this.state.alias)
    ) {
      if (this.isAdd(this.props.mode)) {
        Actions.createOrg(auth.token, {
          name: this.state.name,
          alias: this.state.alias,
          email_domain_autojoin: this.state.domain,
          users_autojoin: this.state.usersAutojoin,
        })
      } else if (this.state.orgChanged) {
        Actions.updateOrg(auth.token, orgId, {
          name: this.state.name,
          alias: this.state.alias,
          users_autojoin: this.state.usersAutojoin,
          onboarding_text: this.state.onboardingText,
        })
      }
    }
  }

  domainButtonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const data = this.state.data ? this.state.data.orgProfile : null

    if (
      auth &&
      data &&
      !data.isUpdating &&
      !this.isAdd(this.props.mode) &&
      (this.state.name || this.state.alias)
    ) {
      if (this.state.domain) {
        Actions.addOrgDomain(auth.token, orgId, this.state.domain)
      }
    }
  }

  deleteDomainHandler = (domainId: number | null, confirmed: boolean) => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null

    /* eslint no-alert: 0 */
    if (
      confirmed &&
      window.confirm(
        'Are you sure you want to delete already confirmed domain?',
      ) === false
    ) {
      return
    }

    Actions.deleteOrgDomain(auth?.token, orgId, domainId)
  }

  isAdd = (mode: OrgFormProps['mode']) => {
    return mode === 'new'
  }

  render() {
    const { classes, orgPermissions, mode } = this.props
    const orgId = this.props.orgId ? this.props.orgId : null
    const data =
      this.state && this.state.data ? this.state.data.orgProfile : null
    const breadcrumbs = this.isAdd(mode) ? (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[
          { name: 'Organizations', url: '/' },
          { name: 'Create organization' },
        ]}
      />
    ) : (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: 'Settings' }]}
      />
    )
    const pageTitle = this.isAdd(mode) ? (
      <ConsolePageTitle title="Create organization" />
    ) : (
      <ConsolePageTitle title="Settings" />
    )

    if (orgPermissions && !orgPermissions.settingsOrganizationUpdate) {
      return (
        <>
          {breadcrumbs}

          {pageTitle}

          <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
        </>
      )
    }

    let domains = <div>No entries.</div>

    let domainField = (
      <div>
        <TextField
          id="domain"
          label="Organization domain"
          variant="outlined"
          disabled={data?.isUpdating}
          value={this.state.domain}
          className={
            !this.isAdd(mode) ? classes.editDomainField : classes.newDomainField
          }
          onChange={(e) => {
            this.setState({
              domain: e.target.value,
            })
          }}
          error={
            data?.updateErrorFields &&
            data.updateErrorFields.indexOf('domain') !== -1
          }
          margin="normal"
          fullWidth
          inputProps={{
            name: 'domain',
            id: 'domain',
            shrink: 'true',
          }}
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel,
          }}
          FormHelperTextProps={{
            style: styles.inputFieldHelper,
          }}
        />

        {!this.isAdd(mode) ? (
          <Button
            variant="contained"
            color="primary"
            disabled={data?.orgDomains.isProcessing}
            onClick={this.domainButtonHandler}
            className={classes.domainButton}
          >
            Add
          </Button>
        ) : null}
      </div>
    )

    let buttonTitle = 'Save'
    if (this.isAdd(mode)) {
      buttonTitle = 'Create'
    }

    if (
      this.state &&
      this.state.data &&
      this.state.data.orgProfile.error &&
      this.state.data.orgProfile.orgId === orgId &&
      !this.state.data.orgProfile.isProcessing
    ) {
      return (
        <div>
          <ErrorWrapper message={data?.errorMessage} code={data?.errorCode} />
        </div>
      )
    }

    if (
      !this.isAdd(mode) &&
      (!data || !data.data || data.data.id !== orgId || data.data.isProcessing)
    ) {
      return (
        <>
          {breadcrumbs}
          {pageTitle}

          <PageSpinner />
        </>
      )
    }

    if (
      !this.isAdd(mode) &&
      data &&
      data?.data?.domains &&
      Object.keys(data.data.domains).length > 0
    ) {
      const domainsData: DomainsType = data?.data?.domains
      domains = (
        <div>
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell className={classes.tableCell}>Domain</TableCell>
                  <TableCell className={classes.tableCell}>Confirmed</TableCell>
                  <TableCell className={classes.tableCell}>&nbsp;</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {Object.keys(domainsData).map((d) => {
                  return (
                    <TableRow key={d}>
                      <TableCell className={classes.tableCell}>{d}</TableCell>
                      <TableCell className={classes.tableCell}>
                        {domainsData[d].confirmed ? (
                          'Yes'
                        ) : (
                          <span>
                            No&nbsp;&nbsp;
                            <span className={classes.errorMessage}>
                              To confirm&nbsp;
                              <span
                                className={
                                  window.Intercom && classes.supportLink
                                }
                                onClick={() =>
                                  window.Intercom && window.Intercom('show')
                                }
                              >
                                contact support
                              </span>
                            </span>
                          </span>
                        )}
                      </TableCell>
                      <TableCell className={classes.tableCellRight}>
                        {data !== null &&
                        domainsData &&
                        data?.orgDomains &&
                        data.orgDomains.isDeleting &&
                        data.orgDomains.domainId === domainsData[d].id ? (
                          <Spinner className={classes.inTableProgress} />
                        ) : null}
                        <IconButton
                          className={classes.tableActionButton}
                          color="primary"
                          aria-label="Delete domain"
                          onClick={() => {
                            this.deleteDomainHandler(
                              domainsData[d].id,
                              domainsData[d].confirmed,
                            )
                          }}
                          component="span"
                          disabled={
                            data &&
                            data.orgDomains &&
                            data.orgDomains.isDeleting
                          }
                        >
                          <DeleteIcon />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        </div>
      )
    }

    return (
      <>
        {breadcrumbs}

        {pageTitle}

        <div className={classes.errorMessage}>
          {data && data.updateErrorMessage ? data.updateErrorMessage : null}
        </div>

        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={12} className={classes.container}>
            <Grid
              xs={12}
              sm={12}
              lg={8}
              item
              container
              direction={theme.breakpoints.up('md') ? 'row' : 'column'}
            >
              <Grid item xs={12} sm={6}>
                <TextField
                  id="orgNameTextField"
                  label="Organization name"
                  variant="outlined"
                  disabled={data?.isUpdating}
                  value={this.state.name}
                  className={classes.textField}
                  onChange={(e) => {
                    if (this.isAdd(this.props.mode)) {
                      this.setState({ alias: formatAlias(e.target.value) })
                    }
                    this.setState({
                      orgChanged: true,
                      name: e.target.value,
                    })
                  }}
                  margin="normal"
                  error={
                    data?.updateErrorFields &&
                    data.updateErrorFields.indexOf('name') !== -1
                  }
                  fullWidth
                  inputProps={{
                    name: 'name',
                    id: 'orgNameTextField',
                    shrink: 'true',
                  }}
                  InputLabelProps={{
                    shrink: true,
                    style: styles.inputFieldLabel,
                  }}
                  FormHelperTextProps={{
                    style: styles.inputFieldHelper,
                  }}
                />

                <TextField
                  id="orgAliasTextField"
                  label="Organization unique ID"
                  fullWidth
                  variant="outlined"
                  disabled={data?.isUpdating}
                  value={this.state.alias}
                  className={classes.textField}
                  onChange={(e) => {
                    this.setState({
                      orgChanged: true,
                      alias: formatAlias(e.target.value),
                    })
                  }}
                  error={
                    data?.updateErrorFields &&
                    data.updateErrorFields.indexOf('alias') !== -1
                  }
                  margin="normal"
                  inputProps={{
                    name: 'alias',
                    id: 'orgAliasTextField',
                    shrink: 'true',
                  }}
                  InputLabelProps={{
                    shrink: true,
                    style: styles.inputFieldLabel,
                  }}
                  FormHelperTextProps={{
                    style: styles.inputFieldHelper,
                  }}
                />

                {!this.isAdd(this.props.mode) ? (
                  <TextField
                    id="onboardingText"
                    label="Getting started text"
                    fullWidth
                    multiline
                    variant="outlined"
                    disabled={data?.isUpdating}
                    value={this.state.onboardingText}
                    className={classes.onboardingField}
                    onChange={(e) => {
                      this.setState({
                        orgChanged: true,
                        onboardingText: e.target.value,
                      })
                    }}
                    helperText={
                      <span>
                        Format:&nbsp;
                        <Link
                          to={'https://commonmark.org/help'}
                          target={'_blank'}
                        >
                          Markdown
                        </Link>
                      </span>
                    }
                    error={
                      data?.updateErrorFields &&
                      data.updateErrorFields.indexOf('onboardingText') !== -1
                    }
                    margin="normal"
                    inputProps={{
                      name: 'onboardingText',
                      id: 'onboardingText',
                      shrink: 'true',
                    }}
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                ) : null}
              </Grid>
            </Grid>

            {!this.isAdd(mode) && (
              <Grid item xs={12} sm={12} lg={8} className={classes.container}>
                <Grid item xs={12} sm={6}>
                  <FormControlLabel
                    className={classes.autoJoinCheckbox}
                    control={
                      <Checkbox
                        checked={this.state.usersAutojoin}
                        onChange={(e) => {
                          this.setState({
                            orgChanged: true,
                            usersAutojoin: e.target.checked,
                          })
                        }}
                        disabled={data?.isUpdating}
                        name="usersAutojoin"
                      />
                    }
                    label="Auto-join based on email domain"
                  />

                  <div>
                    <div className={classes.comment}>
                      Users that have email addresses with the following domain
                      name(s)&nbsp; automatically join the organization when
                      registering in the system.
                      <br />
                    </div>

                    <br />

                    {domains}

                    <br />

                    {domainField}

                    <br />
                  </div>
                </Grid>
              </Grid>
            )}

            <Grid
              item
              xs={12}
              sm={12}
              lg={8}
              className={classes.updateButtonContainer}
            >
              <Button
                variant="contained"
                color="primary"
                disabled={data?.isUpdating}
                id="orgSaveButton"
                onClick={this.buttonHandler}
              >
                {buttonTitle}
              </Button>
            </Grid>
          </Grid>
        </Grid>

        <div className={classes.bottomSpace} />
      </>
    )
  }
}

export default OrgForm
