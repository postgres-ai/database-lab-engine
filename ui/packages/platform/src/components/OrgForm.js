/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Grid from '@material-ui/core/Grid';
import Button from '@material-ui/core/Button';
import TextField from '@material-ui/core/TextField';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import Checkbox from '@material-ui/core/Checkbox';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import DeleteIcon from '@material-ui/icons/Delete';
import IconButton from '@material-ui/core/IconButton';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { styles } from '@postgres.ai/shared/styles/styles';

import Link from './Link';
import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import Aliases from '../utils/aliases';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import Warning from './Warning';
import messages from '../assets/messages';


const getStyles = () => ({
  container: {
    ...styles.root,
    'display': 'flex',
    'flex-wrap': 'wrap',
    'min-height': 0,
    '&:not(:first-child)': {
      'margin-top': '20px'
    }
  },
  textField: {
    ...styles.inputField,
    maxWidth: 450
  },
  onboardingField: {
    ...styles.inputField,
    'maxWidth': 450,
    '& #onboardingText-helper-text': {
      marginTop: 5,
      fontSize: '10px',
      letterSpacing: 0
    }
  },
  updateButtonContainer: {
    marginTop: 10,
    textAlign: 'left'
  },
  errorMessage: {
    color: 'red'
  },
  autoJoinCheckbox: {
    ...styles.checkbox,
    'margin-top': 5,
    'margin-bottom': 10,
    'margin-Left': 0,
    'width': 300
  },
  comment: {
    fontSize: 14,
    marginLeft: 0,
    width: '80%'
  },
  domainsHeader: {
    lineHeight: '42px'
  },
  editDomainField: {
    ...styles.inputField,
    'margin-left': 0,
    'margin-top': 10,
    'width': 'calc(80% - 80px)',
    '& > div.MuiFormControl- > label': {
      fontSize: 20
    },
    '& input, & .MuiOutlinedInput-multiline, & .MuiSelect-select': {
      padding: 13
    }
  },
  domainButton: {
    marginTop: 11,
    marginLeft: 10,
    width: 70
  },
  newDomainField: {
    ...styles.inputField,
    'margin-left': 0,
    'margin-top': 10,
    'width': '80%',
    '& > div.MuiFormControl- > label': {
      fontSize: 20
    },
    '& input, & .MuiOutlinedInput-multiline, & .MuiSelect-select': {
      padding: 13
    }
  },
  tableCell: {
    padding: 7,
    lineHeight: '16px',
    fontSize: 14
  },
  tableCellRight: {
    fontSize: 14,
    padding: 7,
    lineHeight: '16px',
    textAlign: 'right'
  },
  table: {
    width: '80%'
  },
  tableActionButton: {
    padding: 0,
    fontSize: 16
  },
  supportLink: {
    cursor: 'pointer',
    fontWeight: 'bold',
    textDecoration: 'underline'
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

class OrgForm extends Component {
  state = {
    id: null,
    name: '',
    alias: '',
    onboardingText: '',
    domain: '',
    orgChanged: true,
    usersAutojoin: false
  };

  componentDidMount() {
    const { orgId, mode } = this.props;
    const that = this;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const orgProfile = this.data && this.data.orgProfile ? this.data.orgProfile :
        null;

      that.setState({ data: this.data });

      if (auth && auth.token && orgProfile && orgProfile.orgId !== orgId &&
        !orgProfile.isProcessing) {
        Actions.getOrgs(auth.token, orgId);
      }

      if (!that.isAdd(mode) && orgProfile && orgProfile.data) {
        if (orgProfile.orgDomains && orgProfile.orgDomains.isProcessed) {
          that.setState({
            domain: ''
          });
        }

        if (orgProfile.data.updateName && orgProfile.data.updateAlias) {
          that.setState({
            id: orgProfile.orgId,
            name: orgProfile.data.updateName,
            alias: orgProfile.data.updateAlias,
            usersAutojoin: orgProfile.data.updateUsersAutojoin,
            onboardingText: orgProfile.data['onboarding_text'],
            orgChanged: false
          });
        } else {
          if (that.state.id && that.state.id === orgId) {
            return;
          }

          if (orgProfile.isUpdating || orgProfile.updateError) {
            return;
          }

          that.setState({
            id: orgProfile.data.id,
            name: orgProfile.data.name,
            alias: orgProfile.data.alias,
            usersAutojoin: orgProfile.data['users_autojoin'],
            onboardingText: orgProfile.data['onboarding_text'],
            orgChanged: false
          });
        }
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleChange = event => {
    let name = event.target.name;
    let value = event.target.value;

    if (name === 'alias') {
      value = Aliases.formatAlias(event.target.value);
    }

    if (name === 'usersAutojoin') {
      this.setState({
        [name]: event.target.checked,
        orgChanged: true
      });

      return;
    }

    this.setState({
      [name]: value,
      orgChanged: this.state.orgChanged || name !== 'domain'
    });

    if (name === 'name' && this.isAdd(this.props.mode)) {
      this.setState({ alias: Aliases.formatAlias(event.target.value) });
    }
  }

  buttonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.orgProfile : null;

    if (auth && data && !data.isUpdating && (this.state.name || this.state.alias)) {
      if (this.isAdd(this.props.mode)) {
        Actions.createOrg(auth.token, {
          name: this.state.name,
          alias: this.state.alias,
          email_domain_autojoin: this.state.domain,
          users_autojoin: this.state.usersAutojoin
        });
      } else if (this.state.orgChanged) {
        Actions.updateOrg(auth.token, orgId, {
          name: this.state.name,
          alias: this.state.alias,
          users_autojoin: this.state.usersAutojoin,
          onboarding_text: this.state.onboardingText
        });
      }
    }
  }

  domainButtonHandler = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const data = this.state.data ? this.state.data.orgProfile : null;

    if (auth && data && !data.isUpdating && !this.isAdd(this.props.mode) &&
      (this.state.name || this.state.alias)) {
      if (this.state.domain) {
        Actions.addOrgDomain(auth.token, orgId, this.state.domain);
      }
    }
  }

  deleteDomainHandler = (domainId, confirmed) => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;

    /* eslint no-alert: 0 */
    if (confirmed && window.confirm('Are you sure you want to delete already ' +
      'confirmed domain?') === false) {
      return;
    }

    Actions.deleteOrgDomain(auth.token, orgId, domainId);
  }

  isAdd = (mode) => {
    return mode === 'new';
  }

  render() {
    const { classes, orgPermissions, theme } = this.props;
    const orgId = this.props.orgId ? this.props.orgId : null;
    const data = this.state && this.state.data ? this.state.data.orgProfile : null;
    const mode = this.props.mode;
    const breadcrumbs = this.isAdd(mode) ? (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[
          { name: 'Organizations', url: '/' },
          { name: 'Create organization' }
        ]}
      />) : (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[
          { name: 'Settings' }
        ]}
      />);
    const pageTitle = this.isAdd(mode) ?
      (<ConsolePageTitle title='Create organization'/>) :
      (<ConsolePageTitle title='Settings'/>);

    if (orgPermissions && !orgPermissions.settingsOrganizationUpdate) {
      return (
        <>
          { breadcrumbs }

          { pageTitle }

          <Warning>{ messages.noPermissionPage }</Warning>
        </>
      );
    }

    let domains = (
      <div>
        No entries.
      </div>
    );

    let domainField = (
      <div>
        <TextField
          id='domain'
          label='Organization domain'
          variant='outlined'
          disabled={data && data.isUpdating}
          value={this.state.domain}
          className={!this.isAdd(mode) ? classes.editDomainField : classes.newDomainField}
          onChange={this.handleChange}
          error={data && data.updateErrorFields &&
            data.updateErrorFields.indexOf('domain') !== -1}
          margin='normal'
          fullWidth
          inputProps={{
            name: 'domain',
            id: 'domain',
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

        {!this.isAdd(mode) ? (
          <Button
            variant='contained'
            color='primary'
            disabled={data && data.orgDomains.isProcessing}
            onClick={this.domainButtonHandler}
            className={classes.domainButton}
          >
            Add
          </Button>
        ) : null }
      </div>
    );

    let buttonTitle = 'Save';
    if (this.isAdd(mode)) {
      buttonTitle = 'Create';
    }

    if (this.state && this.state.data && this.state.data.orgProfile.error &&
      this.state.data.orgProfile.orgId === orgId &&
      !this.state.data.orgProfile.isProcessing) {
      return (
        <div>
          <Error
            message={data.errorMessage}
            code={data.errorCode}
          />
        </div>
      );
    }

    if (!this.isAdd(mode) && (
      !data || !data.data || (data.data.id !== orgId) || data.data.isProcessing)
    ) {
      return (
        <>
          { breadcrumbs }
          { pageTitle }

          <PageSpinner />
        </>
      );
    }

    if (!this.isAdd(mode) && data.data.domains) {
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
                {Object.keys(data.data.domains).map(d => {
                  return (
                    <TableRow key={d}>
                      <TableCell className={classes.tableCell}>
                        {d}
                      </TableCell>
                      <TableCell className={classes.tableCell}>
                        {data.data.domains[d].confirmed ? 'Yes' : (
                          <span>
                            No&nbsp;&nbsp;
                            <span className={classes.errorMessage}>
                              To confirm&nbsp;
                              <span
                                className={window.Intercom ? classes.supportLink : null}
                                onClick={() => window.Intercom && window.Intercom('show')}
                              >
                                contact support
                              </span>
                            </span>
                          </span>
                        )}
                      </TableCell>
                      <TableCell className={classes.tableCellRight}>
                        {data && data.orgDomains && data.orgDomains.isDeleting &&
                          data.orgDomains.domainId === data.data.domains[d].id ? (
                            <Spinner className={classes.inTableProgress} />
                          ) : null }
                        <IconButton
                          className={classes.tableActionButton}
                          color='primary'
                          aria-label='Delete domain'
                          onClick={() => this.deleteDomainHandler(data.data.domains[d].id,
                            data.data.domains[d].confirmed)}
                          component='span'
                          disabled={data && data.orgDomains && data.orgDomains.isDeleting}
                        >
                          <DeleteIcon/>
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        </div>
      );
    }

    return (
      <>
        { breadcrumbs }

        { pageTitle }

        <div className={classes.errorMessage}>
          {data && data.updateErrorMessage ? data.updateErrorMessage : null}
        </div>

        <Grid container spacing={3}>
          <Grid item xs={12} sm={12} lg={12} className={classes.container} >
            <Grid
              xs={12}
              sm={12}
              lg={8}
              item
              xs
              container
              direction={theme.breakpoints.up('md') ? 'row' : 'column'}
            >
              <Grid item xs={12} sm={6}>
                <TextField
                  id='orgNameTextField'
                  label='Organization name'
                  variant='outlined'
                  disabled={data && data.isUpdating}
                  value={this.state.name}
                  className={classes.textField}
                  onChange={this.handleChange}
                  margin='normal'
                  error={data && data.updateErrorFields &&
                    data.updateErrorFields.indexOf('name') !== -1}
                  fullWidth
                  inputProps={{
                    name: 'name',
                    id: 'orgNameTextField',
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
                  id='orgAliasTextField'
                  label='Organization unique ID'
                  fullWidth
                  variant='outlined'
                  disabled={data && data.isUpdating}
                  value={this.state.alias}
                  className={classes.textField}
                  onChange={this.handleChange}
                  error={data && data.updateErrorFields &&
                    data.updateErrorFields.indexOf('alias') !== -1}
                  margin='normal'
                  inputProps={{
                    name: 'alias',
                    id: 'orgAliasTextField',
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

                { !this.isAdd(this.props.mode) ? (
                  <TextField
                    id='onboardingText'
                    label='Getting started text'
                    fullWidth
                    multiline
                    variant='outlined'
                    disabled={data && data.isUpdating}
                    value={this.state.onboardingText}
                    className={classes.onboardingField}
                    onChange={this.handleChange}
                    helperText={
                      <span>
                        Format:&nbsp;
                        <Link
                          link={'https://commonmark.org/help'}
                          target={'_blank'}
                        >
                          Markdown
                        </Link>
                      </span>
                    }
                    error={data && data.updateErrorFields &&
                      data.updateErrorFields.indexOf('onboardingText') !== -1}
                    margin='normal'
                    inputProps={{
                      name: 'onboardingText',
                      id: 'onboardingText',
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
                ) : null}

              </Grid>
            </Grid>

            { !this.isAdd(mode) && (
              <Grid item xs={12} sm={12} lg={8} className={classes.container} >
                <Grid item xs={12} sm={6}>
                  <FormControlLabel
                    className={classes.autoJoinCheckbox}
                    control={
                      <Checkbox
                        checked={ this.state.usersAutojoin }
                        onChange={ this.handleChange }
                        disabled={ data && data.isUpdating }
                        name='usersAutojoin'
                      />
                    }
                    label='Auto-join based on email domain'
                  />

                  <div>
                    <div className={classes.comment}>
                      Users that have email addresses with the following domain name(s)&nbsp;
                      automatically join the organization when registering in the system.<br/>
                    </div>

                    <br/>

                    {domains}

                    <br/>

                    {domainField}

                    <br/>
                  </div>

                </Grid>
              </Grid>
            )}

            <Grid item xs={12} sm={12} lg={8} className={classes.updateButtonContainer}>
              <Button
                variant='contained'
                color='primary'
                disabled={data && data.isUpdating}
                id='orgSaveButton'
                onClick={this.buttonHandler}
              >
                {buttonTitle}
              </Button>
            </Grid>
          </Grid>
        </Grid>

        <div className={classes.bottomSpace}/>
      </>
    );
  }
}

OrgForm.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(OrgForm);
