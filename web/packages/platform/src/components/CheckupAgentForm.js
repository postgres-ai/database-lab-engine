/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Typography from '@material-ui/core/Typography';
import IconButton from '@material-ui/core/IconButton';
import TextField from '@material-ui/core/TextField';
import Chip from '@material-ui/core/Chip';
import Grid from '@material-ui/core/Grid';
import Tabs from '@material-ui/core/Tabs';
import Tab from '@material-ui/core/Tab';
import Box from '@material-ui/core/Box';
import Button from '@material-ui/core/Button';
import Radio from '@material-ui/core/Radio';
import RadioGroup from '@material-ui/core/RadioGroup';
import FormControlLabel from '@material-ui/core/FormControlLabel';
import FormLabel from '@material-ui/core/FormLabel';
import ExpansionPanel from '@material-ui/core/ExpansionPanel';
import ExpansionPanelSummary from '@material-ui/core/ExpansionPanelSummary';
import ExpansionPanelDetails from '@material-ui/core/ExpansionPanelDetails';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';
import { theme } from '@postgres.ai/shared/styles/theme';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import CfgGen from '../utils/cfggen';


const AUTO_GENERATED_TOKEN_NAME = 'Auto-generated 1-year token';

const getStyles = muiTheme => ({
  root: {
    'min-height': '100%',
    'z-index': 1,
    'position': 'relative',
    [muiTheme.breakpoints.down('sm')]: {
      maxWidth: '100vw'
    },
    [muiTheme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 200px)'
    },
    [muiTheme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 200px)'
    },
    '& h2': {
      ...theme.typography.h2
    },
    '& h3': {
      ...theme.typography.h3
    },
    '& h4': {
      ...theme.typography.h4
    },
    '& .MuiExpansionPanelSummary-root.Mui-expanded': {
      minHeight: 24
    }
  },
  heading: {
    ...theme.typography.h3
  },
  fieldValue: {
    display: 'inline-block',
    width: '100%'
  },
  tokenInput: {
    ...styles.inputField,
    'margin': 0,
    'margin-top': 10,
    'margin-bottom': 10
  },
  textInput: {
    ...styles.inputField,
    margin: 0,
    marginTop: 0,
    marginBottom: 10
  },
  hostItem: {
    marginRight: 10,
    marginBottom: 5,
    marginTop: 5
  },
  fieldRow: {
    marginBottom: 10,
    display: 'block'
  },
  fieldBlock: {
    'width': '100%',
    'max-width': 600,
    'margin-bottom': 15,
    '& > div.MuiFormControl- > label': {
      fontSize: 20
    },
    '& input, & .MuiOutlinedInput-multiline': {
      padding: 13
    }
  },
  relativeFieldBlock: {
    marginBottom: 10,
    marginRight: 20,
    position: 'relative'
  },
  addTokenButton: {
    marginLeft: 10,
    marginTop: 10
  },
  code: {
    'width': '100%',
    'margin-top': 0,
    '& > div': {
      paddingTop: 12
    },
    'background-color': 'rgb(246, 248, 250)',
    '& > div > textarea': {
      fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
      color: 'black',
      fontSize: 14
    }
  },
  codeBlock: {
    fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
      ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
    width: '100%',
    padding: 3,
    marginTop: 0,
    border: 'rgb(204, 204, 204);',
    borderRadius: 3,
    color: 'black',
    backgroundColor: 'rgb(246, 248, 250)'
  },
  details: {
    display: 'block'
  },
  copyButton: {
    position: 'absolute',
    top: 6,
    right: 6,
    fontSize: 20
  },
  relativeDiv: {
    position: 'relative'
  },
  radioButton: {
    '& > label > span.MuiFormControlLabel-label': {
      fontSize: '0.9rem'
    }
  },
  legend: {
    fontSize: '10px'
  },
  advancedExpansionPanelSummary: {
    'justify-content': 'left',
    '& div.MuiExpansionPanelSummary-content': {
      'flex-grow': 0
    }
  }
});

function TabPanel(props) {
  const { children, value, index, ...other } = props;

  return (
    <Typography
      component='div'
      role='tabpanel'
      hidden={value !== index}
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      style={{ marginTop: 0 }}
      {...other}
    >
      <Box
        p={3}
        style={{
          padding: 0,
          paddingTop: 10,
          position: 'relative'
        }}
      >
        {children}
      </Box>
    </Typography>
  );
}

TabPanel.propTypes = {
  children: PropTypes.node,
  index: PropTypes.any.isRequired,
  value: PropTypes.any.isRequired
};

function a11yProps(index) {
  return {
    'id': `simple-tab-${index}`,
    'aria-controls': `simple-tabpanel-${index}`
  };
}

class CheckupAgentForm extends Component {
  state = {
    hosts: '',
    projectName: '',
    databaseName: '',
    databaseUserName: '',
    ssDatabaseName: '',
    port: null,
    sshPort: null,
    pgPort: null,
    statementTimeout: null,
    pgSocketDir: '',
    psqlBinary: '',
    collectPeriod: 600,
    newHostName: '',
    apiToken: '',
    sshKeysPath: '',
    password: '',
    connectionType: '',
    tab: 0
  };

  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const reports = this.data && this.data.reports ?
        this.data.reports : null;
      const projects = this.data && this.data.projects ?
        this.data.projects : null;
      const tokenRequest = this.data && this.data.tokenRequest ?
        this.data.tokenRequest : null;

      that.setState({ data: this.data });

      if (auth && auth.token && !reports.isProcessed && !reports.isProcessing &&
        !reports.error) {
        Actions.getCheckupReports(auth.token);
      }

      if (auth && auth.token && !projects.isProcessed && !projects.isProcessing &&
        !projects.error) {
        Actions.getProjects(auth.token);
      }

      if (tokenRequest && tokenRequest.isProcessed && !tokenRequest.error &&
        tokenRequest.data && tokenRequest.data.name && tokenRequest.data.name
        .startsWith(AUTO_GENERATED_TOKEN_NAME) &&
        tokenRequest.data.expires_at && tokenRequest.data.token) {
        that.setState({ apiToken: tokenRequest.data.token });
      }
    });

    Actions.refresh();
    CfgGen.generateRunCheckupSh(this.state);
  }

  componentWillUnmount() {
    Actions.hideGeneratedAccessToken();
    this.unsubscribe();
  }

  handleClick = (event, id) => {
    this.props.history.push('/report/' + id);
  };

  linkClick = (event) => {
    this.props.history.push(event.target.getAttribute('hrefurl'));

    return false;
  };

  handleDeleteHost = (event, host) => {
    let curHosts = CfgGen.uniqueHosts(this.state.hosts);
    let curDividers = this.state.hosts.match(/[;,(\s)(\n)(\r)(\t)(\r\n)]/gm);
    let hosts = curHosts.split(';');
    let newHosts = '';

    for (let i in hosts) {
      if (hosts[i] !== host) {
        newHosts = newHosts + hosts[i] +
          (curDividers[i] ? curDividers[i] : '');
      }
    }

    this.setState({ hosts: newHosts });
  }

  handleChange = event => {
    const name = event.target.name;
    const value = event.target.value;

    this.setState({
      [name]: value
    });
  };

  handleChangeTab = (event, tabValue) => {
    this.setState({ tab: tabValue });
  }

  addToken = () => {
    const orgId = this.props.orgId ? this.props.orgId : null;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const tokenRequest = this.state.data && this.state.data.tokenRequest ?
      this.state.data.tokenRequest : null;

    if (auth && auth.token && !tokenRequest.isProcessing) {
      let date = new Date();
      let expiresAt = (date.getFullYear() + 1) + '-' + ('0' + (date.getMonth() +
        1)).slice(-2) + '-' + ('0' + date.getDate()).slice(-2);
      let nowDateTime = date.getFullYear() + '-' + ('0' + (date.getMonth() +
          1)).slice(-2) + '-' + ('0' + date.getDate()).slice(-2) +
        ' ' + ('0' + date.getHours()).slice(-2) + ':' + ('0' + date.getMinutes())
        .slice(-2);
      let tokenName = AUTO_GENERATED_TOKEN_NAME + ' (' + nowDateTime + ')';

      Actions.getAccessToken(auth.token, tokenName, expiresAt, orgId);
    }
  }

  copyDockerCfg = () => {
    let copyText = document.getElementById('generatedDockerCfg');

    copyText.select();
    copyText.setSelectionRange(0, 99999);
    document.execCommand('copy');
    copyText.setSelectionRange(0, 0);
  }

  copySrcCfg = () => {
    let copyText = document.getElementById('generatedSrcCfg');

    copyText.select();
    copyText.setSelectionRange(0, 99999);
    document.execCommand('copy');
    copyText.setSelectionRange(0, 0);
  }

  render() {
    const that = this;
    const { classes } = this.props;
    const reports = this.state.data && this.state.data.reports ?
      this.state.data.reports : null;
    const projects = this.state.data && this.state.data.projects ?
      this.state.data.projects : null;
    const tokenRequest = this.state.data && this.state.data.tokenRequest ?
      this.state.data.tokenRequest : null;
    let copySrcCfgBtn = null;
    let copyDockerCfgBtn = null;
    let token = null;
    let content = null;

    if (this.state.projectName !== '' && this.state.databaseName !== '' &&
      this.state.databaseUserName !== '' && this.state.hosts !== '' &&
      this.state.apiToken !== '') {
      copySrcCfgBtn = (
        <IconButton
          className={classes.copyButton}
          aria-label='Copy'
          onClick={this.copySrcCfg}
        >
          {icons.copyIcon}
        </IconButton>
      );
      copyDockerCfgBtn = (
        <IconButton
          className={classes.copyButton}
          aria-label='Copy'
          onClick={this.copyDockerCfg}
        >
          {icons.copyIcon}
        </IconButton>
      );
    }

    token = (
      <div className={classes.relativeFieldBlock}>
        <TextField
          id='apiToken'
          variant='outlined'
          className={classes.tokenInput}
          margin='normal'
          required
          label='API access token'
          onChange={this.handleChange}
          value={this.state.apiToken}
          helperText={'Insert a token or generate a new one. ' +
            'The auto-generated token will expire in 1 year.'}
          inputProps={{
            name: 'apiToken',
            id: 'apiToken'
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
          className={classes.addTokenButton}
          disabled={ tokenRequest && tokenRequest.isProcessing }
          onClick={this.addToken}
        >
          Generate token
        </Button>
      </div>
    );

    if (this.state && this.state.data && (this.state.data.reports.error ||
        this.state.data.projects.error)) {
      return (
        <div>
          <Error/>
        </div>
      );
    }

    if (reports && reports.isProcessed && projects.isProcessed) {
      content = (
        <div style={{ marginTop: 20 }}>
          <span>Use postgres-checkup to check health of your Postgres databases.
            This page will help you to generate the configuration file.
            Provide settings that you will use inside your private network
            (local hostnames, IPs, etc).
          </span>
          <br/>

          <span>
            Do not leave the page in order not to loose the configuration data.
          </span>
          <br/>

          <h2>1. Configure</h2>

          <ExpansionPanel expanded={true}>
            <ExpansionPanelSummary
              aria-controls='panel1a-content'
              id='panel1a-header'
            >
              <Typography className={classes.heading}>
                General options
              </Typography>
            </ExpansionPanelSummary>

            <ExpansionPanelDetails>
              <Grid item xs={12} sm={12} md={12}>
                <div className={classes.fieldBlock}>
                  <TextField
                    variant='outlined'
                    id='projectName'
                    label='Project name'
                    margin='normal'
                    onChange={this.handleChange}
                    required
                    className={classes.textInput}
                    value={this.state.projectName}
                    inputProps={{
                      name: 'projectName',
                      id: 'projectName',
                      shrink: true
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper
                    }}
                  />
                </div>

                <div className={classes.fieldRow} style={{ marginBottom: 20 }}>
                  <FormLabel component='legend' className={classes.legend}>
                    Connection type *
                  </FormLabel>
                  <RadioGroup
                    aria-label='connectionType'
                    name='connectionType'
                    value={this.state.connectionType}
                    onChange={this.handleChange}
                    className={classes.radioButton}
                  >
                    <FormControlLabel
                      value='ssh'
                      control={<Radio />}
                      label='Connect to defined host via SSH'
                    />
                    <FormControlLabel
                      value='pg'
                      control={<Radio />}
                      label={'Connect directly to PostgreSQL (some ' +
                      'reports won’t be available)'}
                    />
                  </RadioGroup>
                </div>

                <div className={classes.fieldRow} style={{ marginBottom: 15 }}>
                  <div className={classes.fieldValue}>
                    <div className={classes.fieldBlock}>
                      <TextField
                        variant='outlined'
                        id='hostName'
                        className={classes.textInput}
                        helperText={'Hostname(s) or IP address(es) divided ' +
                          'by `;`, `,`, space or end of string'}
                        margin='normal'
                        onChange={this.handleChange}
                        value={this.state.hosts}
                        multiline
                        label='Hosts'
                        fullWidth
                        required
                        inputProps={{
                          name: 'hosts',
                          id: 'hosts'
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

                    <div>
                      { CfgGen.uniqueHosts(that.state.hosts).split(';').map(h => {
                        if (h !== '') {
                          return (
                            <Chip
                              label={h}
                              className={classes.hostItem}
                              onDelete={event => this.handleDeleteHost(event, h)}
                              color='primary'
                            />
                          );
                        }

                        return null;
                      })}
                    </div>
                  </div>
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant='outlined'
                    id='databaseUserName'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    required
                    value={this.state.databaseUserName}
                    label='Database username'
                    inputProps={{
                      name: 'databaseUserName',
                      id: 'databaseUserName'
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper
                    }}
                  />
                </div>

                <div className={classes.fieldRow} style={{ marginBottom: 20 }}>
                  <FormLabel component='legend' className={classes.legend}>
                    Database password *
                  </FormLabel>
                  <RadioGroup
                    aria-label='password'
                    name='password'
                    value={this.state.password}
                    onChange={this.handleChange}
                    className={classes.radioButton}
                  >
                    <FormControlLabel
                      value='nopassword'
                      control={<Radio />}
                      label={'No password is required or PGPASSWORD ' +
                        'environment variable is predefined'}
                    />
                    <FormControlLabel
                      value='inputpassword'
                      control={<Radio />}
                      label={'I will enter the password manually ' +
                        '(choose this only for manual testing)'}
                    />
                  </RadioGroup>
                </div>

                <div className={classes.fieldBlock} style={{ marginBottom: 20 }}>
                  <TextField
                    variant='outlined'
                    id='databaseName'
                    label='Database name'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    required
                    value={this.state.databaseName}
                    inputProps={{
                      name: 'databaseName',
                      id: 'databaseName'
                    }}
                    fullWidth
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
                    id='collectPeriod'
                    label='Snapshots period'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.collectPeriod}
                    helperText={'The delay between collection of two ' +
                      'statistics snapshots, sec'}
                    type='number'
                    inputProps={{
                      name: 'collectPeriod',
                      id: 'collectPeriod'
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper
                    }}
                  />
                </div>
              </Grid>
            </ExpansionPanelDetails>
          </ExpansionPanel>

          <ExpansionPanel>
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon/>}
              aria-controls='panel1b-content'
              id='panel1b-header'
              className={classes.advancedExpansionPanelSummary}
            >
              <Typography className={classes.heading}>Advanced options</Typography>
            </ExpansionPanelSummary>

            <ExpansionPanelDetails>
              <Grid item xs={12} sm={12} md={12}>
                <div className={classes.fieldBlock}>
                  <TextField
                    variant='outlined'
                    id='ssDatabaseName'
                    className={classes.textInput}
                    label='pg_stat_statements DB name'
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.ssDatabaseName}
                    helperText={'Database name with enabled "pg_stat_statements"' +
                      ' extension (for detailed query analysis)'}
                    inputProps={{
                      name: 'ssDatabaseName',
                      id: 'ssDatabaseName'
                    }}
                    fullWidth
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
                    id='pgPort'
                    label='Database port'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.port}
                    helperText='PostgreSQL database server port (default: 5432)'
                    type='number'
                    inputProps={{
                      name: 'pgPort',
                      id: 'pgPort'
                    }}
                    fullWidth
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
                    id='sshPort'
                    label='SSH Port'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.port}
                    helperText='SSH server port (default: 22)'
                    type='number'
                    inputProps={{
                      name: 'sshPort',
                      id: 'sshPort'
                    }}
                    fullWidth
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
                    id='statementTimeout'
                    label='Statement timeout'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.statementTimeout}
                    helperText={'Statement timeout for all SQL queries ' +
                      '(default: 30 seconds)'}
                    type='number'
                    inputProps={{
                      name: 'statementTimeout',
                      id: 'statementTimeout'
                    }}
                    fullWidth
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
                    id='pgSocketDir'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.pgSocketDir}
                    label='PostgreSQL domain socket directory'
                    helperText="PostgreSQL domain socket directory (default: psql's default)"
                    inputProps={{
                      name: 'pgSocketDir',
                      id: 'pgSocketDir'
                    }}
                    fullWidth
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
                    id='psqlBinary'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.psqlBinary}
                    helperText='Path to "psql" (default: determined by "$PATH")'
                    inputProps={{
                      name: 'psqlBinary',
                      id: 'psqlBinary'
                    }}
                    fullWidth
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
                    id='sshKeysPath'
                    className={classes.textInput}
                    margin='normal'
                    onChange={this.handleChange}
                    value={this.state.sshKeysPath}
                    helperText='Path to directory with SSH keys'
                    inputProps={{
                      name: 'sshKeysPath',
                      id: 'sshKeysPath'
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper
                    }}
                  />
                </div>
              </Grid>
            </ExpansionPanelDetails>
          </ExpansionPanel>
          <br/>
          <h2>2. Generate token to upload postgres-checkup reports to console</h2>
          <div>
            {token}
          </div>

          <h2>3. Deploy using Docker or building from source</h2>

          <Tabs
            value={this.state.tab}
            onChange={this.handleChangeTab}
            aria-label='simple tabs example'
            inputProps={{
              name: 'tab',
              id: 'tab'
            }}
          >
            <Tab label='Docker' {...a11yProps(0)} />
            <Tab label='Build from source' {...a11yProps(1)} />
          </Tabs>

          <TabPanel value={this.state.tab} index={1}>
            <Typography>
              <strong>Requirements</strong>:
              bash, coreutils, jq, golang, awk, sed, pandoc, wkhtmltopdf
              (see <a href={'https://gitlab.com/postgres-ai/postgres-checkup/' +
                'blob/master/README.md'} target='_blank' > README </a>).
              <br/>
              <strong>Clone repo:</strong>
              <span className={classes.codeBlock}>
                git clone https://gitlab.com/postgres-ai/postgres-checkup.git
                && cd postgres-checkup
              </span><br/>
              <strong>Start script below:</strong>
            </Typography>
            <div className={classes.relativeDiv}>
              <TextField
                variant='outlined'
                id='startWithSource'
                multiline
                value={CfgGen.generateFromSourcesInstruction(this.state)}
                className={classes.code}
                margin='normal'
                variant='outlined'
                InputProps={{
                  readOnly: true,
                  id: 'generatedSrcCfg'
                }}
                error={!CfgGen.requiredParamsFilled(this.state)}
                helperText={!CfgGen.requiredParamsFilled(this.state) ?
                  'Please fill all mandatory fields' : ''}
              />
              {copySrcCfgBtn}
            </div>
          </TabPanel>

          <TabPanel value={this.state.tab} index={0}>
            <Typography>
              <strong>Requirements: </strong> Docker<br/>
              <strong>Start script below:</strong>
            </Typography>
            <div className={classes.relativeDiv}>
              <TextField
                variant='outlined'
                id='configCode'
                multiline
                value={CfgGen.generateDockerInstruction(this.state)}
                className={classes.code}
                margin='normal'
                variant='outlined'
                InputProps={{
                  readOnly: true,
                  id: 'generatedDockerCfg'
                }}
                error={!CfgGen.requiredParamsFilled(this.state)}
                helperText={!CfgGen.requiredParamsFilled(this.state) ?
                  'Please fill all mandatory fields' : ''}
              />
              {copyDockerCfgBtn}
            </div>
          </TabPanel>
        </div>
      );
    } else {
      content = (
        <div>
          <Spinner className={classes.progress} />
        </div>
      );
    }

    return (
      <div className={classes.root}>
        {<ConsoleBreadcrumbs
          {...this.props}
          breadcrumbs={[{ name: 'Checkup configuration' }]}
        />}

        {content}
      </div>
    );
  }
}

CheckupAgentForm.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(CheckupAgentForm);
