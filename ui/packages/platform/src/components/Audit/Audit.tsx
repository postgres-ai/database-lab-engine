/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Button,
  ExpansionPanel,
  ExpansionPanelSummary,
  ExpansionPanelDetails,
  TextField,
} from '@material-ui/core'
import ExpandMoreIcon from '@material-ui/icons/ExpandMore'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import ConsolePageTitle from '../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import Store from '../../stores/store'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { messages } from '../../assets/messages'
import format from '../../utils/format'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { AuditProps } from 'components/Audit/AuditWrapper'
import { FilteredTableMessage } from 'components/AccessTokens/FilteredTableMessage/FilteredTableMessage'

const PAGE_SIZE = 20
const auditTitle = 'Audit log'

interface AuditWithStylesProps extends AuditProps {
  classes: ClassesType
}

export interface AuditLogData {
  id: number
  action: string
  actor: string
  action_data: {
    processed_row_count: number
    data_before: Record<string, string>[]
    data_after: Record<string, string>[]
  }
  created_at: string
  table_name: string
}

interface AuditState {
  filterValue: string
  data: {
    auth: {
      token: string
    } | null
    auditLog: {
      isProcessing: boolean
      orgId: number
      error: boolean
      isComplete: boolean
      errorCode: number
      errorMessage: string
      data: AuditLogData[]
    } | null
  }
}

class Audit extends Component<AuditWithStylesProps, AuditState> {
  unsubscribe: Function
  componentDidMount() {
    const that = this
    const orgId = this.props.orgId

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      const auth: AuditState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const auditLog: AuditState['data']['auditLog'] =
        this.data && this.data.auditLog ? this.data.auditLog : null

      that.setState({ data: this.data })

      if (
        auth &&
        auth.token &&
        auditLog &&
        !auditLog.isProcessing &&
        !auditLog.error &&
        !that.state
      ) {
        Actions.getAuditLog(auth.token, {
          orgId,
          limit: PAGE_SIZE,
        })
      }
    })

    const contentContainer = document.getElementById('content-container')
    if (contentContainer) {
      contentContainer.addEventListener('scroll', () => {
        if (
          contentContainer !== null &&
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

  showMore() {
    const { orgId } = this.props
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const logs = this.state && this.state.data ? this.state.data.auditLog : null
    let lastId = null

    if (logs && logs?.data && logs.data?.length) {
      lastId = logs.data[logs.data.length - 1].id
    }

    if (auth && auth.token && !logs?.isProcessing && lastId) {
      Actions.getAuditLog(auth.token, {
        orgId,
        limit: PAGE_SIZE,
        lastId,
      })
    }
  }

  formatAction = (r: AuditLogData) => {
    const { classes } = this.props
    let acted = r.action
    let actor = r.actor
    let actorSrc = ''
    let rows = 'row'

    if (!actor) {
      actor = 'Unknown'
      actorSrc = ' (changed directly in database) '
    }

    if (r.action_data && r.action_data.processed_row_count) {
      rows =
        r.action_data.processed_row_count +
        ' ' +
        (r.action_data.processed_row_count > 1 ? 'rows' : 'row')
    }

    switch (r.action) {
      case 'insert':
        acted = ' added ' + rows + ' to'
        break
      case 'delete':
        acted = ' deleted ' + rows + ' from'
        break
      default:
        acted = ' updated ' + rows + ' in'
    }

    return (
      <div className={classes?.actionDescription}>
        <strong>{actor}</strong>
        {actorSrc} {acted} table&nbsp;<strong>{r.table_name}</strong>
      </div>
    )
  }

  getDataSectionTitle = (r: AuditLogData, before: boolean) => {
    switch (r.action) {
      case 'insert':
        return ''
      case 'delete':
        return ''
      default:
        return before ? 'Before:' : 'After:'
    }
  }

  getChangesTitle = (r: AuditLogData) => {
    const displayedCount = r.action_data && r.action_data.data_before
      ? r.action_data.data_before?.length
      : r.action_data?.data_after?.length
    const objCount =
      r.action_data && r.action_data.processed_row_count
        ? r.action_data.processed_row_count
        : null

    if (displayedCount && (objCount as number) > displayedCount) {
      return (
        'Changes (displayed ' +
        displayedCount +
        ' rows out of ' +
        objCount +
        ')'
      )
    }

    return 'Changes'
  }

  filterInputHandler = (event: React.ChangeEvent<HTMLInputElement>) => {
    this.setState({ filterValue: event.target.value })
  }

  render() {
    const { classes, orgPermissions, orgId } = this.props
    const data = this.state && this.state.data ? this.state.data.auditLog : null
    const logsStore =
      (this.state && this.state.data && this.state.data.auditLog) || null
    const logs = (logsStore && logsStore.data) || []

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: auditTitle }]}
      />
    )

    const filteredLogs = logs.filter(
      (log) =>
        log.actor
          ?.toLowerCase()
          .indexOf((this.state.filterValue || '')?.toLowerCase()) !== -1,
    )

    const pageTitle = (
      <ConsolePageTitle
        title={auditTitle}
      />
    )

    if (orgPermissions && !orgPermissions.auditLogView) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}
          {pageTitle}
          <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
        </div>
      )
    }

    if (
      !logsStore ||
      !logsStore.data ||
      (logsStore && logsStore.orgId !== orgId)
    ) {
      return (
        <div className={classes?.root}>
          {breadcrumbs}
          {pageTitle}
          <PageSpinner />
        </div>
      )
    }

    if (logsStore.error) {
      return (
        <div className={classes?.root}>
          <ErrorWrapper
            message={logsStore.errorMessage}
            code={logsStore.errorCode}
          />
        </div>
      )
    }

    return (
      <div className={classes?.root}>
        {breadcrumbs}
        {pageTitle}
        {filteredLogs && filteredLogs.length > 0 ? (
          <div>
            <HorizontalScrollContainer>
              <Table className={classes?.table}>
                <TableHead>
                  <TableRow className={classes?.row}>
                    <TableCell>Action</TableCell>
                    <TableCell>Time</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {logs.map((r) => {
                    return (
                      <TableRow hover className={classes?.row} key={r.id}>
                        <TableCell className={classes?.cell}>
                          {this.formatAction(r)}
                          {((r.action_data && r.action_data.data_before) || (r.action_data && r.action_data.data_after)) && (
                            <div>
                              <ExpansionPanel
                                className={classes?.expansionPanel}
                              >
                                <ExpansionPanelSummary
                                  expandIcon={<ExpandMoreIcon />}
                                  aria-controls="panel1b-content"
                                  id="panel1b-header"
                                  className={classes?.expansionPanelSummary}
                                >
                                  {this.getChangesTitle(r)}
                                </ExpansionPanelSummary>
                                <ExpansionPanelDetails
                                  className={classes?.expansionPanelDetails}
                                >
                                  {r.action_data && r.action_data.data_before && (
                                    <div className={classes?.data}>
                                      {this.getDataSectionTitle(r, true)}
                                      <TextField
                                        variant="outlined"
                                        id={'before_data_' + r.id}
                                        multiline
                                        fullWidth
                                        value={JSON.stringify(
                                          r.action_data.data_before,
                                          null,
                                          4,
                                        )}
                                        className={classes?.code}
                                        margin="normal"
                                        InputProps={{
                                          readOnly: true,
                                        }}
                                      />
                                    </div>
                                  )}
                                  {r.action_data && r.action_data.data_after && (
                                    <div className={classes?.data}>
                                      {this.getDataSectionTitle(r, false)}
                                      <TextField
                                        variant="outlined"
                                        id={'after_data_' + r.id}
                                        multiline
                                        fullWidth
                                        value={JSON.stringify(
                                          r.action_data.data_after,
                                          null,
                                          4,
                                        )}
                                        className={classes?.code}
                                        margin="normal"
                                        InputProps={{
                                          readOnly: true,
                                        }}
                                      />
                                    </div>
                                  )}
                                </ExpansionPanelDetails>
                              </ExpansionPanel>
                            </div>
                          )}
                        </TableCell>
                        <TableCell className={classes?.timeCell}>
                          {format.formatTimestampUtc(r.created_at)}
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </HorizontalScrollContainer>
            <div className={classes?.showMoreContainer}>
              {data && data.isProcessing && (
                <Spinner className={classes?.progress} />
              )}
              {data && !data.isProcessing && !data.isComplete && (
                <Button
                  variant="outlined"
                  color="secondary"
                  className={classes?.button}
                  onClick={() => this.showMore()}
                  disabled={data && data.isProcessing}
                >
                  Show more
                </Button>
              )}
            </div>
          </div>
        ) : (
          <FilteredTableMessage
            filteredItems={filteredLogs}
            emptyState="Audit log records not found"
            filterValue={this.state.filterValue}
            clearFilter={() =>
              this.setState({
                filterValue: '',
              })
            }
          />
        )}
        <div className={classes?.bottomSpace} />
      </div>
    )
  }
}

export default Audit
