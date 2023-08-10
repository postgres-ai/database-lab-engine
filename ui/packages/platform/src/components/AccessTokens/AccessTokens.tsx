/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component, MouseEvent } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Button,
  FormControlLabel,
  Checkbox,
} from '@material-ui/core'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { styles } from '@postgres.ai/shared/styles/styles'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import {
  ClassesType,
  RefluxTypes,
  TokenRequestProps,
} from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import ConsolePageTitle from '../ConsolePageTitle'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DisplayTokenWrapper } from 'components/DisplayToken/DisplayTokenWrapper'
import { AccessTokensProps } from 'components/AccessTokens/AccessTokensWrapper'
import { FilteredTableMessage } from 'components/AccessTokens/FilteredTableMessage/FilteredTableMessage'

interface AccessTokensWithStylesProps extends AccessTokensProps {
  classes: ClassesType
}

interface UserTokenData {
  id: number
  name: string
  is_personal: boolean
  username: string
  created_formatted: string
  expires_formatted: string
  revoking: boolean
}

interface AccessTokensState {
  filterValue: string
  data: {
    auth: {
      token: string
    } | null
    userTokens: {
      orgId: number | null
      isProcessing: boolean
      isProcessed: boolean
      data: UserTokenData[]
      error: {
        message: boolean
      }
    }
    tokenRequest: TokenRequestProps
  }
  tokenName: string | null
  tokenExpires: string | null
  processed: boolean
  isPersonal: boolean
}

class AccessTokens extends Component<
  AccessTokensWithStylesProps,
  AccessTokensState
> {
  state = {
    filterValue: '',
    data: {
      auth: {
        token: '',
      },
      userTokens: {
        orgId: null,
        isProcessing: false,
        isProcessed: false,
        data: [],
        error: {
          message: false,
        },
      },
      tokenRequest: {
        isProcessing: false,
        isProcessed: false,
        data: {
          name: '',
          is_personal: false,
          expires_at: '',
          token: '',
        },
        errorMessage: '',
        error: false,
      },
    },
    tokenName: '',
    tokenExpires: '',
    processed: false,
    isPersonal: true,
  }

  handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const name = event.target.name
    const value = event.target.value

    if (name === 'tokenName') {
      this.setState({ tokenName: value })
    } else if (name === 'tokenExpires') {
      this.setState({ tokenExpires: value })
    } else if (name === 'isPersonal') {
      this.setState({ isPersonal: event.target.checked })
    }
  }
  unsubscribe: Function
  componentDidMount() {
    const that = this
    const orgId = this.props.orgId ? this.props.orgId : null
    const date = new Date()
    const expiresDate =
      date.getFullYear() +
      1 +
      '-' +
      ('0' + (date.getMonth() + 1)).slice(-2) +
      '-' +
      ('0' + date.getDate()).slice(-2)

    document.getElementsByTagName('html')[0].style.overflow = 'hidden'

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const auth: AccessTokensState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const userTokens: AccessTokensState['data']['userTokens'] =
        this.data && this.data.userTokens ? this.data.userTokens : null
      const tokenRequest: TokenRequestProps =
        this.data && this.data.tokenRequest ? this.data.tokenRequest : null

      that.setState({ data: this.data })

      if (
        auth &&
        auth.token &&
        (!userTokens.isProcessed || orgId !== userTokens.orgId) &&
        !userTokens.isProcessing &&
        !userTokens.error
      ) {
        Actions.getAccessTokens(auth.token, orgId)
      }

      if (
        tokenRequest &&
        tokenRequest.isProcessed &&
        !tokenRequest.error &&
        tokenRequest.data &&
        tokenRequest.data.name === that.state.tokenName &&
        tokenRequest.data.expires_at &&
        tokenRequest.data.token
      ) {
        that.setState({
          tokenName: '',
          tokenExpires: expiresDate,
          processed: false,
          isPersonal: true,
        })
      }
    })

    that.setState({
      tokenName: '',
      tokenExpires: expiresDate,
      processed: false,
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    Actions.hideGeneratedAccessToken()
    this.unsubscribe()
  }

  addToken = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const tokenRequest =
      this.state.data && this.state.data.tokenRequest
        ? this.state.data.tokenRequest
        : null

    if (
      this.state.tokenName === null ||
      this.state.tokenName === '' ||
      this.state.tokenExpires === null ||
      this.state.tokenExpires === ''
    ) {
      this.setState({ processed: true })
      return
    }

    if (auth && auth.token && !tokenRequest?.isProcessing) {
      Actions.getAccessToken(
        auth.token,
        this.state.tokenName,
        this.state.tokenExpires,
        orgId,
        this.state.isPersonal,
      )
    }
  }

  getTodayDate() {
    const date = new Date()

    return (
      date.getFullYear() +
      '-' +
      ('0' + (date.getMonth() + 1)).slice(-2) +
      '-' +
      ('0' + date.getDate()).slice(-2)
    )
  }

  revokeToken = (
    _event: MouseEvent<HTMLButtonElement>,
    id: number,
    name: string,
  ) => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null

    /* eslint no-alert: 0 */
    if (
      window.confirm(
        'Are you sure you want to revoke token "' + name + '"?',
      ) === true
    ) {
      Actions.revokeAccessToken(auth?.token, orgId, id)
    }
  }

  filterTokensInputHandler = (event: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ filterValue: event.target.value })
  }

  render() {
    const { classes, orgPermissions, orgId } = this.props
    const data =
      this.state && this.state.data ? this.state.data.userTokens : null
    const tokenRequest =
      this.state && this.state.data && this.state.data.tokenRequest
        ? this.state.data.tokenRequest
        : null
    const filteredTokens = data?.data?.filter(
      (token: UserTokenData) =>
        token.name
          ?.toLowerCase()
          .indexOf((this.state.filterValue || '')?.toLowerCase()) !== -1,
    )

    const pageTitle = (
      <ConsolePageTitle
        title="Access tokens"
        filterProps={
          data && data.data?.length > 0
            ? {
                filterValue: this.state.filterValue,
                filterHandler: this.filterTokensInputHandler,
                placeholder: 'Search access tokens by name',
              }
            : null
        }
      />
    )

    let tokenDisplay = null
    if (
      tokenRequest &&
      tokenRequest.isProcessed &&
      !tokenRequest.error &&
      tokenRequest.data &&
      tokenRequest.data.name &&
      tokenRequest.data.expires_at &&
      tokenRequest.data.token
    ) {
      tokenDisplay = (
        <div>
          <h3>
            {tokenRequest.data.is_personal
              ? 'Your new personal access token'
              : 'New administrative access token'}
          </h3>
          <DisplayTokenWrapper />
        </div>
      )
    }

    let tokenError = null
    if (tokenRequest && tokenRequest.error) {
      tokenError = (
        <div className={classes?.errorMessage}>{tokenRequest.errorMessage}</div>
      )
    }

    const tokenForm = (
      <div>
        <h2>Add token</h2>
        <div className={classes?.container}>
          {tokenError}

          <TextField
            id="tokenName"
            variant="outlined"
            label="Name"
            placeholder="Token name"
            margin="normal"
            required={true}
            value={this.state.tokenName}
            disabled={tokenRequest !== null && tokenRequest.isProcessing}
            error={
              this.state.processed &&
              (this.state.tokenName === null || this.state.tokenName === '')
            }
            className={classes?.nameField}
            onChange={this.handleChange}
            inputProps={{
              name: 'tokenName',
              id: 'tokenName',
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
            id="tokenExpires"
            variant="outlined"
            label="Expires"
            type="date"
            required={true}
            defaultValue={this.state.tokenExpires}
            value={this.state.tokenExpires}
            disabled={tokenRequest !== null && tokenRequest.isProcessing}
            className={classes?.textField}
            error={
              this.state.processed &&
              (this.state.tokenExpires === null ||
                this.state.tokenExpires === '')
            }
            onChange={this.handleChange}
            inputProps={{
              name: 'tokenExpires',
              id: 'tokenExpires',
              min: this.getTodayDate(),
            }}
            InputLabelProps={{
              shrink: true,
              style: styles.inputFieldLabel,
            }}
            FormHelperTextProps={{
              style: styles.inputFieldHelper,
            }}
          />

          <FormControlLabel
            control={
              <Checkbox
                checked={this.state.isPersonal}
                onChange={this.handleChange}
                name="isPersonal"
                color="primary"
                disabled={!orgPermissions.settingsTokenCreateImpersonal}
                inputProps={{
                  name: 'isPersonal',
                  id: 'isPersonal',
                }}
              />
            }
            label="Personal token"
          />

          <Button
            variant="contained"
            color="primary"
            className={classes?.addTokenButton}
            disabled={tokenRequest !== null && tokenRequest.isProcessing}
            onClick={this.addToken}
          >
            Add token
          </Button>
        </div>
      </div>
    )

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: 'Access tokens' }]}
      />
    )

    if (this.state && this.state.data && this.state.data.userTokens?.error) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}

          {pageTitle}

          <h2>Access tokens</h2>
          <ErrorWrapper />
        </div>
      )
    }

    if (
      !data ||
      (data && data.isProcessing) ||
      (data && data.orgId !== orgId)
    ) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}
          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    return (
      <div className={classes?.root}>
        {breadcrumbs}

        {pageTitle}

        {tokenDisplay}

        {tokenForm}

        <div className={classes?.remark}>
          Users may manage their personal tokens only. Admins may manage their
          &nbsp;personal tokens, as well as administrative (impersonal) tokens
          &nbsp;used to organize infrastructure. Tokens of all types work in
          &nbsp;the context of a particular organization.
        </div>

        <br />
        <h2>Active access tokens</h2>

        {filteredTokens && filteredTokens.length > 0 ? (
          <HorizontalScrollContainer>
            <Table className={classes?.table}>
              <TableHead>
                <TableRow className={classes?.row}>
                  <TableCell>Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Creator</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>Expires</TableCell>
                  <TableCell align="right">Actions</TableCell>
                </TableRow>
              </TableHead>

              <TableBody>
                {filteredTokens &&
                  filteredTokens.length > 0 &&
                  filteredTokens.map((t: UserTokenData) => {
                    return (
                      <TableRow className={classes?.row} key={t.id}>
                        <TableCell className={classes?.cell}>
                          {t.name}
                        </TableCell>
                        <TableCell className={classes?.cell}>
                          {t.is_personal ? 'Personal' : 'Administrative'}
                        </TableCell>
                        <TableCell className={classes?.cell}>
                          {t.username}
                        </TableCell>
                        <TableCell className={classes?.cell}>
                          {t.created_formatted}
                        </TableCell>
                        <TableCell className={classes?.cell}>
                          {t.expires_formatted}
                        </TableCell>
                        <TableCell className={classes?.cell} align="right">
                          <Button
                            variant="outlined"
                            color="secondary"
                            className={classes?.revokeButton}
                            disabled={
                              (!t.is_personal &&
                                !orgPermissions.settingsTokenRevokeImpersonal) ||
                              t.revoking
                            }
                            onClick={(event) =>
                              this.revokeToken(event, t.id, t.name)
                            }
                          >
                            Revoke
                          </Button>
                        </TableCell>
                      </TableRow>
                    )
                  })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        ) : (
          <FilteredTableMessage
            filteredItems={filteredTokens}
            emptyState="This user has no active access tokens"
            filterValue={this.state.filterValue}
            clearFilter={() =>
              this.setState({
                filterValue: '',
              })
            }
          />
        )}

        <div className={classes?.bottomSpace} />
      </div>
    )
  }
}

export default AccessTokens
