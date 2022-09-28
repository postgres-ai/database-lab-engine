/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import { NavLink } from 'react-router-dom';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableRow
} from '@material-ui/core';
import Typography from '@material-ui/core/Typography';
import ExpansionPanel from '@material-ui/core/ExpansionPanel';
import ExpansionPanelSummary from '@material-ui/core/ExpansionPanelSummary';
import ExpansionPanelDetails from '@material-ui/core/ExpansionPanelDetails';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import TextField from '@material-ui/core/TextField';
import { formatDistanceToNowStrict } from 'date-fns';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import Warning from './Warning';
import messages from '../assets/messages';
import format from '../utils/format';
import urls from '../utils/urls';
import Link from './Link';
import DbLabStatus from './DbLabStatus';
import dblabutils from '../utils/dblabutils';


const PAGE_SIZE = 20;

const getStyles = (theme) => ({
  root: {
    ...styles.root,
    flex: '1 1 100%',
    display: 'flex',
    flexDirection: 'column'
  },
  summary: {
    marginTop: 20,
    marginBottom: 20
  },
  paramTitle: {
    display: 'inline-block',
    width: 200
  },
  sectionHeader: {
    fontWeight: 600,
    display: 'block',
    paddingBottom: 10,
    marginBottom: 10,
    borderBottom: '1px solid ' + colors.consoleStroke
  },
  logContainer: {
    'backgroundColor': 'black',
    'color': 'white',
    'fontFamily': '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
    'fontSize': '13px',
    'maxHeight': 'calc(100vh - 120px)',
    'overflowY': 'auto',
    'width': '100%',
    '& > div': {
      overflowWrap: 'anywhere'
    }
  },
  artifactContainer: {
    backgroundColor: 'black',
    color: 'white',
    fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
    fontSize: '13px',
    maxHeight: 'calc(100vh - 300px)',
    width: '100%',
    whiteSpace: 'break-spaces',
    overflowWrap: 'anywhere',
    overflow: 'auto'
  },
  showMoreContainer: {
    marginTop: 20,
    textAlign: 'center'
  },
  link: {
    'color': colors.secondary2.main,
    '&:visited': {
      color: colors.secondary2.main
    },
    '&:hover': {
      color: colors.secondary2.main
    },
    '&:active': {
      color: colors.secondary2.main
    }
  },
  checkStatusColumn: {
    display: 'block',
    width: 80,
    marginTop: 10,
    height: 30,
    float: 'left'
  },
  checkDescriptionColumn: {
    display: 'inline-block'
  },
  checkDetails: {
    clear: 'both',
    display: 'block',
    color: colors.pgaiDarkGray
  },
  checkListItem: {
    marginBottom: 10,
    minHeight: 30
  },
  cfgListItem: {
    marginBottom: 5
  },
  expansionPanel: {
    marginTop: '5px!important',
    borderRadius: '0px!important'
  },
  expansionPanelSummary: {
    'display': 'inline-block',
    'padding': '5px',
    'paddingLeft': '12px',
    'minHeight': '30px',
    'lineHeight': '30px',
    'width': '100%',
    '& .MuiExpansionPanelSummary-content': {
      margin: '0px',
      display: 'inline-block'
    },
    '&.Mui-expanded': {
      minHeight: '22px'
    },
    '& .MuiExpansionPanelSummary-expandIcon': {
      display: 'inline-block',
      padding: '0px',
      marginTop: '-1px'
    }
  },
  expansionPanelDetails: {
    padding: '12px',
    paddingTop: '0px',
    [theme.breakpoints.down('md')]: {
      display: 'block'
    }
  },
  intervalsRow: {
    'borderBottom': '1px solid ' + colors.consoleStroke,
    'width': '100%',
    'lineHeight': '22px',
    '&:last-child': {
      borderBottom: 'none'
    }
  },
  intervalIcon: {
    display: 'inline-block',
    width: 25
  },
  intervalStarted: {
    display: 'inline-block',
    width: 200
  },
  intervalDuration: {
    display: 'inline-block'
  },
  intervalWarning: {
    display: 'inline-block',
    width: '100%'
  },
  code: {
    'width': '100%',
    'margin-top': 0,
    '& > div': {
      paddingTop: 8,
      padding: 8
    },
    'background-color': 'rgb(246, 248, 250)',
    '& > div > textarea': {
      fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
      color: 'black',
      fontSize: '12px'
    }
  },
  button: {
    marginTop: 5,
    marginBottom: 10
  },
  bottomSpace: {
    ...styles.bottomSpace
  },
  artifactRow: {
    padding: '5px',
    cursor: 'pointer',
    [theme.breakpoints.down('sm')]: {
      paddingLeft: '0px',
      paddingRight: '0px',
      paddingTop: '10px'
    }
  },
  artifactName: {
    'display': 'inline-block',
    'width': '20%',
    [theme.breakpoints.down('sm')]: {
      display: 'block',
      width: '100%',
      marginBottom: '10px'
    },
    '& svg': {
      verticalAlign: 'middle'
    }
  },
  artifactDescription: {
    display: 'inline-block',
    width: '40%',
    [theme.breakpoints.down('sm')]: {
      display: 'block',
      width: '100%',
      marginBottom: '10px'
    }
  },
  artifactSize: {
    display: 'inline-block',
    width: '20%',
    [theme.breakpoints.down('sm')]: {
      display: 'block',
      width: '100%',
      marginBottom: '10px'
    }
  },
  artifactAction: {
    'display': 'inline-block',
    'width': '20%',
    'textAlign': 'right',
    '& button': {
      marginBottom: '5px'
    },
    [theme.breakpoints.down('sm')]: {
      display: 'block',
      width: '100%'
    }
  },
  artifactExpansionPanel: {
    padding: '0px!important',
    boxShadow: 'none'
  },
  artifactExpansionPanelSummary: {
    display: 'none',
    minHeight: '0px!important'
  },
  artifactsExpansionPanelDetails: {
    padding: '0px!important'
  },
  summaryDivider: {
    minHeight: '10px'
  },
  rotate180Icon: {
    '& svg': {
      transform: 'rotate(180deg)'
    }
  },
  rotate0Icon: {
    '& svg': {
      transform: 'rotate(0deg)'
    }
  }
});

class DbLabSession extends Component {
  componentDidMount() {
    let sessionId = this.props.match.params.sessionId;
    let that = this;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const dbLabSession = this.data && this.data.dbLabSession ?
        this.data.dbLabSession : null;

      if (auth && auth.token && !dbLabSession.isProcessing &&
        !that.state) {
        Actions.getDbLabSession(auth.token, sessionId);
      }

      if (auth && auth.token && !dbLabSession.isLogsProcessing &&
        !that.state) {
        Actions.getDbLabSessionLogs(auth.token, { sessionId, limit: PAGE_SIZE });
      }

      if (auth && auth.token && !dbLabSession.isArtifactsProcessing &&
        !that.state) {
        Actions.getDbLabSessionArtifacts(auth.token, sessionId);
      }

      that.setState({ data: this.data });

      let contentContainer = document.getElementById('logs-container');
      if (contentContainer && !contentContainer.getAttribute('onscroll')) {
        contentContainer.addEventListener('scroll', () => {
          if (contentContainer.scrollTop >=
            (contentContainer.scrollHeight - contentContainer.offsetHeight)) {
            if (that.refs.showMoreBtn) {
              that.refs.showMoreBtn.click();
            }
          }
        });
        contentContainer.setAttribute('onscroll', 1);
      }
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  showMore() {
    const sessionId = this.props.match.params.sessionId;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const session = this.state.data && this.state.data.dbLabSession ?
      this.state.data.dbLabSession : [];
    let lastId = null;

    if (session && session.logs && session.logs.length) {
      lastId = session.logs[session.logs.length - 1].id;
    }

    if (auth && auth.token && !session.isLogsProcessing && lastId) {
      Actions.getDbLabSessionLogs(
        auth.token,
        {
          sessionId,
          limit: PAGE_SIZE,
          lastId
        }
      );
    }
  }

  getCheckDetails(session, check) {
    switch (check) {
    case 'no_long_dangerous_locks':
      let intervals = null;
      let maxIntervals = null;
      if (session && session.result && session.result.summary &&
        session.result.summary.total_intervals) {
        intervals = session.result.summary.total_intervals;
      }
      if (session && session.config && session.config.observation_interval) {
        maxIntervals = session.config.observation_interval;
      }
      if (intervals && maxIntervals) {
        return '(' + intervals + ' ' +
          (intervals > 1 ? 'intervals' : 'interval') + ' with locks of ' +
          maxIntervals + ' allowed)';
      }
      break;

    case 'session_duration_acceptable':
      let totalDuration = null;
      let maxDuration = null;
      if (session && session.result && session.result.summary &&
        session.result.summary.total_duration) {
        totalDuration = session.result.summary.total_duration;
      }
      if (session && session.config && session.config.max_duration) {
        maxDuration = session.config.max_duration;
      }
      if (totalDuration && maxDuration) {
        return '(spent ' + format.formatSeconds(totalDuration, 0, '') + ' of the allowed ' +
          format.formatSeconds(maxDuration, 0, '') + ')';
      }
      break;

    default:
    }

    return '';
  }

  downloadLog = () => {
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const sessionId = this.props.match.params.sessionId;

    Actions.downloadDblabSessionLog(auth.token, sessionId);
  };

  downloadArtifact = (artifactType) => {
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const sessionId = this.props.match.params.sessionId;

    Actions.downloadDblabSessionArtifact(auth.token, sessionId, artifactType);
  };

  getArtifact = (artifactType) => {
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const sessionId = this.props.match.params.sessionId;

    Actions.getDbLabSessionArtifact(auth.token, sessionId, artifactType);
  };

  render() {
    const that = this;
    const { classes, orgPermissions } = this.props;
    const sessionId = this.props.match.params.sessionId;
    const data = this.state && this.state.data &&
      this.state.data.dbLabSession ? this.state.data.dbLabSession : null;
    const title = 'Database Lab observed session #' + sessionId;

    const pageTitle = (
      <ConsolePageTitle
        title={title}
        label={'Experimental'}
      />
    );
    const breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: 'Observed sessions', url: 'observed-sessions' },
          { name: title, url: null }
        ]}
      />
    );

    if (orgPermissions && !orgPermissions.dblabSessionView) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <Warning>{ messages.noPermissionPage }</Warning>
        </div>
      );
    }

    let errorWidget = null;
    if (this.state && this.state.data.dbLabSession.error) {
      errorWidget = (
        <Error
          message={this.state.data.dbLabSession.errorMessage}
          code={this.state.data.dbLabSession.errorCode}
        />
      );
    }

    if (this.state && (this.state.data.dbLabSession.error ||
      this.state.data.dbLabInstanceStatus.error)) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          { pageTitle }

          {errorWidget}
        </div>
      );
    }

    if (!data ||
      (this.state && this.state.data &&
      (this.state.data.dbLabSession.isProcessing))) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          { pageTitle }

          <PageSpinner />
        </div>
      );
    }

    const session = data && data.data ? data.data : null;
    const logs = data && data.logs ? data.logs : null;
    const artifacts = data && data.artifacts ? data.artifacts : null;
    const artifactData = data && data.artifactData ? data.artifactData : null;

    return (
      <div className={classes.root}>
        {breadcrumbs}

        { pageTitle }

        <div>
          <div className={classes.sectionHeader}>
            Summary
          </div>

          <Typography component='p'>
            <span className={classes.paramTitle}>Status:</span>
            <DbLabStatus session={{ status: session.result && session.result.status ?
              session.result.status : 'processing' }}/>
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component='p'>
            <span className={classes.paramTitle}>Session:</span>
            {session ? '#' + session.id : '-'}
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>Project:</span>
            {session.project_name ? session.project_name : '-'}
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>DLE instance:</span>
            {session.internal_instance_id ? (
              <NavLink
                to={urls.linkDbLabInstance(this.props, session.internal_instance_id)}
                className={classes.link}
              >
                {'#' + session.internal_instance_id}
              </NavLink>
            ) : ''}
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>DLE version:</span>
            {session && session.tags && session.tags.dle_version ?
              session.tags.dle_version : '-'}
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component='p'>
            <span className={classes.paramTitle}>Data state at:</span>
            {session && session.tags && session.tags.data_state_at ?
              session.tags.data_state_at : '-'}

          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component='p'>
            <span className={classes.paramTitle}>Duration:</span>
            {session && session.result && session.result.summary &&
              session.result.summary.elapsed ?
              session.result.summary.elapsed : null}
            {!(session && session.result && session.result.summary &&
              session.result.summary.elapsed) && session.duration > 0 ?
              format.formatSeconds(session.duration, 0, '') : null}
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>Created:</span>
            {
              session &&
              formatDistanceToNowStrict(new Date(session.started_at), { addSuffix: true })
            }
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component='p'>
            <span className={classes.paramTitle}>Branch:</span>
            {session && session.tags && session.tags.branch &&
              session.tags.branch_link ? (
                <Link
                  link={session.tags.branch_link}
                  target='_blank'
                >
                  {session.tags.branch}
                </Link>
              ) : (
                <span>
                  {session && session.tags && session.tags.branch ?
                    session.tags.branch : '-'}
                </span>
              )
            }
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>Commit:</span>
            {session && session.tags && session.tags.revision &&
              session.tags.revision_link ? (
                <Link
                  link={session.tags.revision_link}
                  target='_blank'
                >
                  {session.tags.revision}
                </Link>
              ) : (
                <span>
                  {session && session.tags && session.tags.revision ?
                    session.tags.revision : '-'}
                </span>
              )
            }
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>Triggered by:</span>
            {session && session.tags && session.tags.launched_by &&
              session.tags.username_link ? (
                <Link
                  link={session.tags.username_link}
                  target='_blank'
                >
                  {session.tags.launched_by}
                  {session.tags.username_full ?
                    ' (' + session.tags.username_full + ')' : ''}
                </Link>
              ) : (
                <span>
                  {session && session.tags && session.tags.launched_by ?
                    session.tags.launched_by : '-'}
                </span>
              )
            }
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>PR/MR:</span>
            {session && session.tags && session.tags.request_link ? (
              <Link
                link={session.tags.request_link}
                target='_blank'
              >
                {session.tags.request_link}
              </Link>) : '-'}
          </Typography>

          <Typography component='p'>
            <span className={classes.paramTitle}>Changes:</span>
            {session && session.tags && session.tags.request_link ? (
              <Link
                link={session.tags.request_link}
                target='_blank'
              >
                {session.tags.request_link}
              </Link>) : '-'}
          </Typography>

          {false && <Typography component='p' style={{ marginTop: 10 }}>
            Check documentation for the details about observed sessions:
            <Link
              link='https://postgres.ai/docs/guides/cloning/observe-sessions'
              target='_blank'
            >
              Database Lab – CI Observer
            </Link>
          </Typography>}
        </div>

        <br/>

        <div>
          <div className={classes.sectionHeader}>
            Checklist
          </div>

          {session.result && session.result.summary &&
            session.result.summary.checklist ? (
              <div>
                {Object.keys(session.result.summary.checklist).map(function (key) {
                  let details = that.getCheckDetails(session, key);
                  return (
                    <Typography component='div' className={classes.checkListItem}>
                      <div className={classes.checkStatusColumn}>
                        {session.result.summary.checklist[key] ?
                          (<DbLabStatus session={{ status: 'passed' }}/>) :
                          (<DbLabStatus session={{ status: 'failed' }}/>)
                        }
                      </div>
                      <div
                        className={classes.checkDescriptionColumn}
                        style={{ marginTop: !details ? 10 : 0 }}
                      >
                        {format.formatDbLabSessionCheck(key)}
                        <div className={classes.checkDetails}>
                          {details}
                        </div>
                      </div>
                    </Typography>
                  );
                })}
              </div>
            ) : (icons.processingLargeIcon)}
        </div>

        <br/>

        <div>
          <div className={classes.sectionHeader}>
            Observed intervals and details
          </div>
          <ExpansionPanel
            className={classes.expansionPanel}
            onChange={(event, expanded) => {
              that.setState({ intervalsExpanded: expanded });
            }}
          >
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls='panel1b-content'
              id='panel1b-header'
              className={classes.expansionPanelSummary}
            >
              {that.state.intervalsExpanded ? 'Hide intervals' : 'Show intervals'}
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.expansionPanelDetails}>
              {session.result && session.result.intervals &&
                session.result.intervals.length > 0 ? (
                  <div style={{ width: '100%' }}>
                    <div className={classes.intervalsRow}>
                      <div className={classes.intervalIcon}/>
                      <div className={classes.intervalStarted}>Started at</div>
                      <div className={classes.intervalDuration}>Duration</div>
                    </div>
                    {session.result.intervals.map(i => {
                      return (
                        <div className={classes.intervalsRow}>
                          <div>
                            <div className={classes.intervalIcon}>
                              {i.warning ?
                                icons.intervalWarning :
                                icons.intervalOk
                              }
                            </div>
                            <div className={classes.intervalStarted}>
                              {format.formatTimestampUtc(i.started_at)}
                            </div>
                            <div className={classes.intervalDuration}>
                              {format.formatSeconds(i.duration, 0, '')}
                            </div>
                          </div>
                          {i.warning &&
                            <div style={{ paddingLeft: 25 }}>
                              <div className={classes.intervalWarning}>
                                <TextField
                                  variant='outlined'
                                  id={'warning' + i.started_at}
                                  multiline
                                  fullWidth
                                  value={i.warning}
                                  className={classes.code}
                                  margin='normal'
                                  variant='outlined'
                                  InputProps={{
                                    readOnly: true
                                  }}
                                />
                              </div>
                            </div>
                          }
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <Typography component='p'>
                    <span className={classes.paramTitle}>Not specified</span>
                  </Typography>
                )}
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </div>

        <br/>

        <div>
          <div className={classes.sectionHeader}>
            Configuration
          </div>

          <ExpansionPanel
            className={classes.expansionPanel}
            onChange={(event, expanded) => {
              that.setState({ configurationExpanded: expanded });
            }}
          >
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls='panel1b-content'
              id='panel1b-header'
              className={classes.expansionPanelSummary}
            >
              {that.state.configurationExpanded ? 'Hide configuration' :
                'Show  configuration'}
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.expansionPanelDetails}>
              {session.config ? (
                <div>
                  {Object.keys(session.config).map(function (key) {
                    return (
                      <Typography component='p' className={classes.cfgListItem}>
                        <span className={classes.paramTitle}>{key}:</span>
                        {session.config[key]}
                      </Typography>
                    );
                  })}
                </div>
              ) : (
                <Typography component='p'>
                  <span className={classes.paramTitle}>Not specified</span>
                </Typography>
              )}
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </div>

        <br/>

        <br/>

        <div>
          <div className={classes.sectionHeader} style={{ marginBottom: '0px' }}>
            Artifacts
          </div>

          {Array.isArray(artifacts) && artifacts.length ?
            (
              <HorizontalScrollContainer>
                <Table className={classes.table}>
                  <TableBody>
                    {artifacts.map(a => {
                      return (
                        <TableRow
                          className={classes.row}
                          key={a.dblab_session_id + '-' + a.artifact_type}
                          onClick={() => {
                            if (orgPermissions && !orgPermissions.dblabSessionArtifactsView) {
                              return;
                            }
                            let artifactsExpanded = that.state.artifactsExpanded || {};
                            artifactsExpanded[a.artifact_type] =
                              !artifactsExpanded[a.artifact_type];
                            that.setState({ artifactsExpanded: artifactsExpanded });
                            if (artifactsExpanded[a.artifact_type] &&
                              a.artifact_type !== 'log' &&
                              (!data.artifactsData ||
                              data.artifactsData && !data.artifactsData[a.artifact_type])) {
                              this.getArtifact(a.artifact_type);
                            }
                          }}
                        >
                          <TableCell className={classes.artifactRow}>
                            <div>
                              <div className={classes.artifactName}>
                                {a.artifact_type}&nbsp;
                                {orgPermissions && orgPermissions.dblabSessionArtifactsView &&
                                <span
                                  className={
                                    that.state &&
                                    typeof that.state.artifactsExpanded !== 'undefined' &&
                                    that.state.artifactsExpanded[a.artifact_type] === true ?
                                      classes.rotate180Icon : classes.rotate0Icon
                                  }
                                >
                                  {icons.detailsArrow}
                                </span>}
                              </div>
                              <div className={classes.artifactDescription}>
                                {dblabutils.getArtifactDescription(a.artifact_type)}
                              </div>
                              <div className={classes.artifactSize}>
                                {format.formatBytes(a.artifact_size, 0, true)}
                              </div>
                              {orgPermissions && orgPermissions.dblabSessionArtifactsView &&
                              <div className={classes.artifactAction}>
                                <Button
                                  variant='outlined'
                                  color='secondary'
                                  className={ classes.button }
                                  disabled={ a.artifact_size < 10 }
                                  onClick={(event) => {
                                    event.stopPropagation();
                                    if (a.artifact_type === 'log') {
                                      this.downloadLog();
                                    } else {
                                      this.downloadArtifact(a.artifact_type);
                                    }
                                  }}
                                >
                                  {data && (
                                    (data.isArtifactDownloading &&
                                    data.downloadingArtifactType === a.artifact_type) ||
                                    (a.artifact_type === 'log' && data.isLogDownloading)
                                  ) ? (
                                      <span>
                                        &nbsp;
                                        <Spinner />
                                      </span>
                                    ) : (
                                      <span>
                                        {a.artifact_size < 10 ?
                                          icons.disabledDownloadIcon : icons.downloadIcon}
                                      </span>
                                    )
                                  }
                                  &nbsp;&nbsp;Download
                                </Button>
                              </div>}
                            </div>
                            <div>
                              <ExpansionPanel
                                square
                                className={classes.artifactExpansionPanel}
                                expanded={that.state &&
                                  typeof that.state.artifactsExpanded !== 'undefined' &&
                                  that.state.artifactsExpanded[a.artifact_type] === true}
                              >
                                <ExpansionPanelSummary
                                  expandIcon={<ExpandMoreIcon />}
                                  aria-controls='panel1b-content'
                                  id='panel1b-header'
                                  className={classes.artifactExpansionPanelSummary}
                                >
                                  {that.state.logsExpanded ? 'Hide log' : 'Show  log'}
                                </ExpansionPanelSummary>
                                <ExpansionPanelDetails
                                  className={classes.artifactsExpansionPanelDetails}>
                                  { a.artifact_type === 'log' ? <div>
                                    {Array.isArray(logs) && logs.length ? (
                                      <div id='logs-container' className={classes.logContainer}>
                                        {logs.map(r => {
                                          return (
                                            <div>
                                              {r.log_time},
                                              {r.user_name},
                                              {r.database_name},
                                              {r.process_id},
                                              {r.connection_from},
                                              {r.session_id},
                                              {r.session_line_num},
                                              {r.command_tag},
                                              {r.session_start_time},
                                              {r.virtual_transaction_id},
                                              {r.transaction_id},
                                              {r.error_severity},
                                              {r.sql_state_code},
                                              {r.message},
                                              {r.detail},
                                              {r.hint},
                                              {r.internal_query},
                                              {r.internal_query_pos},
                                              {r.context},
                                              {r.query},
                                              {r.query_pos},
                                              {r.location},
                                              {r.application_name},
                                              {r.backend_type}
                                            </div>
                                          );
                                        })}
                                        <div className={classes.showMoreContainer}>
                                          {data && data.isLogsProcessing &&
                                            <Spinner size='lg' className={classes.progress} />}
                                          {data && !data.isLogsProcessing && !data.isLogsComplete &&
                                            <Button
                                              ref='showMoreBtn'
                                              variant='outlined'
                                              color='secondary'
                                              className={classes.button}
                                              onClick={() => this.showMore()}
                                              disabled={data && data.isLogsProcessing}
                                            >
                                              Show more
                                            </Button>
                                          }
                                        </div>
                                      </div>
                                    ) : 'No log uploaded yet.'}
                                  </div> : <div>
                                    {artifactData && artifactData[a.artifact_type] &&
                                      artifactData[a.artifact_type].isProcessing ?
                                      <Spinner size='lg' className={classes.progress} /> : null}
                                    {artifactData && artifactData[a.artifact_type] &&
                                      artifactData[a.artifact_type].isProcessed &&
                                      artifactData[a.artifact_type].data ?
                                      <div
                                        id='artifact-container'
                                        className={classes.artifactContainer}
                                      >
                                        {artifactData[a.artifact_type].data}
                                      </div> : null}
                                  </div>}
                                </ExpansionPanelDetails>
                              </ExpansionPanel>
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </HorizontalScrollContainer>
            ) :
            (<span>
              {data.isArtifactsProcessed ? 'Artifacts not found' : ''}
              {data.isArtifactsProcessing ?
                <Spinner size='lg' className={classes.progress} /> : null}
            </span>)
          }
        </div>

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

DbLabSession.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(DbLabSession);
