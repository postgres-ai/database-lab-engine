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
  Menu,
  MenuItem,
  Tooltip,
  IconButton,
} from '@material-ui/core'
import WarningIcon from '@material-ui/icons/Warning'
import MoreVertIcon from '@material-ui/icons/MoreVert'
import { styles } from '@postgres.ai/shared/styles/styles'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { icons } from '@postgres.ai/shared/styles/icons'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from './../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { messages } from '../../assets/messages'
import Store from '../../stores/store'
import Urls from '../../utils/urls'
import { isHttps } from 'utils/utils'
import { getProjectAliasById, ProjectDataType } from 'utils/aliases'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { JoeInstancesProps } from 'components/JoeInstances/JoeInstancesWrapper'

interface JoeInstancesWithStylesProps extends JoeInstancesProps {
  classes: ClassesType
}

interface JoeInstancesState {
  anchorEl: (EventTarget & HTMLButtonElement) | null
  data: {
    auth: {
      token: string
    } | null
    joeInstances: {
      isProcessing: boolean
      projectId: string | number | undefined
      error: boolean
      orgId: number
      data: {
        [instance: string]: {
          id: number
          project_name: string
          project_label: string
          project_label_or_name: string
          url: string
          use_tunnel: boolean
        }
      }
    } | null
    userProfile: {
      data: {
        orgs: ProjectDataType
      }
    } | null
    projects: {
      isProcessing: boolean
      isProcessed: boolean
      error: boolean
      data: {
        id: number
        name: string
        label: string
        project_label_or_name: string
      }[]
    } | null
  } | null
}

class JoeInstances extends Component<
  JoeInstancesWithStylesProps,
  JoeInstancesState
> {
  unsubscribe: () => void
  componentDidMount() {
    const that = this
    let orgId = this.props.orgId ? this.props.orgId : null
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
      Actions.setJoeInstancesProject(orgId, projectId)
    } else {
      Actions.setJoeInstancesProject(orgId, 0)
    }

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null
      const joeInstances =
        this.data && this.data.joeInstances ? this.data.joeInstances : null
      const projects =
        this.data && this.data.projects ? this.data.projects : null

      if (
        auth &&
        auth.token &&
        !joeInstances?.isProcessing &&
        !joeInstances?.error &&
        !that.state
      ) {
        Actions.getJoeInstances(auth.token, orgId, projectId)
      }

      if (
        auth &&
        auth.token &&
        !projects?.isProcessing &&
        !projects?.error &&
        !that.state
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
  ) => {
    const url = Urls.linkJoeInstance(this.props, id)

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
    const project = getProjectAliasById(
      this.state.data?.userProfile?.data.orgs as ProjectDataType,
      projectId,
    )
    const props = { org, orgId, projectId, project }

    Actions.setJoeInstancesProject(orgId, event.target.value)
    this.props.history.push(Urls.linkJoeInstances(props))
  }

  registerButtonHandler = () => {
    this.props.history.push(Urls.linkJoeInstanceAdd(this.props))
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

      if (anchorEl) {
        let instanceId = anchorEl.getAttribute('instanceid')
        if (!instanceId) {
          return
        }

        switch (action) {
          case 'destroy':
            /* eslint no-alert: 0 */
            if (
              window.confirm(
                'Are you sure you want to remove this Joe Bot instance?',
              ) === true
            ) {
              Actions.destroyJoeInstance(auth?.token, instanceId)
            }

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
      this.state && this.state.data && this.state.data.joeInstances
        ? this.state.data.joeInstances
        : null
    const projects =
      this.state && this.state.data && this.state.data.projects
        ? this.state.data.projects
        : null
    let projectId = this.props.projectId ? this.props.projectId : null
    const menuOpen = Boolean(this.state && this.state.anchorEl)
    const title = 'Joe Bot instances'

    if (!projectId) {
      projectId =
        this.props.match &&
        this.props.match.params &&
        this.props.match.params.projectId
          ? this.props.match.params.projectId
          : null
    }

    const createPermitted = !orgPermissions || orgPermissions.joeInstanceCreate
    const deletePermitted = !orgPermissions || orgPermissions.joeInstanceDelete

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
                  {p.project_label_or_name || p.label || p.name}
                </MenuItem>
              )
            })}
          </TextField>
        </div>
      )
    }

    let breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: 'SQL Optimization', url: null }, { name: title }]}
      />
    )

    if (this.state && this.state.data?.joeInstances?.error) {
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

    const addInstanceButton = (
      <ConsoleButtonWrapper
        disabled={!createPermitted}
        variant="contained"
        color="primary"
        key="add_instance"
        onClick={this.registerButtonHandler}
        title={
          createPermitted ? 'Add a new Joe Bot instance' : messages.noPermission
        }
      >
        Add instance
      </ConsoleButtonWrapper>
    )

    const emptyListTitle = projectId
      ? 'There are no Joe Bot instances in this project yet'
      : 'There are no Joe Bot instances yet'

    let table = (
      <StubContainer>
        <ProductCardWrapper
          inline
          title={emptyListTitle}
          actions={[
            {
              id: 'addInstanceButton',
              content: addInstanceButton,
            },
          ]}
          icon={icons.joeLogo}
        >
          <p>
            Joe Bot is a virtual DBA for SQL Optimization. Joe helps engineers
            quickly troubleshoot and optimize SQL. Joe runs on top of the
            Database Lab Engine. (
            <GatewayLink
              href="https://postgres.ai/docs/joe-bot"
              target="_blank"
            >
              Learn more
            </GatewayLink>
            ).
          </p>
        </ProductCardWrapper>
      </StubContainer>
    )

    if (data.data && Object.keys(data.data).length > 0) {
      table = (
        <HorizontalScrollContainer>
          <Table className={classes.table} id="joeInstancesTable">
            <TableHead>
              <TableRow className={classes.row}>
                <TableCell>Project</TableCell>
                <TableCell>URL</TableCell>
                <TableCell>&nbsp;</TableCell>
              </TableRow>
            </TableHead>

            <TableBody>
              {Object.keys(data.data).map((i) => {
                return (
                  <TableRow
                    hover
                    className={classes.row}
                    key={data.data[i].id}
                    onClick={(event) =>
                      this.handleClick(event, data.data[i].id)
                    }
                    style={{ cursor: 'pointer' }}
                  >
                    <TableCell className={classes.cell}>
                      {data.data[i].project_label_or_name ||
                        data.data[i].project_label ||
                        data.data[i].project_name}
                    </TableCell>

                    <TableCell className={classes.cell}>
                      {data.data[i].url ? data.data[i].url : ''}
                      {!isHttps(data.data[i].url) &&
                      !data.data[i].use_tunnel ? (
                        <Tooltip
                          title="The connection to Joe Bot API is not secure"
                          aria-label="add"
                          classes={{ tooltip: classes.toolTip }}
                        >
                          <WarningIcon className={classes.warningIcon} />
                        </Tooltip>
                      ) : null}
                    </TableCell>

                    <TableCell align={'right'}>
                      <IconButton
                        aria-label="more"
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
    }

    const menu = (
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
          disabled={!deletePermitted}
          key={1}
          onClick={(event) => this.menuHandler(event, 'destroy')}
        >
          Remove
        </MenuItem>
      </Menu>
    )

    return (
      <div className={classes.root}>
        {breadcrumbs}

        <ConsolePageTitle title={title} actions={[addInstanceButton]} />

        {projectFilter}

        {table}

        {menu}
      </div>
    )
  }
}

export default JoeInstances
