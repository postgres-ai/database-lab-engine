/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component, MouseEvent } from 'react'
import { NavLink } from 'react-router-dom'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  MenuItem,
  Button,
  Checkbox,
} from '@material-ui/core'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import {
  ClassesType,
  ProjectProps,
} from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from './../ConsolePageTitle'
import { messages } from '../../assets/messages'
import { getProjectAliasById, ProjectDataType } from 'utils/aliases'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { ReportsProps } from 'components/Reports/ReportsWrapper'

interface ReportsWithStylesProps extends ReportsProps {
  classes: ClassesType
}

interface ReportsType {
  error: boolean
  errorMessage: string
  errorCode: number | null
  orgId: number | null
  projectId: string | number | undefined
  isDeleting: boolean
  isProcessing: boolean
  data: {
    id: number
    project_id: string
    created_formatted: string
    project_name: string
    project_label_or_name: string
    project_label: string
    epoch: string
  }[]
}

interface ReportsState {
  data: {
    auth: {
      token: string
    } | null
    userProfile: {
      data: {
        orgs: ProjectDataType
      }
    } | null
    reports: ReportsType | null
    projects: Omit<ProjectProps, 'isProcessed'>
  } | null
  selectedRows: {
    [rows: number]: number | boolean
  }
}

class Reports extends Component<ReportsWithStylesProps, ReportsState> {
  state = {
    data: {
      auth: {
        token: '',
      },
      userProfile: {
        data: {
          orgs: {},
        },
      },
      reports: {
        error: false,
        errorMessage: '',
        errorCode: null,
        orgId: null,
        projectId: undefined,
        isDeleting: false,
        isProcessing: false,
        data: [],
      },
      projects: {
        error: false,
        isProcessing: false,
        isProcessed: false,
        data: [],
      },
    },
    selectedRows: {},
  }

  onSelectRow(event: React.ChangeEvent<HTMLInputElement>, rowId: number) {
    let selectedRows: ReportsState['selectedRows'] = this.state.selectedRows

    if (selectedRows[rowId] && !event.target.checked) {
      delete selectedRows[rowId]
    } else {
      selectedRows[rowId] = event.target.checked
    }

    this.setState({ selectedRows: selectedRows })
  }

  onCheckBoxClick(
    event: React.ChangeEvent<HTMLInputElement> | MouseEvent<HTMLButtonElement>,
  ) {
    event.stopPropagation()
  }

  onSelectAllClick(
    event: React.ChangeEvent<HTMLInputElement>,
    reports: ReportsType['data'],
  ) {
    if (!event.target.checked) {
      this.setState({ selectedRows: {} })
      return
    }

    let selectedRows: ReportsState['selectedRows'] = {}
    if (selectedRows)
      for (let i in reports) {
        if (reports.hasOwnProperty(i)) {
          selectedRows[reports[i].id] = true
        }
      }

    this.setState({ selectedRows: selectedRows })
  }

  deleteReports() {
    const count = Object.keys(this.state.selectedRows).length
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null

    /* eslint no-alert: 0 */
    if (
      window.confirm(
        'Are you sure you want to delete ' + count + ' report(s)?',
      ) === true
    ) {
      let reports = []
      for (let i in this.state.selectedRows) {
        if (this.state.selectedRows.hasOwnProperty(i)) {
          reports.push(parseInt(i, 10))
        }
      }

      Actions.deleteCheckupReports(auth?.token, reports)
      this.setState({ selectedRows: {} })
    }
  }

  unsubscribe: () => void
  componentDidMount() {
    const that = this
    const orgId = this.props.orgId ? this.props.orgId : null
    let projectId = this.props.projectId ? this.props.projectId : null

    if (projectId) {
      Actions.setReportsProject(orgId, projectId)
    } else {
      Actions.setReportsProject(orgId, 0)
    }

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const reports: ReportsType =
        this.data && this.data.reports ? this.data.reports : null
      const projects: Omit<ProjectProps, 'isProcessed'> =
        this.data && this.data.projects ? this.data.projects : null

      if (
        auth &&
        auth.token &&
        !reports.isProcessing &&
        !reports.error &&
        !that.state
      ) {
        Actions.getCheckupReports(auth.token, orgId, projectId)
      }

      if (
        auth &&
        auth.token &&
        !projects.isProcessing &&
        !projects.error &&
        !that?.state &&
        !that?.state?.data
      ) {
        Actions.getProjects(auth.token, orgId)
      }

      that.setState({ data: this.data })
    })
    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  handleClick = (
    _: MouseEvent<HTMLTableRowElement, globalThis.MouseEvent>,
    id: number,
    projectId: string | number,
  ) => {
    const url = this.getReportLink(id, projectId)

    if (url) {
      this.props.history.push(url)
    }
  }

  getReportLink(id: number, projectId: string | number) {
    const org = this.props.org ? this.props.org : null
    const project = this.props.project ? this.props.project : null
    let projectAlias = project

    if (
      !projectAlias &&
      org &&
      this.state.data &&
      this.state.data.userProfile?.data.orgs
    ) {
      projectAlias = getProjectAliasById(
        this.state.data.userProfile.data.orgs,
        projectId,
      )
    }

    if (org && projectAlias) {
      return '/' + org + '/' + projectAlias + '/reports/' + id
    }

    return null
  }

  handleChangeProject = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const org = this.props.org ? this.props.org : null
    const orgId = this.props.orgId ? this.props.orgId : null
    const projectId = event.target.value
    let projectAlias = null

    if (org && this.state.data && this.state.data.userProfile?.data.orgs) {
      projectAlias = getProjectAliasById(
        this.state.data.userProfile.data.orgs,
        projectId,
      )
    }

    Actions.setReportsProject(orgId, event.target.value)
    if (org && this.props.history) {
      if (Number(event.target.value) !== 0 && projectId && projectAlias) {
        this.props.history.push('/' + org + '/' + projectAlias + '/reports/')
      } else {
        this.props.history.push('/' + org + '/reports/')
      }
    }
  }

  render() {
    const org = this.props.org ? this.props.org : null
    const { classes, orgId } = this.props
    const data =
      this.state && this.state.data && this.state.data.reports
        ? this.state.data.reports
        : null
    const projects =
      this.state && this.state.data && this.state.data.projects
        ? this.state.data.projects
        : null
    let projectId = this.props.projectId ? this.props.projectId : null
    let projectFilter = null

    const addAgentButton = (
      <ConsoleButtonWrapper
        disabled={!this.props.orgPermissions?.checkupReportConfigure}
        variant="contained"
        color="primary"
        onClick={() => this.props.history.push('/' + org + '/checkup-config')}
        title={
          this.props.orgPermissions?.checkupReportConfigure
            ? 'Add Checkup agent to your server'
            : messages.noPermission
        }
      >
        Add agent
      </ConsoleButtonWrapper>
    )

    const pageTitle = (
      <ConsolePageTitle title="Checkup reports" actions={[addAgentButton]} />
    )

    if (projects && projects.data && data) {
      projectFilter = (
        <div>
          <TextField
            value={data.projectId}
            onChange={(event) => this.handleChangeProject(event)}
            select
            label="Project"
            inputProps={{
              name: 'project',
              id: 'project-filter',
            }}
            InputLabelProps={{
              shrink: true,
              style: styles.inputFieldLabel,
            }}
            FormHelperTextProps={{
              style: styles.inputFieldHelper,
            }}
            variant="outlined"
            className={classes.filterSelect}
          >
            <MenuItem value={0}>All</MenuItem>

            {projects.data.map(
              (p: {
                id: number
                name: string
                label: string
                project_label_or_name: string
              }) => {
                return (
                  <MenuItem value={p.id} key={p.id}>
                    {p.project_label_or_name || p.label || p.name}
                  </MenuItem>
                )
              },
            )}
          </TextField>
        </div>
      )
    }

    let breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[{ name: 'Reports' }]}
      />
    )

    if (this.state && this.state.data && this.state.data.reports?.error) {
      return (
        <div>
          {breadcrumbs}

          {pageTitle}

          <ErrorWrapper />
        </div>
      )
    }

    if (
      !data ||
      (data && data.isProcessing) ||
      data.orgId !== orgId ||
      data.projectId !== (projectId ? projectId : 0)
    ) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <PageSpinner />
        </div>
      )
    }

    const emptyListTitle = projectId
      ? 'There are no uploaded checkup reports in this project yet'
      : 'There are no uploaded checkup reports'

    let reports = (
      <StubContainer className={classes.stubContainer}>
        <ProductCardWrapper
          inline
          title={emptyListTitle}
          actions={[
            {
              id: 'addAgentButton',
              content: addAgentButton,
            },
          ]}
          icon={icons.checkupLogo}
        >
          <p>
            Automated routine checkup for your PostgreSQL databases. Configure
            Checkup agent to start collecting reports (
            <GatewayLink
              href="https://postgres.ai/docs/checkup"
              target="_blank"
            >
              Learn more
            </GatewayLink>
            ).
          </p>
        </ProductCardWrapper>
      </StubContainer>
    )

    if (data && data.data && data.data.length > 0) {
      reports = (
        <div>
          {this.props.orgPermissions?.checkupReportDelete ? (
            <div className={classes.tableHead}>
              {data.isDeleting ? (
                <span>Processing...</span>
              ) : (
                <div>
                  {Object.keys(this.state.selectedRows).length > 0 ? (
                    <span>
                      Selected: {Object.keys(this.state.selectedRows).length}{' '}
                      rows
                    </span>
                  ) : (
                    'Select table rows to process them'
                  )}
                </div>
              )}
              <div className={classes.tableHeadActions}>
                <Button
                  variant="contained"
                  color="primary"
                  disabled={
                    Object.keys(this.state.selectedRows).length === 0 ||
                    data.isDeleting
                  }
                  onClick={() => this.deleteReports()}
                >
                  Delete
                  {data && data.isDeleting ? (
                    <span>
                      &nbsp;
                      <Spinner size="sm" />
                    </span>
                  ) : (
                    ''
                  )}
                </Button>
              </div>
            </div>
          ) : null}
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  {this.props.orgPermissions?.checkupReportDelete ? (
                    <TableCell className={classes.checkboxTableCell}>
                      <Checkbox
                        indeterminate={
                          Object.keys(this.state.selectedRows).length > 0
                        }
                        checked={
                          Object.keys(this.state.selectedRows).length ===
                          data.data.length
                        }
                        onChange={(event) =>
                          this.onSelectAllClick(event, data.data)
                        }
                        onClick={(event) => this.onCheckBoxClick(event)}
                      />
                    </TableCell>
                  ) : null}
                  <TableCell>Report #</TableCell>
                  <TableCell>Project</TableCell>
                  <TableCell>Created</TableCell>
                  <TableCell>Epoch</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data.data.map(
                  (r: {
                    id: number
                    project_id: string
                    created_formatted: string
                    project_name: string
                    epoch: string
                    project_label_or_name: string
                    project_label: string
                  }) => {
                    return (
                      <TableRow
                        hover
                        className={classes.row}
                        key={r.id}
                        onClick={(event) =>
                          this.handleClick(event, r.id, r.project_id)
                        }
                        style={{ cursor: 'pointer' }}
                      >
                        {this.props.orgPermissions?.checkupReportDelete ? (
                          <TableCell className={classes.checkboxTableCell}>
                            <Checkbox
                              checked={
                                !!(
                                  this.state
                                    .selectedRows as ReportsState['selectedRows']
                                )[r.id]
                              }
                              onChange={(event) =>
                                this.onSelectRow(event, r.id)
                              }
                              onClick={(event) => this.onCheckBoxClick(event)}
                            />
                          </TableCell>
                        ) : null}
                        <TableCell className={classes.cell}>
                          <NavLink
                            to={
                              this.getReportLink(r.id, r.project_id) as string
                            }
                          >
                            {r.id}
                          </NavLink>
                        </TableCell>
                        <TableCell className={classes.cell}>
                          {r.project_label_or_name ||
                            r.project_label ||
                            r.project_name}
                        </TableCell>
                        <TableCell className={classes.cell}>
                          {r.created_formatted}
                        </TableCell>
                        <TableCell className={classes.cell}>
                          {r.epoch}
                        </TableCell>
                      </TableRow>
                    )
                  },
                )}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        </div>
      )
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {projectFilter}

        {reports}
      </div>
    )
  }
}

export default Reports
