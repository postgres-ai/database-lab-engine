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
import FormControlLabel from '@material-ui/core/FormControlLabel';

import { styles } from '@postgres.ai/shared/styles/styles';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';

import Actions from '../actions/actions';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import Store from '../stores/store';
import Urls from '../utils/urls';
import Utils from '../utils/utils';
import Warning from './Warning';


const getStyles = () => ({
  textField: {
    ...styles.inputField,
    maxWidth: 400
  },
  errorMessage: {
    color: 'red'
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

class DbLabInstanceForm extends Component {
  state = {
    url: 'https://',
    token: '',
    useTunnel: false,
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
      const projects = this.data && this.data.projects ?
        this.data.projects : null;
      const dbLabInstances = this.data && this.data.dbLabInstances ?
        this.data.dbLabInstances : null;

      if (auth && auth.token && !projects.isProcessing && !projects.error &&
        !projects.isProcessed) {
        Actions.getProjects(auth.token, orgId);
      }

      if (auth && auth.token && !dbLabInstances.isProcessing &&
        !dbLabInstances.error && !dbLabInstances.isProcessed) {
        Actions.getDbLabInstances(auth.token, orgId, 0);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
    Actions.resetNewDbLabInstance();
  }

  handleChange = event => {
    const name = event.target.name;
    let value = event.target.value;

    if (name === 'useTunnel') {
      value = event.target.checked;
    }

    this.setState({
      [name]: value
    });

    Actions.resetNewDbLabInstance();
  };

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.newDbLabInstance : null;
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
      this.state.token && this.state.project) {
      Actions.addDbLabInstance(auth.token, {
        orgId: orgId,
        project: this.state.project,
        url: this.state.url,
        instanceToken: this.state.token,
        useTunnel: this.state.useTunnel,
        sshServerUrl: this.state.sshServerUrl
      });
    }
  };

  checkUrlHandler = () => {
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.newDbLabInstance : null;
    let errorFields = [];

    if (!this.state.url) {
      errorFields.push('url');
      return;
    }

    if (auth && data && !data.isChecking && this.state.url) {
      Actions.checkDbLabInstanceUrl(auth.token, this.state.url, this.state.token,
        this.state.useTunnel);
    }
  };

  returnHandler = () => {
    this.props.history.push(Urls.linkDbLabInstances(this.props));
  };

  processedHandler = () => {
    const data = this.state.data ? this.state.data.newDbLabInstance : null;

    this.props.history.push(Urls.linkDbLabInstance(this.props, data.data.id));
  };

  generateTokenHandler = () => {
    this.setState({ token: Utils.generateToken() });
  };

  render() {
    const { classes, orgPermissions } = this.props;
    const data = this.state && this.state.data ?
      this.state.data.newDbLabInstance : null;
    const projects = this.state && this.state.data &&
      this.state.data.projects ? this.state.data.projects : null;
    let projectsList = [];
    const dbLabInstances = this.state && this.state.data &&
      this.state.data.dbLabInstances ? this.state.data.dbLabInstances : null;

    if (data && data.isProcessed && !data.error) {
      this.processedHandler();
      Actions.resetNewDbLabInstance();
    }

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: 'Database Lab Instances', url: 'instances' },
          { name: 'Add instance' }
        ]}
      />
    );

    const pageTitle = (
      <ConsolePageTitle title='Add instance'/>
    );

    const permitted = !orgPermissions || orgPermissions.dblabInstanceCreate;

    const instancesLoaded = dbLabInstances && dbLabInstances.data;

    if (!projects || !projects.data || !instancesLoaded) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <PageSpinner className={classes.progress} />
        </div>
      );
    }

    if (projects.data) {
      projects.data.map(p => {
        projectsList.push({ title: p.name, value: p.id });
      });
    }

    const isDataUpdating = (data && (data.isUpdating || data.isChecking));

    return (
      <div className={classes.root}>
        { breadcrumbs }

        { pageTitle }

        { !permitted &&
          <Warning>
            You do not have permission to add Database Lab instances.
          </Warning>
        }

        <span>
          Database Lab provisioning is currently semi-automated.<br/>
          First, you need to prepare a Database Lab instance on a separate&nbsp;
          machine. Once the instance is ready, register it here.
        </span>

        <div className={ classes.errorMessage }>
          { data.errorMessage ? data.errorMessage : null }
        </div>

        <Grid container>
          <div className={ classes.fieldBlock }>
            <TextField
              disabled={ !permitted }
              variant='outlined'
              id='project'
              label='Project'
              value={ this.state.project }
              required
              className={ classes.textField }
              onChange={ this.handleChange }
              margin='normal'
              error={ this.state.errorFields.indexOf('project') !== -1 }
              fullWidth
              inputProps={ {
                name: 'project',
                id: 'project',
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
          </div>

          <div className={ classes.fieldBlock }>
            <TextField
              disabled={ !permitted }
              variant='outlined'
              id='token'
              label='Verification token'
              value={ this.state.token }
              required
              className={ classes.textField }
              onChange={ this.handleChange }
              margin='normal'
              error={ this.state.errorFields.indexOf('token') !== -1 }
              fullWidth
              inputProps={ {
                name: 'token',
                id: 'token',
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
            <div>
              <Button
                variant='contained'
                color='primary'
                disabled={ isDataUpdating || !permitted }
                onClick={ this.generateTokenHandler }
              >
                Generate
              </Button>
            </div>
          </div>

          <div className={ classes.fieldBlock } style={ { marginTop: 10 } }>
            <TextField
              disabled={ !permitted }
              variant='outlined'
              id='url'
              label='URL'
              required
              value={ this.state.url }
              className={ classes.textField }
              onChange={ this.handleChange }
              margin='normal'
              helperText={ !Utils.isHttps(this.state.url) && !this.state.useTunnel ? (
                <span>
                  <WarningIcon className={ classes.warningIcon }/>
                  <span className={ classes.warning }>
                    The connection to the Database Lab API is not secure. Use HTTPS.
                  </span>
                </span>) : null
              }
              error={ this.state.errorFields.indexOf('url') !== -1 }
              fullWidth
              inputProps={ {
                name: 'url',
                id: 'url',
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
          </div>

          <div className={ classes.fieldBlock }>
            <FormControlLabel
              control={
                <Checkbox
                  variant='outlined'
                  disabled={ !permitted }
                  checked={ this.state.useTunnel }
                  onChange={ this.handleChange }
                  id='useTunnel'
                  name='useTunnel'
                />
              }
              label='Use tunnel'
              labelPlacement='end'
            />
            <div>
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
            </div>
          </div>
          <div>
            <Button
              variant='contained'
              color='primary'
              disabled={ isDataUpdating || !permitted }
              onClick={ this.checkUrlHandler }
            >
              Verify URL
            </Button>

            { data.isCheckProcessed && data.isChecked &&
              (Utils.isHttps(this.state.url) || this.state.useTunnel) ? (
                <span className={ classes.urlOk }>
                  <CheckCircleOutlineIcon className={ classes.urlOkIcon }/> Verified
                </span>) : null }

            { data.isCheckProcessed && data.isChecked &&
              !Utils.isHttps(this.state.url) && !this.state.useTunnel ? (
                <span className={ classes.urlFail }>
                  <BlockIcon className={ classes.urlFailIcon }/> Verified but is not secure
                </span>) : null }

            { data.isCheckProcessed && !data.isChecked ? (
              <span className={ classes.urlFail }>
                <BlockIcon className={ classes.urlFailIcon }/> Not available
              </span>) : null }
          </div>

          <div className={ classes.fieldBlock } style={ { marginTop: 40 } }>
            <Button
              variant='contained'
              color='primary'
              disabled={ isDataUpdating || !permitted }
              onClick={ this.buttonHandler }
            >
              Add
            </Button>
            &nbsp;&nbsp;
            <Button
              variant='contained'
              color='primary'
              disabled={ isDataUpdating || !permitted }
              onClick={ this.returnHandler }
            >
              Cancel
            </Button>
          </div>
        </Grid>
      </div>
    );
  }
}

DbLabInstanceForm.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(DbLabInstanceForm);
