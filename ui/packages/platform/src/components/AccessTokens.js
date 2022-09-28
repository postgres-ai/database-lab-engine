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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Button,
  FormControlLabel,
  Checkbox
} from '@material-ui/core';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { styles } from '@postgres.ai/shared/styles/styles';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import DisplayToken from './DisplayToken';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';

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
    ...styles.inputField,
    maxWidth: 400,
    marginBottom: 15,
    marginRight: theme.spacing(1),
    marginTop: '16px'
  },
  nameField: {
    ...styles.inputField,
    maxWidth: 400,
    marginBottom: 15,
    width: '400px',
    marginRight: theme.spacing(1)
  },
  addTokenButton: {
    marginTop: 15,
    height: '33px',
    marginBottom: 10
  },
  revokeButton: {
    paddingRight: 5,
    paddingLeft: 5,
    paddingTop: 3,
    paddingBottom: 3
  },
  errorMessage: {
    color: 'red',
    width: '100%'
  },
  remark: {
    width: '100%',
    maxWidth: 960
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

class AccessTokens extends Component {
  state = {
    tokenName: '',
    tokenExpires: '',
    processed: false,
    isPersonal: true
  };

  handleChange = event => {
    const name = event.target.name;
    const value = event.target.value;

    if (name === 'tokenName') {
      this.setState({ tokenName: value });
    } else if (name === 'tokenExpires') {
      this.setState({ tokenExpires: value });
    } else if (name === 'isPersonal') {
      this.setState({ isPersonal: event.target.checked });
    }
  };

  componentDidMount() {
    const that = this;
    const orgId = this.props.orgId ? this.props.orgId : null;
    let date = new Date();
    let expiresDate = (date.getFullYear() + 1) + '-' +
      ('0' + (date.getMonth() + 1)).slice(-2) +
      '-' + ('0' + date.getDate()).slice(-2);

    document.getElementsByTagName('html')[0].style.overflow = 'hidden';

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const userTokens = this.data && this.data.userTokens ?
        this.data.userTokens : null;
      const tokenRequest = this.data && this.data.tokenRequest ?
        this.data.tokenRequest : null;

      that.setState({ data: this.data });

      if (auth && auth.token && (!userTokens.isProcessed ||
          orgId !== userTokens.orgId) && !userTokens.isProcessing &&
        !userTokens.error) {
        Actions.getAccessTokens(auth.token, orgId);
      }

      if (tokenRequest && tokenRequest.isProcessed && !tokenRequest.error &&
        tokenRequest.data && tokenRequest.data.name === that.state.tokenName &&
        tokenRequest.data.expires_at && tokenRequest.data.token) {
        that.setState({
          tokenName: '',
          tokenExpires: expiresDate,
          processed: false,
          isPersonal: true
        });
      }
    });

    that.setState({ tokenName: '', tokenExpires: expiresDate, processed: false });

    Actions.refresh();
  }

  componentWillUnmount() {
    Actions.hideGeneratedAccessToken();
    this.unsubscribe();
  }

  addToken = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const tokenRequest = this.state.data && this.state.data.tokenRequest ?
      this.state.data.tokenRequest : null;

    if (this.state.tokenName === null || this.state.tokenName === '' ||
      this.state.tokenExpires === null || this.state.tokenExpires === '') {
      this.setState({ processed: true });
      return;
    }

    if (auth && auth.token && !tokenRequest.isProcessing) {
      Actions.getAccessToken(auth.token, this.state.tokenName,
        this.state.tokenExpires, orgId, this.state.isPersonal);
    }
  };

  getTodayDate() {
    const date = new Date();

    return date.getFullYear() + '-' + ('0' + (date.getMonth() + 1)).slice(-2) +
      '-' + ('0' + date.getDate()).slice(-2);
  }

  revokeToken = (event, id, name) => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;

    /* eslint no-alert: 0 */
    if (window.confirm('Are you sure you want to revoke token "' +
      name + '"?') === true) {
      Actions.revokeAccessToken(auth.token, orgId, id);
    }
  };

  render() {
    const { classes, orgPermissions, orgId } = this.props;
    const data = this.state && this.state.data ?
      this.state.data.userTokens : null;
    const tokenRequest = this.state && this.state.data &&
      this.state.data.tokenRequest ? this.state.data.tokenRequest : null;
    const pageTitle = (
      <ConsolePageTitle title='Access tokens'/>
    );

    let tokenDisplay = null;
    if (tokenRequest && tokenRequest.isProcessed && !tokenRequest.error &&
      tokenRequest.data && tokenRequest.data.name &&
      tokenRequest.data.expires_at && tokenRequest.data.token) {
      tokenDisplay = (
        <div>
          <h3>{tokenRequest.data.is_personal ? 'Your new personal access token' :
            'New administrative access token'}</h3>
          <DisplayToken/>
        </div>
      );
    }

    let tokenError = null;
    if (tokenRequest && tokenRequest.error) {
      tokenError = (
        <div className={classes.errorMessage}>
          {tokenRequest.errorMessage}
        </div>
      );
    }

    const tokenForm = (
      <div>
        <h2>Add token</h2>
        <div className={classes.container}>
          { tokenError }

          <TextField
            id='tokenName'
            ref='tokenName'
            variant='outlined'
            label='Name'
            placeholder='Token name'
            margin='normal'
            required={true}
            value={this.state.tokenName}
            disabled={tokenRequest && tokenRequest.isProcessing}
            error={this.state.processed &&
              (this.state.tokenName === null || this.state.tokenName === '')}
            className={classes.nameField}
            onChange={this.handleChange}
            inputProps={{
              name: 'tokenName',
              id: 'tokenName',
              shrink: 'true'
            }}
            InputLabelProps={{
              shrink: true,
              style: styles.inputFieldLabel
            }}
            FormHelperTextProps={{
              style: styles.inputFieldHelper
            }}
          />

          <TextField
            id='tokenExpires'
            ref='tokenExpires'
            variant='outlined'
            label='Expires'
            type='date'
            required={true}
            defaultValue={this.state.tokenExpires}
            value={this.state.tokenExpires}
            disabled={tokenRequest && tokenRequest.isProcessing}
            className={classes.textField}
            error={this.state.processed &&
              (this.state.tokenExpires === null || this.state.tokenExpires === '')}
            onChange={this.handleChange}
            inputProps={{
              name: 'tokenExpires',
              id: 'tokenExpires',
              min: this.getTodayDate()
            }}
            InputLabelProps={{
              shrink: true,
              style: styles.inputFieldLabel
            }}
            FormHelperTextProps={{
              style: styles.inputFieldHelper
            }}
          />

          <FormControlLabel
            control={
              <Checkbox
                checked={this.state.isPersonal}
                onChange={this.handleChange}
                name='isPersonal'
                color='primary'
                disabled={!orgPermissions.settingsTokenCreateImpersonal}
                inputProps={{
                  name: 'isPersonal',
                  id: 'isPersonal'
                }}
              />
            }
            label='Personal token'
          />

          <Button
            variant='contained'
            color='primary'
            className={classes.addTokenButton}
            disabled={tokenRequest && tokenRequest.isProcessing}
            onClick={this.addToken}
          >
            Add token
          </Button>
        </div>
      </div>
    );

    let breadcrumbs = (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: 'Access tokens' }]}
      />
    );

    if (this.state && this.state.data && this.state.data.userTokens.error) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <h2>Access tokens</h2>
          <Error/>
        </div>
      );
    }

    if (!data || (data && data.isProcessing) || (data && data.orgId !== orgId)) {
      return (
        <div className={classes.root}>
          {breadcrumbs}
          {pageTitle}

          <PageSpinner />
        </div>
      );
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {tokenDisplay}

        {tokenForm}

        <div className={classes.remark}>
          Users may manage their personal tokens only. Admins may manage their
          &nbsp;personal tokens, as well as administrative (impersonal) tokens
          &nbsp;used to organize infrastructure. Tokens of all types work in
          &nbsp;the context of a particular organization.
        </div>

        <br/>
        <h2>Active access tokens</h2>

        {data.data.length > 0 ? (
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell>Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Creator</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>Expires</TableCell>
                  <TableCell align='right'>Actions</TableCell>
                </TableRow>
              </TableHead>

              <TableBody>
                {data.data.map(t => {
                  return (
                    <TableRow className={classes.row} key={t.id}>
                      <TableCell className={classes.cell}>{t.name}</TableCell>
                      <TableCell className={classes.cell}>
                        {t.is_personal ? 'Personal' : 'Administrative'}
                      </TableCell>
                      <TableCell className={classes.cell}>{t.username}</TableCell>
                      <TableCell className={classes.cell}>{t.created_formatted}</TableCell>
                      <TableCell className={classes.cell}>{t.expires_formatted}</TableCell>
                      <TableCell className={classes.cell} align='right'>
                        <Button
                          variant='outlined'
                          color='secondary'
                          className={classes.revokeButton}
                          disabled={(!t.is_personal &&
                            !orgPermissions.settingsTokenRevokeImpersonal) || t.revoking}
                          onClick={event => this.revokeToken(event, t.id, t.name)}
                        >
                          Revoke
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>) : 'This user has no active access tokens'
        }

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

AccessTokens.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(AccessTokens);
