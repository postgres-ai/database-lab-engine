/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component, MouseEvent } from 'react'
import { formatDistanceToNowStrict } from 'date-fns'
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
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { Modal } from '@postgres.ai/shared/components/Modal'
import { styles } from '@postgres.ai/shared/styles/styles'
import {
  ClassesType,
  ProjectProps,
  RefluxTypes,
} from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import ConsolePageTitle from './../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { messages } from '../../assets/messages'
import Store from '../../stores/store'
import Urls from '../../utils/urls'
import format from '../../utils/format'
import { isHttps } from '../../utils/utils'
import { WarningWrapper } from 'components/Warning/WarningWrapper'
import { ProjectDataType, getProjectAliasById } from 'utils/aliases'
import { InstanceStateDto } from '@postgres.ai/shared/types/api/entities/instanceState'
import { InstanceDto } from '@postgres.ai/shared/types/api/entities/instance'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { DbLabStatusWrapper } from 'components/DbLabStatus/DbLabStatusWrapper'
import { DbLabInstancesProps } from 'components/DbLabInstances/DbLabInstancesWrapper'
import { CreatedDbLabCards } from 'components/CreateDbLabCards/CreateDbLabCards'
import { CreateClusterCards } from 'components/CreateClusterCards/CreateClusterCards'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'

interface PostgresClustersProps extends DbLabInstancesProps {
  classes: ClassesType
}

interface DbLabInstancesState {
  modalState: {
    open: boolean
    type: string
  }
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
          created_at: string
          project_label_or_name: string
          plan: string
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

class PostgresClusters extends Component<
  PostgresClustersProps,
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

    this.unsubscribe = (Store.listen as RefluxTypes['listen'])(function () {
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

  unsubscribe: Function
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

  registerButtonHandler = (provider: string) => {
    this.props.history.push(Urls.linkDbLabInstanceAdd(this.props, provider))
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
    const title = 'Postgres Clusters'
    const addPermitted = !orgPermissions || orgPermissions.dblabInstanceCreate
    const deletePermitted =
      !orgPermissions || orgPermissions.dblabInstanceDelete

    const getVersionDigits = (str: string) => {
      if (!str) {
        return 'N/A'
      }

      const digits = str.match(/\d+/g)

      if (digits && digits.length > 0) {
        return `${digits[0]}.${digits[1]}.${digits[2]}`
      }
      return ''
    }

    const addInstanceButton = (
      <div className={classes.buttonContainer}>
        <ConsoleButtonWrapper
          disabled={!addPermitted}
          variant="contained"
          color="primary"
          key="add-cluster"
          onClick={() =>
            this.setState({ modalState: { open: true, type: 'cluster' } })
          }
          title={addPermitted ? 'Create new cluster' : messages.noPermission}
        >
          New Postgres cluster
        </ConsoleButtonWrapper>
      </div>
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
            label="Name"
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
            disabled
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

    const CardsModal = () => (
      <Modal
        size={'md'}
        isOpen={this.state.modalState.open}
        title={'Choose the location for your Cluster installation'}
        onClose={() => this.setState({ modalState: { open: false, type: '' } })}
        aria-labelledby="simple-modal-title"
        aria-describedby="simple-modal-description"
      >
        <CreateClusterCards isModal props={this.props} dblabPermitted={addPermitted} />
      </Modal>
    )

    let table = (
      <CreateClusterCards props={this.props} dblabPermitted={addPermitted} />
    )

    let menu = null
    if (true) {
      table = (
        <HorizontalScrollContainer>
          <Table className={classes.table} id="dblabInstancesTable">
            <TableHead>
              <TableRow className={classes.row}>
                <TableCell>Cluster name</TableCell>
                <TableCell>ID</TableCell>
                <TableCell>Plan</TableCell>
                <TableCell>Version</TableCell>
                <TableCell>State</TableCell>
                <TableCell>Created at</TableCell>
                <TableCell>&nbsp;</TableCell>
              </TableRow>
            </TableHead>

            <TableBody>
              {/* {Object.keys(data.data).map((index) => {
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
                      {data.data[index].id}
                    </TableCell>
                    <TableCell className={classes.cell}>
                      {data.data[index].state && data.data[index].url
                        ? data.data[index].url
                        : 'N/A'}
                      {!isHttps(data.data[index].url) &&
                      data.data[index].url &&
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
                      {data.data[index]?.state?.cloning?.numClones ??
                        data.data[index]?.state?.clones?.length ??
                        'N/A'}
                    </TableCell>
                    <TableCell>
                      {data.data[index] &&
                        (data.data[index]?.plan === 'EE'
                          ? 'Enterprise'
                          : data.data[index]?.plan === 'SE'
                          ? 'Standard'
                          : data.data[index]?.plan)}
                    </TableCell>
                    <TableCell>
                      {getVersionDigits(
                        data.data[index] &&
                          (data.data[index].state?.engine?.version as string),
                      )}
                    </TableCell>
                    <TableCell className={classes.cell}>
                      {data.data[index].state && data.data[index].url ? (
                        <DbLabStatusWrapper
                          instance={data.data[index]}
                          onlyText={false}
                        />
                      ) : (
                        'N/A'
                      )}
                    </TableCell>
                    <TableCell className={classes.cell}>
                      <Tooltip
                        title={formatDistanceToNowStrict(
                          new Date(data.data[index].created_at),
                          {
                            addSuffix: true,
                          },
                        )}
                        classes={{ tooltip: classes.timeLabel }}
                      >
                        <span>
                          {format.formatTimestampUtc(
                            data.data[index].created_at,
                          ) ?? ''}
                        </span>
                      </Tooltip>
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
              })} */}
            </TableBody>
          </Table>
        </HorizontalScrollContainer>
      )

      const selectedInstance = Object.values(data.data).filter((item) => {
        const anchorElLabel = this.state.anchorEl?.getAttribute('aria-label')
        // eslint-disable-next-line eqeqeq
        return anchorElLabel && item.id == anchorElLabel
      })[0]

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
            disabled={!addPermitted || selectedInstance?.plan === 'SE'}
          >
            Edit
          </MenuItem>
          <MenuItem
            key={2}
            onClick={(event) => this.menuHandler(event, 'addclone')}
            disabled={selectedInstance?.plan === 'SE'}
          >
            Create clone
          </MenuItem>
          <MenuItem
            key={3}
            onClick={(event) => this.menuHandler(event, 'refresh')}
            disabled={selectedInstance?.plan === 'SE'}
          >
            Refresh
          </MenuItem>
          <MenuItem
            disabled={!deletePermitted}
            key={4}
            onClick={(event) => this.menuHandler(event, 'destroy')}
          >
            Remove from List
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

        {this.state.modalState && <CardsModal />}
      </div>
    )
  }
}

export default PostgresClusters
