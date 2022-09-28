/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import Button from '@material-ui/core/Button';
import Grid from '@material-ui/core/Grid';
import PropTypes from 'prop-types';
import TextField from '@material-ui/core/TextField';
import { withStyles } from '@material-ui/core/styles';

import { styles } from '@postgres.ai/shared/styles/styles';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';

import Actions from '../actions/actions';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import Error from './Error';
import Store from '../stores/store';
import Warning from './Warning';
import messages from '../assets/messages';


const getStyles = () => ({
  container: {
    display: 'flex',
    flexWrap: 'wrap'
  },
  textField: {
    ...styles.inputField,
    maxWidth: 400
  },
  dense: {
    marginTop: 10
  },
  errorMessage: {
    color: 'red'
  },
  button: {
    marginTop: 17,
    display: 'inline-block',
    marginLeft: 7
  }
});

class InviteForm extends Component {
  state = {
    email: ''
  };

  componentDidMount() {
    const that = this;
    const { org, orgId } = this.props;

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });

      if (this.data.inviteUser.isProcessed && !this.data.inviteUser.error) {
        that.props.history.push('/' + org + '/members');
      }

      const auth = this.data && this.data.auth ? this.data.auth : null;
      const orgProfile = this.data && this.data.orgProfile ?
        this.data.orgProfile : null;

      if (auth && auth.token && orgProfile &&
        orgProfile.orgId !== orgId && !orgProfile.isProcessing && !orgProfile.error) {
        Actions.getOrgs(auth.token, orgId);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleChange = event => {
    const name = event.target.name;
    const value = event.target.value;

    this.setState({
      [name]: value
    });
  };

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.inviteUser : null;

    if (auth && data && !data.isUpdating && this.state.email) {
      Actions.inviteUser(auth.token, orgId, this.state.email.trim());
    }
  };

  render() {
    const { classes, orgPermissions, theme } = this.props;

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[
          { name: 'Add user' }
        ]}
      />
    );

    const pageTitle = (
      <ConsolePageTitle
        title='Add a registered user'
        information='Use this form to add a registered users to your organization.'
      />
    );

    if (orgPermissions && !orgPermissions.settingsMemberAdd) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <Warning>{ messages.noPermissionPage }</Warning>
        </div>
      );
    }

    if (this.state && this.state.data && this.state.data.orgProfile.error) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <Error/>
        </div>
      );
    }

    if (!this.state || !this.state.data ||
      !(this.state.data.orgProfile && this.state.data.orgProfile.isProcessed)) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <PageSpinner />
        </div>
      );
    }

    const inviteData = this.state.data.inviteUser;

    return (
      <div className={classes.root}>
        { breadcrumbs }

        { pageTitle }

        <div>
            If the person is not registered yet, ask them to register first.
        </div>

        <div className={classes.errorMessage}>
          {inviteData && inviteData.errorMessage ? (
            <div className={classes.dense}>{inviteData.errorMessage}</div>
          ) : null}
        </div>

        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={8} className={classes.container} >
            <Grid
              item
              xs
              container
              direction={theme.breakpoints.up('md') ? 'row' : 'column'}
              spacing={2}
            >
              <Grid item xs={12} sm={6}>
                <TextField
                  id='email'
                  label='Email'
                  variant='outlined'
                  value={this.state.email}
                  className={classes.textField}
                  onChange={this.handleChange}
                  margin='normal'
                  error={inviteData && inviteData.updateErrorFields &&
                    inviteData.updateErrorFields.indexOf('email') !== -1}
                  fullWidth
                  inputProps={{
                    name: 'email',
                    id: 'email'
                  }}
                  InputLabelProps={{
                    shrink: true,
                    style: styles.inputFieldLabel
                  }}
                  FormHelperTextProps={{
                    style: styles.inputFieldHelper
                  }}
                />

                <Button
                  variant='contained'
                  color='primary'
                  disabled={inviteData && inviteData.isUpdating}
                  onClick={this.buttonHandler}
                  className={classes.button}
                >
                  Add
                </Button>
              </Grid>
            </Grid>
          </Grid>
        </Grid>
      </div>
    );
  }
}

InviteForm.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(InviteForm);
