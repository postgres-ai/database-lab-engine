/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import { Checkbox } from '@material-ui/core';
import Grid from '@material-ui/core/Grid';
import Button from '@material-ui/core/Button';
import TextField from '@material-ui/core/TextField';
import CheckCircleOutlineIcon from '@material-ui/icons/CheckCircleOutline';
import BlockIcon from '@material-ui/icons/Block';
import WarningIcon from '@material-ui/icons/Warning';

import { styles } from '@postgres.ai/shared/styles/styles';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

import Store from '../stores/store';
import Actions from '../actions/actions';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import Urls from '../utils/urls';
import Utils from '../utils/utils';
import Warning from './Warning';
import FormControlLabel from '@material-ui/core/FormControlLabel';


const getStyles = () => ({
  textField: {
    ...styles.inputField,
    maxWidth: 400
  },
  errorMessage: {
    color: 'red',
    marginTop: 10
  },
  fieldBlock: {
    width: '100%'
  },
  urlOkIcon: {
    marginBottom: -5,
    marginLeft: 10,
    color: 'green'
  },
  urlOk: {
    color: 'green'
  },
  urlFailIcon: {
    marginBottom: -5,
    marginLeft: 10,
    color: 'red'
  },
  urlFail: {
    color: 'red'
  },
  warning: {
    color: '#801200',
    fontSize: '0.9em'
  },
  warningIcon: {
    color: '#801200',
    fontSize: '1.2em',
    position: 'relative',
    marginBottom: -3
  }
});

class JoeInstanceForm extends Component {
  state = {
    url: 'https://',
    useTunnel: false,
    token: '',
    project: this.props.project ? this.props.project : '',
    errorFields: [],
    sshServerUrl: ''
  };

  componentDidMount() {
    const that = this;
    const { orgId } = this.props;

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const joeInstances = this.data && this.data.joeInstances ?
        this.data.joeInstances : null;

      if (auth && auth.token && !joeInstances.isProcessing &&
        !joeInstances.error && !joeInstances.isProcessed) {
        Actions.getJoeInstances(auth.token, orgId, 0);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
    Actions.resetNewJoeInstance();
  }

  handleChange = event => {
    const name = event.target.name;
    const value = event.target.value;

    if (name === 'useTunnel') {
      this.setState({
        [name]: event.target.checked
      });

      return;
    }

    this.setState({
      [name]: value
    });

    Actions.resetNewJoeInstance();
  };

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.newJoeInstance : null;
    let errorFields = [];

    if (!this.state.url) {
      errorFields.push('url');
    }

    if (!this.state.project) {
      errorFields.push('project');
    }

    if (!this.state.token) {
      errorFields.push('token');
    }

    if (errorFields.length > 0) {
      this.setState({ errorFields: errorFields });
      return;
    }

    this.setState({ errorFields: [] });

    if (auth && data && !data.isUpdating && this.state.url &&
      this.state.token && this.state.token && this.state.project) {
      Actions.addJoeInstance(auth.token, {
        orgId: orgId,
        project: this.state.project,
        url: this.state.url,
        verifyToken: this.state.token,
        useTunnel: this.state.useTunnel,
        sshServerUrl: this.state.sshServerUrl
      });
    }
  };

  checkUrlHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.newJoeInstance : null;
    let errorFields = [];

    if (!this.state.url) {
      errorFields.push('url');
      return;
    }

    if (auth && data && !data.isChecking && this.state.url) {
      Actions.checkJoeInstanceUrl(auth.token, {
        orgId: orgId,
        project: this.state.project,
        url: this.state.url,
        verifyToken: this.state.token,
        useTunnel: this.state.useTunnel
      });
    }
  };

  returnHandler = () => {
    this.props.history.push(Urls.linkJoeInstances(this.props));
  };

  generateTokenHandler = () => {
    this.setState({ token: Utils.generateToken() });
  }

  render() {
    const { classes, orgPermissions } = this.props;
    const data = this.state && this.state.data ?
      this.state.data.newJoeInstance : null;

    if (data && data.isProcessed && !data.error) {
      this.returnHandler();
      Actions.resetNewJoeInstance();
    }

    const joeInstances = this.state && this.state.data &&
      this.state.data.joeInstances ? this.state.data.joeInstances : null;

    const permitted = !orgPermissions || orgPermissions.joeInstanceCreate;

    const instancesLoaded = joeInstances && joeInstances.data;

    if (!data || !instancesLoaded) {
      return (
        <div>
          <Spinner size='lg' className={classes.progress} />
        </div>
      );
    }

    const isDataUpdating = data && (data.isUpdating || data.isChecking);

    return (
      <div className={classes.root}>
        <ConsoleBreadcrumbs
          org={this.props.org}
          project={this.props.project}
          breadcrumbs={[
            { name: 'SQL Optimization', url: null },
            { name: 'Joe Instances', url: 'joe-instances' },
            { name: 'Add instance' }
          ]}
        />

        <ConsolePageTitle title='Add instance'/>

        { !permitted &&
          <Warning>
            You do not have permission to add Joe instances.
          </Warning>
        }

        <span>
          Joe provisioning is currently semi-automated.<br/>
          First, you need to prepare a Joe instance on a separate&nbsp;
          machine. Once the instance is ready, register it here.
        </span>

        <div className={classes.errorMessage}>
          {data.errorMessage ? data.errorMessage : null}
        </div>

        <Grid container>
          <div className={classes.fieldBlock}>
            <TextField
              variant='outlined'
              id='project'
              disabled={ !permitted }
              label='Project'
              value={this.state.project}
              required
              className={classes.textField}
              onChange={this.handleChange}
              margin='normal'
              error={this.state.errorFields.indexOf('project') !== -1}
              fullWidth
              inputProps={{
                name: 'project',
                id: 'project',
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
          </div>

          <div className={classes.fieldBlock}>
            <TextField
              variant='outlined'
              id='token'
              disabled={ !permitted }
              label='Signing secret'
              value={this.state.token}
              required
              className={classes.textField}
              onChange={this.handleChange}
              margin='normal'
              error={this.state.errorFields.indexOf('token') !== -1}
              fullWidth
              inputProps={{
                name: 'token',
                id: 'token',
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
            <div>
              <Button
                variant='contained'
                color='primary'
                disabled={ isDataUpdating || !permitted }
                onClick={this.generateTokenHandler}
              >
                Generate
              </Button>
            </div>
          </div>

          <div className={classes.fieldBlock} style={{ marginTop: 10 }}>
            <TextField
              variant='outlined'
              id='url'
              disabled={ !permitted }
              label='URL'
              required
              value={this.state.url}
              className={classes.textField}
              onChange={this.handleChange}
              margin='normal'
              helperText={!Utils.isHttps(this.state.url) && !this.state.useTunnel ? (
                <span>
                  <WarningIcon className={classes.warningIcon}/>
                  <span className={classes.warning}>
                    The connection to the Joe API is not secure. Use HTTPS.
                  </span>
                </span>) : null
              }
              error={this.state.errorFields.indexOf('url') !== -1}
              fullWidth
              inputProps={{
                name: 'url',
                id: 'url',
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

            <div className={classes.fieldBlock}>
              <FormControlLabel
                control={
                  <Checkbox
                    variant='outlined'
                    disabled={ !permitted }
                    checked={this.state.useTunnel}
                    onChange={this.handleChange}
                    id='useTunnel'
                    name='useTunnel'
                  />
                }
                label='Use tunnel'
                labelPlacement='end'
              />
              {this.state.useTunnel && <div>
                <TextField
                  variant='outlined'
                  id='token'
                  label='SSH server URL'
                  value={ this.state.sshServerUrl }
                  className={ classes.textField }
                  onChange={ this.handleChange }
                  margin='normal'
                  error={ this.state.errorFields.indexOf('sshServerUrl') !== -1 }
                  fullWidth
                  inputProps={ {
                    name: 'sshServerUrl',
                    id: 'sshServerUrl',
                    shrink: true
                  } }
                  InputLabelProps={ {
                    shrink: true,
                    style: styles.inputFieldLabel
                  } }
                  FormHelperTextProps={ {
                    style: styles.inputFieldHelper
                  } }
                />
              </div>}
            </div>
          </div>

          <div>
            <Button
              variant='contained'
              color='primary'
              disabled={isDataUpdating || !permitted}
              onClick={this.checkUrlHandler}
            >
              Verify URL
            </Button>

            {data.isCheckProcessed && data.isChecked &&
              (Utils.isHttps(this.state.url) || this.state.useTunnel) ? (
                <span className={classes.urlOk}>
                  <CheckCircleOutlineIcon className={classes.urlOkIcon}/> Verified
                </span>) : null}

            {data.isCheckProcessed && data.isChecked &&
              !Utils.isHttps(this.state.url) && !this.state.useTunnel ? (
                <span className={classes.urlFail}>
                  <BlockIcon className={classes.urlFailIcon}/> Verified but is not secure
                </span>) : null}

            {data.isCheckProcessed && !data.isChecked ? (
              <span className={classes.urlFail}>
                <BlockIcon className={classes.urlFailIcon}/> Not available
              </span>) : null}
          </div>

          <div className={classes.fieldBlock} style={{ marginTop: 40 }}>
            <Button
              variant='contained'
              color='primary'
              disabled={isDataUpdating || !permitted}
              onClick={this.buttonHandler}
            >
              Add
            </Button>
            &nbsp;&nbsp;
            <Button
              variant='contained'
              color='primary'
              disabled={isDataUpdating || !permitted}
              onClick={this.returnHandler}
            >
              Cancel
            </Button>
          </div>
        </Grid>

      </div>
    );
  }
}

JoeInstanceForm.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(JoeInstanceForm);
