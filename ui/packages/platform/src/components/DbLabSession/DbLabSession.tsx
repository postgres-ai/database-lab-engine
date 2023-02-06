/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { NavLink } from 'react-router-dom'
import {
  Button,
  Table,
  TableBody,
  TableCell,
  TableRow,
  TextField,
  ExpansionPanelSummary,
  ExpansionPanelDetails,
  ExpansionPanel,
  Typography,
} from '@material-ui/core'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'
import { formatDistanceToNowStrict } from 'date-fns'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { icons } from '@postgres.ai/shared/styles/icons'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import ConsolePageTitle from './../ConsolePageTitle'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { messages } from '../../assets/messages'
import format from '../../utils/format'
import urls, { PropsType } from '../../utils/urls'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import dblabutils from '../../utils/dblabutils'
import { MomentInput } from 'moment'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DbLabStatusWrapper } from 'components/DbLabStatus/DbLabStatusWrapper'
import { DbLabSessionProps } from 'components/DbLabSession/DbLabSessionWrapper'

interface DbLabSessionWithStylesProps extends DbLabSessionProps {
  classes: ClassesType
}

interface Session {
  id: number
  project_name: string
  internal_instance_id: string | undefined
  duration: number
  started_at: string
  config: {
    [config: string | number]: number
  }
  tags: {
    request_link: string
    username_full: string
    username_link: string
    launched_by: string
    dle_version: string
    data_state_at: string
    branch: string
    branch_link: string
    revision: string
    revision_link: string
  }
  result: {
    status: string
    intervals: {
      warning: boolean
      started_at: MomentInput
      duration: number
    }[]
    summary: {
      elapsed: boolean
      total_intervals: number
      total_duration: number
      checklist: {
        [x: string]: { author_id: string }
      }
    }
  }
}

interface DbLabSessionState {
  logsExpanded: boolean
  artifactsExpanded: { [x: string]: boolean }
  configurationExpanded: boolean
  intervalsExpanded: boolean
  data: {
    dbLabInstanceStatus: {
      error: boolean
    }
    auth: {
      token: string | null
    } | null
    dbLabSession: {
      artifactsData: Record<string, unknown>
      isLogsComplete: boolean
      isArtifactsProcessed: boolean
      isArtifactsProcessing: boolean
      isLogDownloading: boolean
      downloadingArtifactType: string
      isArtifactDownloading: boolean
      artifacts:
        | {
            artifact_size: number
            artifact_type: string
            dblab_session_id: string
          }[]
        | null
      artifactData: {
        [data: string]: {
          isProcessing: boolean
          isProcessed: boolean
          data: string
        }
      } | null
      logs: {
        process_id: string
        connection_from: string
        session_id: string
        session_line_num: string
        command_tag: string
        session_start_time: string
        virtual_transaction_id: string
        transaction_id: string
        error_severity: string
        sql_state_code: string
        message: string
        detail: string
        hint: string
        internal_query: string
        internal_query_pos: string
        context: string
        query: string
        query_pos: string
        location: string
        application_name: string
        backend_type: string
        database_name: string
        user_name: string
        log_time: string
        id: number
      }[]
      error: boolean
      errorMessage: string
      errorCode: number
      isLogsProcessing: boolean
      isProcessing: boolean
      data: Session
    } | null
  } | null
}

const PAGE_SIZE = 20

class DbLabSession extends Component<
  DbLabSessionWithStylesProps,
  DbLabSessionState
> {
  unsubscribe: () => void
  componentDidMount() {
    const sessionId = this.props.match.params.sessionId
    const that = this

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const dbLabSession =
        this.data && this.data.dbLabSession ? this.data.dbLabSession : null

      if (auth && auth.token && !dbLabSession?.isProcessing && !that.state) {
        Actions.getDbLabSession(auth.token, sessionId)
      }

      if (
        auth &&
        auth.token &&
        !dbLabSession?.isLogsProcessing &&
        !that.state
      ) {
        Actions.getDbLabSessionLogs(auth.token, { sessionId, limit: PAGE_SIZE })
      }

      if (
        auth &&
        auth.token &&
        !dbLabSession?.isArtifactsProcessing &&
        !that.state
      ) {
        Actions.getDbLabSessionArtifacts(auth.token, sessionId)
      }

      that.setState({ data: this.data })

      const contentContainer = document.getElementById('logs-container')
      if (contentContainer && !contentContainer.getAttribute('onscroll')) {
        contentContainer.addEventListener('scroll', () => {
          if (
            contentContainer.scrollTop >=
            contentContainer.scrollHeight - contentContainer.offsetHeight
          ) {
            this.showMore()
          }
        })
        contentContainer.setAttribute('onscroll', '1')
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  showMore() {
    const sessionId = this.props.match.params.sessionId
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const session =
      this.state.data && this.state.data.dbLabSession
        ? this.state.data.dbLabSession
        : null
    let lastId = null

    if (session && session.logs && session.logs.length) {
      lastId = session.logs[session.logs.length - 1].id
    }

    if (auth && auth.token && !session?.isLogsProcessing && lastId) {
      Actions.getDbLabSessionLogs(auth.token, {
        sessionId,
        limit: PAGE_SIZE,
        lastId,
      })
    }
  }

  getCheckDetails(session: Session, check: string) {
    let intervals = null
    let maxIntervals = null
    let totalDuration = null
    let maxDuration = null
    switch (check) {
      case 'no_long_dangerous_locks':
        if (
          session &&
          session.result &&
          session.result.summary &&
          session.result.summary.total_intervals
        ) {
          intervals = session.result.summary.total_intervals
        }
        if (session && session.config && session.config.observation_interval) {
          maxIntervals = session.config.observation_interval
        }
        if (intervals && maxIntervals) {
          return (
            '(' +
            intervals +
            ' ' +
            (intervals > 1 ? 'intervals' : 'interval') +
            ' with locks of ' +
            maxIntervals +
            ' allowed)'
          )
        }
        break
      case 'session_duration_acceptable':
        if (
          session &&
          session.result &&
          session.result.summary &&
          session.result.summary.total_duration
        ) {
          totalDuration = session.result.summary.total_duration
        }
        if (session && session.config && session.config.max_duration) {
          maxDuration = session.config.max_duration
        }
        if (totalDuration && maxDuration) {
          return (
            '(spent ' +
            format.formatSeconds(totalDuration, 0, '') +
            ' of the allowed ' +
            format.formatSeconds(maxDuration, 0, '') +
            ')'
          )
        }
        break

      default:
    }

    return ''
  }

  downloadLog = () => {
    const auth =
      this.state && this.state.data && this.state.data.auth
        ? this.state.data.auth
        : null
    const sessionId = this.props.match.params.sessionId

    Actions.downloadDblabSessionLog(auth?.token, sessionId)
  }

  downloadArtifact = (artifactType: string) => {
    const auth =
      this.state && this.state.data && this.state.data.auth
        ? this.state.data.auth
        : null
    const sessionId = this.props.match.params.sessionId

    Actions.downloadDblabSessionArtifact(auth?.token, sessionId, artifactType)
  }

  getArtifact = (artifactType: string) => {
    const auth =
      this.state && this.state.data && this.state.data.auth
        ? this.state.data.auth
        : null
    const sessionId = this.props.match.params.sessionId

    Actions.getDbLabSessionArtifact(auth?.token, sessionId, artifactType)
  }

  render() {
    const that = this
    const { classes, orgPermissions } = this.props
    const sessionId = this.props.match.params.sessionId
    const data =
      this.state && this.state.data && this.state.data.dbLabSession
        ? this.state.data.dbLabSession
        : null
    const title = 'Database Lab observed session #' + sessionId

    const pageTitle = <ConsolePageTitle title={title} label={'Experimental'} />
    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[
          { name: 'Observed sessions', url: 'observed-sessions' },
          { name: title, url: null },
        ]}
      />
    )

    if (orgPermissions && !orgPermissions.dblabSessionView) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
        </div>
      )
    }

    let errorWidget = null
    if (this.state && this.state.data?.dbLabSession?.error) {
      errorWidget = (
        <ErrorWrapper
          message={this.state.data.dbLabSession.errorMessage}
          code={this.state.data.dbLabSession.errorCode}
        />
      )
    }

    if (
      this.state &&
      (this.state.data?.dbLabSession?.error ||
        this.state.data?.dbLabInstanceStatus?.error)
    ) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          {errorWidget}
        </div>
      )
    }

    if (
      !data ||
      (this.state &&
        this.state.data &&
        this.state.data?.dbLabSession?.isProcessing)
    ) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    const session = data && data.data ? data.data : null
    const logs = data && data.logs ? data.logs : null
    const artifacts = data && data.artifacts ? data.artifacts : null
    const artifactData = data && data.artifactData ? data.artifactData : null

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        <div>
          <div className={classes.sectionHeader}>Summary</div>

          <Typography component="p">
            <span className={classes.paramTitle}>Status:</span>
            <DbLabStatusWrapper
              session={{
                status:
                  session?.result && session.result.status
                    ? session.result.status
                    : 'processing',
              }}
            />
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component="p">
            <span className={classes.paramTitle}>Session:</span>
            {session ? '#' + session.id : '-'}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>Project:</span>
            {session?.project_name ? session.project_name : '-'}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>DLE instance:</span>
            {session?.internal_instance_id ? (
              <NavLink
                to={urls.linkDbLabInstance(
                  this.props as PropsType,
                  session.internal_instance_id,
                )}
                className={classes.link}
              >
                {'#' + session.internal_instance_id}
              </NavLink>
            ) : (
              ''
            )}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>DLE version:</span>
            {session && session.tags && session.tags.dle_version
              ? session.tags.dle_version
              : '-'}
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component="p">
            <span className={classes.paramTitle}>Data state at:</span>
            {session && session.tags && session.tags.data_state_at
              ? session.tags.data_state_at
              : '-'}
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component="p">
            <span className={classes.paramTitle}>Duration:</span>
            {session &&
            session.result &&
            session.result.summary &&
            session.result.summary.elapsed
              ? session.result.summary.elapsed
              : null}
            {!(
              session &&
              session.result &&
              session.result.summary &&
              session.result.summary.elapsed
            ) &&
            session &&
            session.duration > 0
              ? format.formatSeconds(session.duration, 0, '')
              : null}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>Created:</span>
            {session &&
              formatDistanceToNowStrict(new Date(session.started_at), {
                addSuffix: true,
              })}
          </Typography>

          <div className={classes.summaryDivider} />

          <Typography component="p">
            <span className={classes.paramTitle}>Branch:</span>
            {session &&
            session.tags &&
            session.tags.branch &&
            session.tags.branch_link ? (
              <GatewayLink href={session.tags.branch_link} target="_blank">
                {session.tags.branch}
              </GatewayLink>
            ) : (
              <span>
                {session && session.tags && session.tags.branch
                  ? session.tags.branch
                  : '-'}
              </span>
            )}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>Commit:</span>
            {session &&
            session.tags &&
            session.tags.revision &&
            session.tags.revision_link ? (
              <GatewayLink href={session.tags.revision_link} target="_blank">
                {session.tags.revision}
              </GatewayLink>
            ) : (
              <span>
                {session && session.tags && session.tags.revision
                  ? session.tags.revision
                  : '-'}
              </span>
            )}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>Triggered by:</span>
            {session &&
            session.tags &&
            session.tags.launched_by &&
            session.tags.username_link ? (
              <GatewayLink href={session.tags.username_link} target="_blank">
                {session.tags.launched_by}
                {session.tags.username_full
                  ? ' (' + session.tags.username_full + ')'
                  : ''}
              </GatewayLink>
            ) : (
              <span>
                {session && session.tags && session.tags.launched_by
                  ? session.tags.launched_by
                  : '-'}
              </span>
            )}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>PR/MR:</span>
            {session && session.tags && session.tags.request_link ? (
              <GatewayLink href={session.tags.request_link} target="_blank">
                {session.tags.request_link}
              </GatewayLink>
            ) : (
              '-'
            )}
          </Typography>

          <Typography component="p">
            <span className={classes.paramTitle}>Changes:</span>
            {session && session.tags && session.tags.request_link ? (
              <GatewayLink href={session.tags.request_link} target="_blank">
                {session.tags.request_link}
              </GatewayLink>
            ) : (
              '-'
            )}
          </Typography>

          {false && (
            <Typography component="p" style={{ marginTop: 10 }}>
              Check documentation for the details about observed sessions:
              <GatewayLink
                href="https://postgres.ai/docs/guides/cloning/observe-sessions"
                target="_blank"
              >
                Database Lab – CI Observer
              </GatewayLink>
            </Typography>
          )}
        </div>

        <br />

        <div>
          <div className={classes.sectionHeader}>Checklist</div>

          {session?.result &&
          session.result.summary &&
          session.result.summary.checklist ? (
            <div>
              {Object.keys(session.result.summary.checklist).map(function (
                key,
              ) {
                const details = that.getCheckDetails(session, key)
                return (
                  <Typography component="div" className={classes.checkListItem}>
                    <div className={classes.checkStatusColumn}>
                      {session.result.summary?.checklist &&
                      session.result.summary.checklist[key] ? (
                        <DbLabStatusWrapper session={{ status: 'passed' }} />
                      ) : (
                        <DbLabStatusWrapper session={{ status: 'failed' }} />
                      )}
                    </div>
                    <div
                      className={classes.checkDescriptionColumn}
                      style={{ marginTop: !details ? 10 : 0 }}
                    >
                      {format.formatDbLabSessionCheck(key)}
                      <div className={classes.checkDetails}>{details}</div>
                    </div>
                  </Typography>
                )
              })}
            </div>
          ) : (
            icons.processingLargeIcon
          )}
        </div>

        <br />

        <div>
          <div className={classes.sectionHeader}>
            Observed intervals and details
          </div>
          <ExpansionPanel
            className={classes.expansionPanel}
            onChange={(event, expanded) => {
              that.setState({ intervalsExpanded: expanded })
            }}
          >
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls="panel1b-content"
              id="panel1b-header"
              className={classes.expansionPanelSummary}
            >
              {that.state.intervalsExpanded
                ? 'Hide intervals'
                : 'Show intervals'}
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.expansionPanelDetails}>
              {session?.result &&
              session.result?.intervals &&
              session.result.intervals.length > 0 ? (
                <div style={{ width: '100%' }}>
                  <div className={classes.intervalsRow}>
                    <div className={classes.intervalIcon} />
                    <div className={classes.intervalStarted}>Started at</div>
                    <div className={classes.intervalDuration}>Duration</div>
                  </div>
                  {session.result.intervals.map((i) => {
                    return (
                      <div className={classes.intervalsRow}>
                        <div>
                          <div className={classes.intervalIcon}>
                            {i.warning
                              ? icons.intervalWarning
                              : icons.intervalOk}
                          </div>
                          <div className={classes.intervalStarted}>
                            {format.formatTimestampUtc(i.started_at)}
                          </div>
                          <div className={classes.intervalDuration}>
                            {format.formatSeconds(i.duration, 0, '')}
                          </div>
                        </div>
                        {i.warning && (
                          <div style={{ paddingLeft: 25 }}>
                            <div className={classes.intervalWarning}>
                              <TextField
                                variant="outlined"
                                id={'warning' + i.started_at}
                                multiline
                                fullWidth
                                value={i.warning}
                                className={classes.code}
                                margin="normal"
                                InputProps={{
                                  readOnly: true,
                                }}
                              />
                            </div>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              ) : (
                <Typography component="p">
                  <span className={classes.paramTitle}>Not specified</span>
                </Typography>
              )}
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </div>

        <br />

        <div>
          <div className={classes.sectionHeader}>Configuration</div>

          <ExpansionPanel
            className={classes.expansionPanel}
            onChange={(_, expanded) => {
              that.setState({ configurationExpanded: expanded })
            }}
          >
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon />}
              aria-controls="panel1b-content"
              id="panel1b-header"
              className={classes.expansionPanelSummary}
            >
              {that.state.configurationExpanded
                ? 'Hide configuration'
                : 'Show  configuration'}
            </ExpansionPanelSummary>
            <ExpansionPanelDetails className={classes.expansionPanelDetails}>
              {session?.config ? (
                <div>
                  {Object.keys(session.config).map(function (key) {
                    return (
                      <Typography component="p" className={classes.cfgListItem}>
                        <span className={classes.paramTitle}>{key}:</span>
                        {session.config && session.config[key]}
                      </Typography>
                    )
                  })}
                </div>
              ) : (
                <Typography component="p">
                  <span className={classes.paramTitle}>Not specified</span>
                </Typography>
              )}
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </div>

        <br />

        <br />

        <div>
          <div
            className={classes.sectionHeader}
            style={{ marginBottom: '0px' }}
          >
            Artifacts
          </div>

          {Array.isArray(artifacts) && artifacts.length ? (
            <HorizontalScrollContainer>
              <Table className={classes.table}>
                <TableBody>
                  {artifacts.map((a) => {
                    return (
                      <TableRow
                        className={classes.row}
                        key={a.dblab_session_id + '-' + a.artifact_type}
                        onClick={() => {
                          if (
                            orgPermissions &&
                            !orgPermissions.dblabSessionArtifactsView
                          ) {
                            return
                          }
                          const artifactsExpanded =
                            that.state.artifactsExpanded || {}
                          artifactsExpanded[a.artifact_type] =
                            !artifactsExpanded[a.artifact_type]
                          that.setState({
                            artifactsExpanded: artifactsExpanded,
                          })
                          if (
                            artifactsExpanded[a.artifact_type] &&
                            a.artifact_type !== 'log' &&
                            (!data.artifactsData ||
                              (data.artifactsData &&
                                !data.artifactsData[a.artifact_type]))
                          ) {
                            this.getArtifact(a.artifact_type)
                          }
                        }}
                      >
                        <TableCell className={classes.artifactRow}>
                          <div>
                            <div className={classes.artifactName}>
                              {a.artifact_type}&nbsp;
                              {orgPermissions &&
                                orgPermissions.dblabSessionArtifactsView && (
                                  <span
                                    className={
                                      that.state &&
                                      typeof that.state.artifactsExpanded !==
                                        'undefined' &&
                                      that.state.artifactsExpanded[
                                        a.artifact_type
                                      ] === true
                                        ? classes.rotate180Icon
                                        : classes.rotate0Icon
                                    }
                                  >
                                    {icons.detailsArrow}
                                  </span>
                                )}
                            </div>
                            <div className={classes.artifactDescription}>
                              {dblabutils.getArtifactDescription(
                                a.artifact_type,
                              )}
                            </div>
                            <div className={classes.artifactSize}>
                              {format.formatBytes(a.artifact_size, 0, true)}
                            </div>
                            {orgPermissions &&
                              orgPermissions.dblabSessionArtifactsView && (
                                <div className={classes.artifactAction}>
                                  <Button
                                    variant="outlined"
                                    color="secondary"
                                    className={classes.button}
                                    disabled={a.artifact_size < 10}
                                    onClick={(event) => {
                                      event.stopPropagation()
                                      if (a.artifact_type === 'log') {
                                        this.downloadLog()
                                      } else {
                                        this.downloadArtifact(a.artifact_type)
                                      }
                                    }}
                                  >
                                    {data &&
                                    ((data.isArtifactDownloading &&
                                      data.downloadingArtifactType ===
                                        a.artifact_type) ||
                                      (a.artifact_type === 'log' &&
                                        data.isLogDownloading)) ? (
                                      <span>
                                        &nbsp;
                                        <Spinner />
                                      </span>
                                    ) : (
                                      <span>
                                        {a.artifact_size < 10
                                          ? icons.disabledDownloadIcon
                                          : icons.downloadIcon}
                                      </span>
                                    )}
                                    &nbsp;&nbsp;Download
                                  </Button>
                                </div>
                              )}
                          </div>
                          <div>
                            <ExpansionPanel
                              square
                              className={classes.artifactExpansionPanel}
                              expanded={
                                that.state &&
                                typeof that.state.artifactsExpanded !==
                                  'undefined' &&
                                that.state.artifactsExpanded[
                                  a.artifact_type
                                ] === true
                              }
                            >
                              <ExpansionPanelSummary
                                expandIcon={<ExpandMoreIcon />}
                                aria-controls="panel1b-content"
                                id="panel1b-header"
                                className={
                                  classes.artifactExpansionPanelSummary
                                }
                              >
                                {that.state.logsExpanded
                                  ? 'Hide log'
                                  : 'Show  log'}
                              </ExpansionPanelSummary>
                              <ExpansionPanelDetails
                                className={
                                  classes.artifactsExpansionPanelDetails
                                }
                              >
                                {a.artifact_type === 'log' ? (
                                  <div>
                                    {Array.isArray(logs) && logs.length ? (
                                      <div
                                        id="logs-container"
                                        className={classes.logContainer}
                                      >
                                        {logs.map((r) => {
                                          return (
                                            <div>
                                              {r.log_time},{r.user_name},
                                              {r.database_name},{r.process_id},
                                              {r.connection_from},{r.session_id}
                                              ,{r.session_line_num},
                                              {r.command_tag},
                                              {r.session_start_time},
                                              {r.virtual_transaction_id},
                                              {r.transaction_id},
                                              {r.error_severity},
                                              {r.sql_state_code},{r.message},
                                              {r.detail},{r.hint},
                                              {r.internal_query},
                                              {r.internal_query_pos},{r.context}
                                              ,{r.query},{r.query_pos},
                                              {r.location},{r.application_name},
                                              {r.backend_type}
                                            </div>
                                          )
                                        })}
                                        <div
                                          className={classes.showMoreContainer}
                                        >
                                          {data && data.isLogsProcessing && (
                                            <Spinner
                                              size="lg"
                                              className={classes.progress}
                                            />
                                          )}
                                          {data &&
                                            !data.isLogsProcessing &&
                                            !data.isLogsComplete && (
                                              <Button
                                                variant="outlined"
                                                color="secondary"
                                                className={classes.button}
                                                onClick={() => this.showMore()}
                                                disabled={
                                                  data && data.isLogsProcessing
                                                }
                                              >
                                                Show more
                                              </Button>
                                            )}
                                        </div>
                                      </div>
                                    ) : (
                                      'No log uploaded yet.'
                                    )}
                                  </div>
                                ) : (
                                  <div>
                                    {artifactData &&
                                    artifactData[a.artifact_type] &&
                                    artifactData[a.artifact_type]
                                      .isProcessing ? (
                                      <Spinner
                                        size="lg"
                                        className={classes.progress}
                                      />
                                    ) : null}
                                    {artifactData &&
                                    artifactData[a.artifact_type] &&
                                    artifactData[a.artifact_type].isProcessed &&
                                    artifactData[a.artifact_type].data ? (
                                      <div
                                        id="artifact-container"
                                        className={classes.artifactContainer}
                                      >
                                        {artifactData[a.artifact_type].data}
                                      </div>
                                    ) : null}
                                  </div>
                                )}
                              </ExpansionPanelDetails>
                            </ExpansionPanel>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </HorizontalScrollContainer>
          ) : (
            <span>
              {data.isArtifactsProcessed ? 'Artifacts not found' : ''}
              {data.isArtifactsProcessing ? (
                <Spinner size="lg" className={classes.progress} />
              ) : null}
            </span>
          )}
        </div>

        <div className={classes.bottomSpace} />
      </div>
    )
  }
}

export default DbLabSession
