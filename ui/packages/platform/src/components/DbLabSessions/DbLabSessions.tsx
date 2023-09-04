/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component, MouseEvent } from 'react'
import { RouteComponentProps } from 'react-router'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Button,
} from '@material-ui/core'
import { formatDistanceToNowStrict } from 'date-fns'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { icons } from '@postgres.ai/shared/styles/icons'
import { isValidDate } from '@postgres.ai/shared/utils/date'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { DbLabStatusWrapper } from 'components/DbLabStatus/DbLabStatusWrapper'

import ConsolePageTitle from './../ConsolePageTitle'
import format from '../../utils/format'

interface DbLabSessionsProps {
  classes: ClassesType
  org: string | number
  orgId: number
  history: RouteComponentProps['history']
}

interface DbLabSessionsState {
  data: {
    auth: {
      token: string
    } | null
    dbLabSessions: {
      isProcessing: boolean
      isProcessed: boolean
      isComplete: boolean
      error: boolean
      data: {
        id: number
        duration: number
        started_at: string
        tags: {
          instance_id: string
          branch: string
          revision: string
          launched_by: string
          project_id: number
        }
        result: {
          status: string
          summary: {
            checklist: { [list: string]: { string: string | boolean } }
            elapsed: string
          }
        }
      }[]
    } | null
  } | null
}

const PAGE_SIZE = 20

class DbLabSessions extends Component<DbLabSessionsProps, DbLabSessionsState> {
  unsubscribe: Function
  componentDidMount() {
    const that = this
    const { orgId } = this.props

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const sessions =
        this.data && this.data.dbLabSessions ? this.data.dbLabSessions : null

      if (
        auth &&
        auth.token &&
        !sessions?.isProcessing &&
        !sessions?.error &&
        !that.state
      ) {
        Actions.getDbLabSessions(auth.token, { orgId, limit: PAGE_SIZE })
      }

      that.setState({ data: this.data })
    })

    const contentContainer = document.getElementById('content-container')
    if (contentContainer) {
      contentContainer.addEventListener('scroll', () => {
        if (
          contentContainer.scrollTop >=
          contentContainer.scrollHeight - contentContainer.offsetHeight
        ) {
          this.showMore()
        }
      })
    }

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  onSessionClick(
    _: MouseEvent<HTMLTableRowElement, globalThis.MouseEvent>,
    sessionId: string | number,
  ) {
    const { org } = this.props

    this.props.history.push('/' + org + '/observed-sessions/' + sessionId)
  }

  formatStatus(status: string) {
    const { classes } = this.props
    let icon = null
    let className = null
    let label = status
    if (status.length) {
      label = status.charAt(0).toUpperCase() + status.slice(1)
    }

    switch (status) {
      case 'passed':
        icon = icons.okIcon
        className = classes.passedStatus
        break
      case 'failed':
        icon = icons.failedIcon
        className = classes.failedStatus
        break
      default:
        icon = icons.processingIcon
        className = classes.processingStatus
    }

    return (
      <div className={className}>
        <span style={{ whiteSpace: 'nowrap' }}>
          {icon}&nbsp;{label}
        </span>
      </div>
    )
  }

  showMore() {
    const { orgId } = this.props
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const sessions =
      this.state.data && this.state.data.dbLabSessions
        ? this.state.data.dbLabSessions
        : null
    let lastId = null

    if (sessions && sessions.data && sessions.data.length) {
      lastId = sessions.data[sessions.data.length - 1].id
    }

    if (auth && auth.token && !sessions?.isProcessing && lastId) {
      Actions.getDbLabSessions(auth.token, {
        orgId,
        limit: PAGE_SIZE,
        lastId,
      })
    }
  }

  render() {
    const { classes, org } = this.props

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={org}
        breadcrumbs={[{ name: 'Database Lab observed sessions', url: null }]}
      />
    )

    const pageTitle = (
      <ConsolePageTitle
        title="Database Lab observed sessions"
        label={'Experimental'}
      />
    )

    if (!this.state || !this.state.data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}
          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    const sessionsStore =
      (this.state.data && this.state.data.dbLabSessions) || null
    const sessions = (sessionsStore && sessionsStore.data) || []

    if (sessionsStore && sessionsStore.error) {
      return (
        <div>
          {breadcrumbs}

          {pageTitle}

          <ErrorWrapper />
        </div>
      )
    }

    if (!sessionsStore || !sessionsStore.data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}
        {pageTitle}

        {sessions && sessions.length > 0 ? (
          <div>
            <HorizontalScrollContainer>
              <Table className={classes.table}>
                <TableHead>
                  <TableCell className={classes.tableHead}>Status</TableCell>
                  <TableCell className={classes.tableHead}>Session</TableCell>
                  <TableCell className={classes.tableHead}>
                    Project/Instance
                  </TableCell>
                  <TableCell className={classes.tableHead}>Commit</TableCell>
                  <TableCell className={classes.tableHead}>Checklist</TableCell>
                  <TableCell className={classes.tableHead}>&nbsp;</TableCell>
                </TableHead>

                <TableBody>
                  {sessions.map((s) => {
                    if (s) {
                      return (
                        <TableRow
                          hover={false}
                          className={classes.row}
                          key={s.id}
                          onClick={(event) => {
                            this.onSessionClick(event, s.id)
                            return false
                          }}
                          style={{ cursor: 'pointer' }}
                        >
                          <TableCell className={classes.tableCell}>
                            <DbLabStatusWrapper
                              session={{
                                status:
                                  s.result && s.result.status
                                    ? s.result.status
                                    : 'processing',
                              }}
                            />
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            #{s.id}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            {s.tags && s.tags.project_id
                              ? s.tags.project_id
                              : '-'}
                            /
                            {s.tags && s.tags.instance_id
                              ? s.tags.instance_id
                              : '-'}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            {icons.branch}&nbsp;
                            {s.tags && s.tags.branch && s.tags.revision
                              ? s.tags.branch + '/' + s.tags.revision
                              : '-'}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            {s.result &&
                            s.result.summary &&
                            s.result.summary.checklist ? (
                              <div>
                                {Object.keys(s.result.summary.checklist).map(
                                  function (key) {
                                    return (
                                      <span>
                                        {s.result?.summary?.checklist &&
                                        s.result.summary.checklist[key]
                                          ? icons.okLargeIcon
                                          : icons.failedLargeIcon}
                                        &nbsp;
                                      </span>
                                    )
                                  },
                                )}
                              </div>
                            ) : (
                              icons.processingLargeIcon
                            )}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            <div>
                              {s.duration > 0 ||
                              (s.result &&
                                s.result.summary &&
                                s.result.summary.elapsed) ? (
                                <span>
                                  {icons.timer}&nbsp;
                                  {s.result &&
                                  s.result.summary &&
                                  s.result.summary.elapsed
                                    ? s.result.summary.elapsed
                                    : format.formatSeconds(s.duration, 0, '')}
                                </span>
                              ) : (
                                '-'
                              )}
                            </div>
                            <div>
                              {icons.calendar}&nbsp;created&nbsp;
                              {isValidDate(new Date(s.started_at))
                                ? formatDistanceToNowStrict(
                                    new Date(s.started_at),
                                    {
                                      addSuffix: true,
                                    },
                                  )
                                : '-'}
                              {s.tags && s.tags.launched_by ? (
                                <span> by {s.tags.launched_by}</span>
                              ) : (
                                ''
                              )}
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    }

                    return null
                  })}
                </TableBody>
              </Table>
            </HorizontalScrollContainer>
            <div className={classes.showMoreContainer}>
              {sessionsStore && sessionsStore.isProcessing && (
                <Spinner className={classes.progress} />
              )}
              {sessionsStore &&
                !sessionsStore.isProcessing &&
                !sessionsStore.isComplete && (
                  <Button
                    variant="outlined"
                    color="secondary"
                    className={classes.button}
                    onClick={() => this.showMore()}
                    disabled={sessionsStore && sessionsStore.isProcessing}
                  >
                    Show more
                  </Button>
                )}
            </div>
          </div>
        ) : (
          <>
            {sessions && sessions.length === 0 && sessionsStore.isProcessed && (
              <StubContainer>
                <ProductCardWrapper
                  inline
                  title={'There are no Database Lab observed sessions yet.'}
                  icon={icons.databaseLabLogo}
                />
              </StubContainer>
            )}
          </>
        )}
      </div>
    )
  }
}

export default DbLabSessions
