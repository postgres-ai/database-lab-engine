/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component, MouseEvent } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  IconButton,
  Menu,
  MenuItem,
  Tooltip,
} from '@material-ui/core'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import WarningIcon from '@material-ui/icons/Warning'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'
import {
  ClassesType,
  ProjectProps,
} from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import ConsolePageTitle from './../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import { messages } from '../../assets/messages'
import Store from '../../stores/store'
import Urls from '../../utils/urls'
import { isHttps } from '../../utils/utils'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { ProjectDataType, getProjectAliasById } from 'utils/aliases'
import { InstanceStateDto } from '@postgres.ai/shared/types/api/entities/instanceState'
import { InstanceDto } from '@postgres.ai/shared/types/api/entities/instance'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { DbLabStatusWrapper } from 'components/DbLabStatus/DbLabStatusWrapper'
import { DbLabInstancesProps } from 'components/DbLabInstances/DbLabInstancesWrapper'

interface DbLabInstancesWithStylesProps extends DbLabInstancesProps {
  classes: ClassesType
}

interface DbLabInstancesState {
  data: {
    auth: {
      token: string
    } | null
    userProfile: {
      data: {
        orgs: ProjectDataType
      }
    }
    dbLabInstances: {
      orgId: number
      data: {
        [org: string]: {
          project_label_or_name: string
          project_name: string
          project_label: string
          url: string
          use_tunnel: boolean
          isProcessing: boolean
          id: string
          project_alias: string
          state: InstanceStateDto
          dto: InstanceDto
        }
      }
      isProcessing: boolean
      projectId: string | number | undefined
      error: boolean
    } | null
    dbLabInstanceStatus: {
      instanceId: string
      isProcessing: boolean
    }
    projects: Omit<ProjectProps, 'isProcessed'>
  }
  anchorEl: (EventTarget & HTMLButtonElement) | null
}

class DbLabInstances extends Component<
  DbLabInstancesWithStylesProps,
  DbLabInstancesState
> {
  componentDidMount() {
    const that = this
    const orgId = this.props.orgId ? this.props.orgId : null
    let projectId = this.props.projectId ? this.props.projectId : null

    if (!projectId) {
      projectId =
        this.props.match &&
        this.props.match.params &&
        this.props.match.params.projectId
          ? this.props.match.params.projectId
          : null
    }

    if (projectId) {
      Actions.setDbLabInstancesProject(orgId, projectId)
    } else {
      Actions.setDbLabInstancesProject(orgId, 0)
    }

    this.unsubscribe = Store.listen(function () {
      const auth: DbLabInstancesState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const dbLabInstances: DbLabInstancesState['data']['dbLabInstances'] =
        this.data && this.data.dbLabInstances ? this.data.dbLabInstances : null
      const projects: Omit<ProjectProps, 'isProcessed'> =
        this.data && this.data.projects ? this.data.projects : null

      if (
        auth &&
        auth.token &&
        !dbLabInstances?.isProcessing &&
        !dbLabInstances?.error &&
        !that.state
      ) {
        Actions.getDbLabInstances(auth.token, orgId, projectId)
      }

      if (
        auth &&
        auth.token &&
        !projects.isProcessing &&
        !projects.error &&
        !that.state
      ) {
        Actions.getProjects(auth.token, orgId)
      }

      that.setState({ data: this.data })
    })

    Actions.refresh()
  }

  unsubscribe: () => void
  componentWillUnmount() {
    this.unsubscribe()
  }

  handleClick = (
    _: MouseEvent<HTMLTableRowElement, globalThis.MouseEvent>,
    id: string,
  ) => {
    const url = Urls.linkDbLabInstance(this.props, id)

    if (url) {
      this.props.history.push(url)
    }
  }

  handleChangeProject = (
    event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>,
  ) => {
    const org = this.props.org ? this.props.org : null
    const orgId = this.props.orgId ? this.props.orgId : null
    const projectId = event.target.value
    const project = this.state.data
      ? getProjectAliasById(this.state.data?.userProfile?.data?.orgs, projectId)
      : ''
    const props = { org, orgId, projectId, project }

    Actions.setDbLabInstancesProject(orgId, event.target.value)
    this.props.history.push(Urls.linkDbLabInstances(props))
  }

  registerButtonHandler = () => {
    this.props.history.push(Urls.linkDbLabInstanceAdd(this.props))
  }

  openMenu = (event: MouseEvent<HTMLButtonElement>) => {
    event.stopPropagation()
    this.setState({ anchorEl: event.currentTarget })
  }

  closeMenu = () => {
    this.setState({ anchorEl: null })
  }

  menuHandler = (_: MouseEvent<HTMLLIElement>, action: string) => {
    const anchorEl = this.state.anchorEl

    this.closeMenu()

    setTimeout(() => {
      const auth =
        this.state.data && this.state.data.auth ? this.state.data.auth : null
      const data =
        this.state.data && this.state.data.dbLabInstances
          ? this.state.data.dbLabInstances
          : null

      if (anchorEl) {
        const instanceId = anchorEl.getAttribute('aria-label')
        if (!instanceId) {
          return
        }

        let project = ''
        if (data?.data) {
          for (const i in data.data) {
            if (parseInt(data.data[i].id, 10) === parseInt(instanceId, 10)) {
              project = data.data[i].project_alias
            }
          }
        }

        switch (action) {
          case 'addclone':
            this.props.history.push(
              Urls.linkDbLabCloneAdd(
                { org: this.props.org, project: project },
                instanceId,
              ),
            )

            break

          case 'destroy':
            /* eslint no-alert: 0 */
            if (
              window.confirm(
                'Are you sure you want to remove this Database Lab instance?',
              ) === true
            ) {
              Actions.destroyDbLabInstance(auth?.token, instanceId)
            }

            break

          case 'refresh':
            Actions.getDbLabInstanceStatus(auth?.token, instanceId)

            break

          case 'editProject':
            this.props.history.push(
              Urls.linkDbLabInstanceEditProject(
                { org: this.props.org, project: project },
                instanceId,
              ),
            )

            break

          default:
            break
        }
      }
    }, 100)
  }

  render() {
    const { classes, orgPermissions, orgId } = this.props
    const data =
      this.state && this.state.data && this.state.data.dbLabInstances
        ? this.state.data.dbLabInstances
        : null
    const projects =
      this.state && this.state.data && this.state.data?.projects
        ? this.state.data.projects
        : null
    const projectId = this.props.projectId ? this.props.projectId : null
    const menuOpen = Boolean(this.state && this.state.anchorEl)
    const title = 'Database Lab Instances'
    const addPermitted = !orgPermissions || orgPermissions.dblabInstanceCreate
    const deletePermitted =
      !orgPermissions || orgPermissions.dblabInstanceDelete

    const addInstanceButton = (
      <ConsoleButtonWrapper
        disabled={!addPermitted}
        variant="contained"
        color="primary"
        key="add_dblab_instance"
        onClick={this.registerButtonHandler}
        title={
          addPermitted
            ? 'Add a new Database Lab instance'
            : messages.noPermission
        }
      >
        Add instance
      </ConsoleButtonWrapper>
    )
    const pageTitle = (
      <ConsolePageTitle title={title} actions={[addInstanceButton]} />
    )

    let projectFilter = null
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

            {projects.data.map((p) => {
              return (
                <MenuItem value={p.id} key={p.id}>
                  {p?.project_label_or_name || p.name}
                </MenuItem>
              )
            })}
          </TextField>
        </div>
      )
    }

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: title }]}
      />
    )

    if (orgPermissions && !orgPermissions.dblabInstanceList) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <WarningWrapper>{messages.noPermissionPage}</WarningWrapper>
        </div>
      )
    }

    if (this.state?.data && this.state.data?.dbLabInstances?.error) {
      return (
        <div>
          {breadcrumbs}

          <ConsolePageTitle title={title} />

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

          <ConsolePageTitle title={title} />

          <PageSpinner />
        </div>
      )
    }

    const emptyListTitle = projectId
      ? 'There are no Database Lab instances in this project yet'
      : 'There are no Database Lab instances yet'

    let table = (
      <StubContainer className={classes.stubContainer}>
        <ProductCardWrapper
          inline
          title={emptyListTitle}
          actions={[
            {
              id: 'addInstanceButton',
              content: addInstanceButton,
            },
          ]}
          icon={icons.databaseLabLogo}
        >
          <p>
            Clone multi-terabyte databases in seconds and use them to test your
            database migrations, optimize SQL, or deploy full-size staging apps.
            Start here to work with all Database Lab tools. Setup
            <GatewayLink
              href="https://postgres.ai/docs/database-lab"
              target="_blank"
            >
              documentation here
            </GatewayLink>
            .
          </p>
        </ProductCardWrapper>
      </StubContainer>
    )

    let menu = null
    if (data.data && Object.keys(data.data).length > 0) {
      table = (
        <HorizontalScrollContainer>
          <Table className={classes.table} id="dblabInstancesTable">
            <TableHead>
              <TableRow className={classes.row}>
                <TableCell>Project</TableCell>
                <TableCell>URL</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Clones</TableCell>
                <TableCell>&nbsp;</TableCell>
              </TableRow>
            </TableHead>

            <TableBody>
              {Object.keys(data.data).map((index) => {
                return (
                  <TableRow
                    hover
                    className={classes.row}
                    key={index}
                    onClick={(event) =>
                      this.handleClick(event, data.data[index].id)
                    }
                    style={{ cursor: 'pointer' }}
                  >
                    <TableCell className={classes.cell}>
                      {data.data[index].project_label_or_name ||
                        data.data[index].project_name}
                    </TableCell>

                    <TableCell className={classes.cell}>
                      {data.data[index].state && data.data[index].url
                        ? data.data[index].url
                        : ''}
                      {!isHttps(data.data[index].url) &&
                      !data.data[index].use_tunnel ? (
                        <Tooltip
                          title="The connection to Database Lab API is not secure"
                          aria-label="add"
                          classes={{ tooltip: classes.tooltip }}
                        >
                          <WarningIcon className={classes.warningIcon} />
                        </Tooltip>
                      ) : null}
                    </TableCell>

                    <TableCell className={classes.cell}>
                      <DbLabStatusWrapper
                        instance={data.data[index]}
                        onlyText={false}
                      />
                    </TableCell>

                    <TableCell className={classes.cell}>
                      {data.data[index]?.state?.cloning?.numClones ??
                        data.data[index]?.state?.clones?.length ??
                        ''}
                    </TableCell>

                    <TableCell align={'right'}>
                      {data.data[index].isProcessing ||
                      (this.state.data?.dbLabInstanceStatus.instanceId ===
                        index &&
                        this.state.data.dbLabInstanceStatus.isProcessing) ? (
                        <Spinner className={classes.inTableProgress} />
                      ) : null}
                      <IconButton
                        aria-label={data.data[index].id}
                        aria-controls="instance-menu"
                        aria-haspopup="true"
                        onClick={this.openMenu}
                      >
                        <MoreVertIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </HorizontalScrollContainer>
      )

      menu = (
        <Menu
          id="instance-menu"
          anchorEl={this.state.anchorEl}
          keepMounted
          open={menuOpen}
          onClose={this.closeMenu}
          PaperProps={{
            style: {
              width: 200,
            },
          }}
        >
          <MenuItem
            key={1}
            onClick={(event) => this.menuHandler(event, 'editProject')}
            disabled={!addPermitted}
          >
            Edit
          </MenuItem>
          <MenuItem
            key={2}
            onClick={(event) => this.menuHandler(event, 'addclone')}
          >
            Create clone
          </MenuItem>
          <MenuItem
            key={3}
            onClick={(event) => this.menuHandler(event, 'refresh')}
          >
            Refresh
          </MenuItem>
          <MenuItem
            disabled={!deletePermitted}
            key={4}
            onClick={(event) => this.menuHandler(event, 'destroy')}
          >
            Remove
          </MenuItem>
        </Menu>
      )
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {projectFilter}

        {table}

        {menu}
      </div>
    )
  }
}

export default DbLabInstances
