/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import {
  Typography,
  IconButton,
  TextField,
  Chip,
  Grid,
  Tabs,
  Tab,
  Button,
  Radio,
  RadioGroup,
  FormControlLabel,
  FormLabel,
  ExpansionPanel,
  ExpansionPanelSummary,
  ExpansionPanelDetails,
} from '@material-ui/core'
import Box from '@mui/material/Box'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import {
  ClassesType,
  TabPanelProps,
  ProjectProps,
  TokenRequestProps,
  RefluxTypes,
} from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import CfgGen, { DataType } from '../../utils/cfggen'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { CheckupAgentFormProps } from 'components/CheckupAgentForm/CheckupAgentFormWrapper'

const AUTO_GENERATED_TOKEN_NAME = 'Auto-generated 1-year token'

interface CheckupAgentFormWithStylesProps extends CheckupAgentFormProps {
  classes: ClassesType
}

interface CheckupAgentFormState extends DataType {
  tab: number
  data: {
    auth: {
      token: string
    } | null
    tokenRequest: TokenRequestProps
    reports: {
      error: boolean
      isProcessed: boolean
      isProcessing: boolean
    } | null
    projects: Omit<ProjectProps, 'data'>
  }
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props

  return (
    <Typography
      component="div"
      role="tabpanel"
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
          position: 'relative',
        }}
      >
        {children}
      </Box>
    </Typography>
  )
}

function a11yProps(index: number) {
  return {
    id: `simple-tab-${index}`,
    'aria-controls': `simple-tabpanel-${index}`,
  }
}

class CheckupAgentForm extends Component<
  CheckupAgentFormWithStylesProps,
  CheckupAgentFormState
> {
  state = {
    data: {
      auth: {
        token: '',
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
      reports: {
        error: false,
        isProcessed: false,
        isProcessing: false,
      },
      projects: {
        error: false,
        isProcessing: false,
        isProcessed: false,
      },
    },
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
    tab: 0,
  }

  unsubscribe: Function
  componentDidMount() {
    const that = this

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const auth: CheckupAgentFormState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const reports: CheckupAgentFormState['data']['reports'] =
        this.data && this.data.reports ? this.data.reports : null
      const projects: Omit<ProjectProps, 'data'> =
        this.data && this.data.projects ? this.data.projects : null
      const tokenRequest: TokenRequestProps =
        this.data && this.data.tokenRequest ? this.data.tokenRequest : null

      that.setState({ data: this.data })

      if (
        auth &&
        auth.token &&
        !reports?.isProcessed &&
        !reports?.isProcessing &&
        !reports?.error
      ) {
        Actions.getCheckupReports(auth.token)
      }

      if (
        auth &&
        auth.token &&
        !projects?.isProcessed &&
        !projects?.isProcessing &&
        !projects?.error
      ) {
        Actions.getProjects(auth.token)
      }

      if (
        tokenRequest &&
        tokenRequest.isProcessed &&
        !tokenRequest.error &&
        tokenRequest.data &&
        tokenRequest.data.name &&
        tokenRequest.data.name.startsWith(AUTO_GENERATED_TOKEN_NAME) &&
        tokenRequest.data.expires_at &&
        tokenRequest.data.token
      ) {
        that.setState({ apiToken: tokenRequest.data.token })
      }
    })

    Actions.refresh()
    CfgGen.generateRunCheckupSh(this.state)
  }

  componentWillUnmount() {
    Actions.hideGeneratedAccessToken()
    this.unsubscribe()
  }

  handleDeleteHost = (_: React.ChangeEvent<HTMLInputElement>, host: string) => {
    const curHosts = CfgGen.uniqueHosts(this.state.hosts)
    const curDividers = this.state.hosts.match(/[;,(\s)(\n)(\r)(\t)(\r\n)]/gm)
    const hosts = curHosts.split(';')
    let newHosts = ''

    for (const i in hosts) {
      if (hosts[i] !== host) {
        newHosts =
          newHosts +
          hosts[i] +
          (curDividers !== null && curDividers[i] ? curDividers[i] : '')
      }
    }

    this.setState({ hosts: newHosts })
  }

  handleChangeTab = (_: React.ChangeEvent<{}>, tabValue: number) => {
    this.setState({ tab: tabValue })
  }

  addToken = () => {
    const orgId = this.props.orgId ? this.props.orgId : null
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const tokenRequest =
      this.state.data && this.state.data.tokenRequest
        ? this.state.data.tokenRequest
        : null

    if (auth && auth.token && !tokenRequest?.isProcessing) {
      const date = new Date()
      const expiresAt =
        date.getFullYear() +
        1 +
        '-' +
        ('0' + (date.getMonth() + 1)).slice(-2) +
        '-' +
        ('0' + date.getDate()).slice(-2)
      const nowDateTime =
        date.getFullYear() +
        '-' +
        ('0' + (date.getMonth() + 1)).slice(-2) +
        '-' +
        ('0' + date.getDate()).slice(-2) +
        ' ' +
        ('0' + date.getHours()).slice(-2) +
        ':' +
        ('0' + date.getMinutes()).slice(-2)
      const tokenName = AUTO_GENERATED_TOKEN_NAME + ' (' + nowDateTime + ')'

      Actions.getAccessToken(auth.token, tokenName, expiresAt, orgId)
    }
  }

  copyDockerCfg = () => {
    const copyText = document.getElementById(
      'generatedDockerCfg',
    ) as HTMLInputElement

    if (copyText) {
      copyText.select()
      copyText.setSelectionRange(0, 99999)
      document.execCommand('copy')
      copyText.setSelectionRange(0, 0)
    }
  }

  copySrcCfg = () => {
    const copyText = document.getElementById(
      'generatedSrcCfg',
    ) as HTMLInputElement

    if (copyText) {
      copyText.select()
      copyText.setSelectionRange(0, 99999)
      document.execCommand('copy')
      copyText.setSelectionRange(0, 0)
    }
  }

  render() {
    const that = this
    const { classes } = this.props
    const reports =
      this.state.data && this.state.data.reports
        ? this.state.data.reports
        : null
    const projects =
      this.state.data && this.state.data.projects
        ? this.state.data.projects
        : null
    const tokenRequest =
      this.state.data && this.state.data.tokenRequest
        ? this.state.data.tokenRequest
        : null
    let copySrcCfgBtn = null
    let copyDockerCfgBtn = null
    let token = null
    let content = null

    if (
      this.state.projectName !== '' &&
      this.state.databaseName !== '' &&
      this.state.databaseUserName !== '' &&
      this.state.hosts !== '' &&
      this.state.apiToken !== ''
    ) {
      copySrcCfgBtn = (
        <IconButton
          className={classes.copyButton}
          aria-label="Copy"
          onClick={this.copySrcCfg}
        >
          {icons.copyIcon}
        </IconButton>
      )
      copyDockerCfgBtn = (
        <IconButton
          className={classes.copyButton}
          aria-label="Copy"
          onClick={this.copyDockerCfg}
        >
          {icons.copyIcon}
        </IconButton>
      )
    }

    token = (
      <div className={classes.relativeFieldBlock}>
        <TextField
          id="apiToken"
          variant="outlined"
          className={classes.tokenInput}
          margin="normal"
          required
          label="API access token"
          onChange={(e) => {
            this.setState({
              apiToken: e.target.value,
            })
          }}
          value={this.state.apiToken}
          helperText={
            'Insert a token or generate a new one. ' +
            'The auto-generated token will expire in 1 year.'
          }
          inputProps={{
            name: 'apiToken',
            id: 'apiToken',
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
          className={classes.addTokenButton}
          disabled={tokenRequest !== null && tokenRequest?.isProcessing}
          onClick={this.addToken}
        >
          Generate token
        </Button>
      </div>
    )

    if (
      this.state &&
      this.state.data &&
      ((this.state.data.reports && this.state.data.reports.error) ||
        (this.state.data.projects && this.state.data.projects.error))
    ) {
      return (
        <div>
          <ErrorWrapper />
        </div>
      )
    }

    if (reports && reports.isProcessed && projects?.isProcessed) {
      content = (
        <div style={{ marginTop: 20 }}>
          <span>
            Use postgres-checkup to check health of your Postgres databases.
            This page will help you to generate the configuration file. Provide
            settings that you will use inside your private network (local
            hostnames, IPs, etc).
          </span>
          <br />

          <span>
            Do not leave the page in order not to loose the configuration data.
          </span>
          <br />

          <h2>1. Configure</h2>

          <ExpansionPanel expanded={true}>
            <ExpansionPanelSummary
              aria-controls="panel1a-content"
              id="panel1a-header"
            >
              <Typography className={classes.heading}>
                General options
              </Typography>
            </ExpansionPanelSummary>

            <ExpansionPanelDetails>
              <Grid item xs={12} sm={12} md={12}>
                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="projectName"
                    label="Project name"
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        projectName: e.target.value,
                      })
                    }}
                    required
                    className={classes.textInput}
                    value={this.state.projectName}
                    inputProps={{
                      name: 'projectName',
                      id: 'projectName',
                      shrink: true,
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldRow} style={{ marginBottom: 20 }}>
                  <FormLabel component="legend" className={classes.legend}>
                    Connection type *
                  </FormLabel>
                  <RadioGroup
                    aria-label="connectionType"
                    name="connectionType"
                    value={this.state.connectionType}
                    onChange={(e) => {
                      this.setState({
                        connectionType: e.target.value,
                      })
                    }}
                    className={classes.radioButton}
                  >
                    <FormControlLabel
                      value="ssh"
                      control={<Radio />}
                      label="Connect to defined host via SSH"
                    />
                    <FormControlLabel
                      value="pg"
                      control={<Radio />}
                      label={
                        'Connect directly to PostgreSQL (some ' +
                        'reports won’t be available)'
                      }
                    />
                  </RadioGroup>
                </div>

                <div className={classes.fieldRow} style={{ marginBottom: 15 }}>
                  <div className={classes.fieldValue}>
                    <div className={classes.fieldBlock}>
                      <TextField
                        variant="outlined"
                        id="hostName"
                        className={classes.textInput}
                        helperText={
                          'Enter hostnames or IPs, separated by commas, spaces, or line breaks. ' +
                          'To reach your machine\'s "localhost" from the container, use "host.docker.internal". ' +
                          'The same approach works when using SSH port forwarding from your machine to a remote host.'
                        }
                        margin="normal"
                        onChange={(e) => {
                          this.setState({
                            hosts: e.target.value,
                          })
                        }}
                        value={this.state.hosts}
                        multiline
                        label="Hosts"
                        fullWidth
                        required
                        inputProps={{
                          name: 'hosts',
                          id: 'hosts',
                        }}
                        InputLabelProps={{
                          shrink: true,
                          style: styles.inputFieldLabel,
                        }}
                        FormHelperTextProps={{
                          style: styles.inputFieldHelper,
                        }}
                      />
                    </div>

                    <div>
                      {CfgGen.uniqueHosts(that.state.hosts)
                        .split(';')
                        .map((h) => {
                          if (h !== '') {
                            return (
                              <Chip
                                label={h}
                                className={classes.hostItem}
                                onDelete={(event) =>
                                  this.handleDeleteHost(event, h)
                                }
                                color="primary"
                              />
                            )
                          }

                          return null
                        })}
                    </div>
                  </div>
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="databaseUserName"
                    className={classes.textInput}
                    helperText={
                      'You can create a new DB user named "pgai_observer", then grant ' +
                      'the "pg_monitor" role to this user using the following SQL command: ' +
                      'GRANT pg_monitor TO pgai_observer;'
                    }
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        databaseUserName: e.target.value,
                      })
                    }}
                    required
                    value={this.state.databaseUserName}
                    label="Database username"
                    inputProps={{
                      name: 'databaseUserName',
                      id: 'databaseUserName',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldRow} style={{ marginBottom: 20 }}>
                  <FormLabel component="legend" className={classes.legend}>
                    Database password *
                  </FormLabel>
                  <RadioGroup
                    aria-label="password"
                    name="password"
                    value={this.state.password}
                    onChange={(e) => {
                      this.setState({
                        password: e.target.value,
                      })
                    }}
                    className={classes.radioButton}
                  >
                    <FormControlLabel
                      value="nopassword"
                      control={<Radio />}
                      label={
                        'No password is required or PGPASSWORD ' +
                        'environment variable is predefined'
                      }
                    />
                    <FormControlLabel
                      value="inputpassword"
                      control={<Radio />}
                      label={
                        'I will enter the password manually ' +
                        '(choose this only for manual testing)'
                      }
                    />
                  </RadioGroup>
                </div>

                <div
                  className={classes.fieldBlock}
                  style={{ marginBottom: 20 }}
                >
                  <TextField
                    variant="outlined"
                    id="databaseName"
                    label="Database name"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        databaseName: e.target.value,
                      })
                    }}
                    required
                    value={this.state.databaseName}
                    inputProps={{
                      name: 'databaseName',
                      id: 'databaseName',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="collectPeriod"
                    label="Snapshots period"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        collectPeriod: e.target.value,
                      })
                    }}
                    value={this.state.collectPeriod}
                    helperText={
                      'The delay between collection of two ' +
                      'statistics snapshots, sec'
                    }
                    type="number"
                    inputProps={{
                      name: 'collectPeriod',
                      id: 'collectPeriod',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>
              </Grid>
            </ExpansionPanelDetails>
          </ExpansionPanel>

          <ExpansionPanel>
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls="panel1b-content"
              id="panel1b-header"
              className={classes.advancedExpansionPanelSummary}
            >
              <Typography className={classes.heading}>
                Advanced options
              </Typography>
            </ExpansionPanelSummary>

            <ExpansionPanelDetails>
              <Grid item xs={12} sm={12} md={12}>
                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="ssDatabaseName"
                    className={classes.textInput}
                    label="pg_stat_statements DB name"
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        ssDatabaseName: e.target.value,
                      })
                    }}
                    value={this.state.ssDatabaseName}
                    helperText={
                      'Database name with enabled "pg_stat_statements"' +
                      ' extension (for detailed query analysis)'
                    }
                    inputProps={{
                      name: 'ssDatabaseName',
                      id: 'ssDatabaseName',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="pgPort"
                    label="Database port"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        pgPort: e.target.value,
                      })
                    }}
                    value={this.state.port}
                    helperText="PostgreSQL database server port (default: 5432)"
                    type="number"
                    inputProps={{
                      name: 'pgPort',
                      id: 'pgPort',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="sshPort"
                    label="SSH Port"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        sshPort: e.target.value,
                      })
                    }}
                    value={this.state.port}
                    helperText="SSH server port (default: 22)"
                    type="number"
                    inputProps={{
                      name: 'sshPort',
                      id: 'sshPort',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="statementTimeout"
                    label="Statement timeout"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        statementTimeout: e.target.value,
                      })
                    }}
                    value={this.state.statementTimeout}
                    helperText={
                      'Statement timeout for all SQL queries ' +
                      '(default: 30 seconds)'
                    }
                    type="number"
                    inputProps={{
                      name: 'statementTimeout',
                      id: 'statementTimeout',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="pgSocketDir"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        pgSocketDir: e.target.value,
                      })
                    }}
                    value={this.state.pgSocketDir}
                    label="PostgreSQL domain socket directory"
                    helperText="PostgreSQL domain socket directory (default: psql's default)"
                    inputProps={{
                      name: 'pgSocketDir',
                      id: 'pgSocketDir',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="psqlBinary"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        psqlBinary: e.target.value,
                      })
                    }}
                    value={this.state.psqlBinary}
                    helperText='Path to "psql" (default: determined by "$PATH")'
                    inputProps={{
                      name: 'psqlBinary',
                      id: 'psqlBinary',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>

                <div className={classes.fieldBlock}>
                  <TextField
                    variant="outlined"
                    id="sshKeysPath"
                    className={classes.textInput}
                    margin="normal"
                    onChange={(e) => {
                      this.setState({
                        sshKeysPath: e.target.value,
                      })
                    }}
                    value={this.state.sshKeysPath}
                    helperText="Path to directory with SSH keys"
                    inputProps={{
                      name: 'sshKeysPath',
                      id: 'sshKeysPath',
                    }}
                    fullWidth
                    InputLabelProps={{
                      shrink: true,
                      style: styles.inputFieldLabel,
                    }}
                    FormHelperTextProps={{
                      style: styles.inputFieldHelper,
                    }}
                  />
                </div>
              </Grid>
            </ExpansionPanelDetails>
          </ExpansionPanel>
          <br />
          <h2>
            2. Generate token to upload postgres-checkup reports to console
          </h2>
          <div>{token}</div>

          <h2>3. Deploy using Docker or building from source</h2>

          <Tabs
            value={this.state?.tab}
            onChange={this.handleChangeTab}
            aria-label="simple tabs example"
          >
            <Tab label="Docker" {...a11yProps(0)} />
            <Tab label="Build from source" {...a11yProps(1)} />
          </Tabs>

          <TabPanel value={this.state?.tab} index={1}>
            <Typography>
              <strong>Requirements</strong>: bash, coreutils, jq, golang, awk,
              sed, pandoc, wkhtmltopdf (see{' '}
              <a
                href={
                  'https://gitlab.com/postgres-ai/postgres-checkup/' +
                  'blob/master/README.md'
                }
                target="_blank"
                rel="noreferrer"
              >
                {' '}
                README{' '}
              </a>
              ).
              <br />
              <strong>Clone repo:</strong>
              <span className={classes.codeBlock}>
                git clone https://gitlab.com/postgres-ai/postgres-checkup.git &&
                cd postgres-checkup
              </span>
              <br />
              <strong>Start script below:</strong>
            </Typography>
            <div className={classes.relativeDiv}>
              <TextField
                variant="outlined"
                id="startWithSource"
                multiline
                value={CfgGen.generateFromSourcesInstruction(this.state)}
                className={classes.code}
                margin="normal"
                InputProps={{
                  readOnly: true,
                  id: 'generatedSrcCfg',
                }}
                error={!CfgGen.requiredParamsFilled(this.state)}
                helperText={
                  !CfgGen.requiredParamsFilled(this.state)
                    ? 'Please fill all mandatory fields'
                    : ''
                }
              />
              {copySrcCfgBtn}
            </div>
          </TabPanel>

          <TabPanel value={this.state?.tab} index={0}>
            <Typography>
              <strong>Requirements: </strong> Docker
              <br />
              <strong>Start script below:</strong>
            </Typography>
            <div className={classes.relativeDiv}>
              <TextField
                variant="outlined"
                id="configCode"
                multiline
                value={CfgGen.generateDockerInstruction(this.state)}
                className={classes.code}
                margin="normal"
                InputProps={{
                  readOnly: true,
                  id: 'generatedDockerCfg',
                }}
                error={!CfgGen.requiredParamsFilled(this.state)}
                helperText={
                  !CfgGen.requiredParamsFilled(this.state)
                    ? 'Please fill all mandatory fields'
                    : ''
                }
              />
              {copyDockerCfgBtn}
            </div>
          </TabPanel>
        </div>
      )
    } else {
      content = (
        <div>
          <Spinner className={classes.progress} />
        </div>
      )
    }

    return (
      <div className={classes.root}>
        {
          <ConsoleBreadcrumbsWrapper
            {...this.props}
            breadcrumbs={[{ name: 'Checkup configuration' }]}
          />
        }

        {content}
      </div>
    )
  }
}

export default CheckupAgentForm
