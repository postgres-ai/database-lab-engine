/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component, MouseEvent } from 'react'
import { NavLink } from 'react-router-dom'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Button,
  Grid,
} from '@material-ui/core'
import ReactMarkdown from 'react-markdown'
import rehypeRaw from 'rehype-raw'
import remarkGfm from 'remark-gfm'

import { HorizontalScrollContainer } from '@postgres.ai/shared/components/HorizontalScrollContainer'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { StubContainer } from '@postgres.ai/shared/components/StubContainer'
import { icons } from '@postgres.ai/shared/styles/icons'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import { ROUTES } from 'config/routes'

import Actions from '../../actions/actions'
import ConsolePageTitle from '../ConsolePageTitle'
import { ErrorWrapper } from 'components/Error/ErrorWrapper'
import { Link } from '@postgres.ai/shared/components/Link2'
import { messages } from '../../assets/messages'
import Store from '../../stores/store'
import Urls from '../../utils/urls'
import settings from '../../utils/settings'

import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { ProductCardWrapper } from 'components/ProductCard/ProductCardWrapper'
import { DashboardProps } from 'components/Dashboard/DashboardWrapper'

interface DashboardWithStylesProps extends DashboardProps {
  classes: ClassesType
}

interface DashboardState {
  data: {
    auth: {
      token: string
    } | null
    projects: {
      error: boolean
      isProcessing: boolean
      orgId: number
      data: {
        id: number
        name: string
        alias: string
      }[]
    } | null
    orgId: number
    userProfile: {
      data: {
        platform_onboarding_text: string
        orgs: {
          [org: string]: {
            id: number
            alias: string
            name: string
            onboarding_text: string
            projects: Object
          }
        }
      }
      isProcessing: boolean
      isProcessed: boolean
      error: boolean
    } | null
    useDemoData: {
      isProcessing: boolean
      isProcessed: boolean
    } | null
    dashboard: {
      profileUpdateInitAfterDemo: boolean
    } | null
  }
}

class Dashboard extends Component<DashboardWithStylesProps, DashboardState> {
  unsubscribe: () => void
  componentDidMount() {
    const that = this
    const orgId = this.props.orgId
    const onlyProjects = this.props.onlyProjects

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data })
      const auth: DashboardState['data']['auth'] =
        this.data && this.data.auth ? this.data.auth : null
      const userProfile: DashboardState['data']['userProfile'] =
        this.data && this.data.userProfile ? this.data.userProfile : null

      if (onlyProjects) {
        const projects: DashboardState['data']['projects'] =
          this.data && this.data.projects ? this.data.projects : null

        if (
          auth &&
          auth.token &&
          !projects?.isProcessing &&
          !projects?.error &&
          !that.state
        ) {
          Actions.getProjects(auth.token, orgId)
        }

        if (
          auth &&
          !that.state &&
          !userProfile?.isProcessing &&
          !userProfile?.error
        ) {
          Actions.getUserProfile(auth.token)
        }
      } else {
        const useDemoData =
          this.data && this.data.useDemoData ? this.data.useDemoData : null
        const profileUpdateInitAfterDemo =
          this.data && this.data.dashboard
            ? this.data.dashboard.profileUpdateInitAfterDemo
            : null

        if (
          auth &&
          auth.token &&
          ((!userProfile?.isProcessed &&
            !userProfile?.isProcessing &&
            !userProfile?.error) ||
            (!profileUpdateInitAfterDemo &&
              useDemoData.isProcessed &&
              !useDemoData.error))
        ) {
          if (useDemoData.isProcessed) {
            this.data.dashboard.profileUpdateInitAfterDemo = true
          }

          Actions.getUserProfile(auth.token)
        }
      }
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  handleClick = (
    _: MouseEvent<HTMLTableRowElement, globalThis.MouseEvent>,
    alias?: string,
  ) => {
    this.props.history.push('/' + alias)
  }

  useDemoDataButtonHandler = () => {
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    Actions.useDemoData(auth?.token)
  }

  addOrgButtonHandler = () => {
    this.props.history.push(ROUTES.CREATE_ORG.path)
  }

  addDblabInstanceButtonHandler = () => {
    this.props.history.push(Urls.linkDbLabInstanceAdd(this.props))
  }

  addCheckupAgentButtonHandler = () => {
    this.props.history.push(Urls.linkCheckupAgentAdd(this.props))
  }

  dblabInstancesButtonHandler = (org: string | number, project: string) => {
    return () => {
      this.props.history.push(Urls.linkDbLabInstances({ org, project }))
    }
  }

  joeInstancesButtonHandler = (org: string | number, project: string) => {
    return () => {
      this.props.history.push(Urls.linkJoeInstances({ org, project }))
    }
  }

  checkupReportsButtonHandler = (org: string | number, project: string) => {
    return () => {
      this.props.history.push(Urls.linkReports({ org, project }))
    }
  }

  render() {
    const renderProjects = this.props.onlyProjects

    if (renderProjects) {
      return this.renderProjects()
    }

    // TODO(anatoly): Move organization to a separate page component.
    return this.renderOrgs()
  }

  renderProjects() {
    const { classes, org } = this.props
    const projectsData =
      this.state && this.state.data && this.state.data.projects
        ? this.state.data.projects
        : null
    const orgId = this.props.orgId

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: 'Dashboard' }]}
      />
    )

    const pageTitle = (
      <ConsolePageTitle
        title="Dashboard"
        information={
          'Project is a workspace for a specific Postgres cluster. ' +
          'Currently projects can be created only during Checkup agent, or ' +
          'Database Lab, or Joe instance configuration.'
        }
      />
    )

    if (projectsData && projectsData.error) {
      return (
        <>
          {breadcrumbs}
          <ErrorWrapper />
        </>
      )
    }

    if (!projectsData || !projectsData.data || projectsData.orgId !== orgId) {
      return (
        <>
          {breadcrumbs}
          <PageSpinner />
        </>
      )
    }

    const projects = projectsData.data

    const dblabPermitted = this.props.orgPermissions.dblabInstanceCreate
    const checkupPermitted = this.props.orgPermissions.checkupReportConfigure

    const addDblabInstanceButton = (
      <ConsoleButtonWrapper
        disabled={!dblabPermitted}
        variant="contained"
        color="primary"
        onClick={this.addDblabInstanceButtonHandler}
        title={
          dblabPermitted
            ? 'Add a new Database Lab instance'
            : messages.noPermission
        }
      >
        Add instance
      </ConsoleButtonWrapper>
    )

    const addCheckupAgentButton = (
      <ConsoleButtonWrapper
        disabled={!checkupPermitted}
        variant="contained"
        color="primary"
        onClick={this.addCheckupAgentButtonHandler}
        title={
          checkupPermitted ? 'Add a new Checkup agent' : messages.noPermission
        }
      >
        Add agent
      </ConsoleButtonWrapper>
    )

    let table = (
      <StubContainer className={classes.stubContainerProjects}>
        <ProductCardWrapper
          inline
          className={classes.productCardProjects}
          title={'Setup Database Lab Engine'}
          actions={[
            {
              id: 'addDblabInstanceButton',
              content: addDblabInstanceButton,
            },
          ]}
          icon={icons.databaseLabLogo}
        >
          <p>
            Clone multi-terabyte databases in seconds and use them to test your
            database migrations, optimize SQL, or deploy full-size staging apps.
            Start here to work with all Database Lab tools.
            <Link to={settings.rootUrl + '/docs/database-lab'} target="_blank">
              Learn more
            </Link>
            .
          </p>
        </ProductCardWrapper>
        <ProductCardWrapper
          inline
          className={classes.productCardProjects}
          title={'Configure automated checkups'}
          actions={[
            {
              id: 'addCheckupAgentButton',
              content: addCheckupAgentButton,
            },
          ]}
          icon={icons.checkupLogo}
        >
          <p>
            Automated routine checkup for your PostgreSQL databases. Configure
            Checkup agent to start collecting reports (
            <Link to={settings.rootUrl + '/docs/checkup'} target="_blank">
              Learn more
            </Link>
            ).
          </p>
        </ProductCardWrapper>
      </StubContainer>
    )

    if (projects.length > 0) {
      table = (
        <HorizontalScrollContainer>
          <Table className={classes.table}>
            <TableHead>
              <TableRow className={classes.row}>
                <TableCell>Project</TableCell>
                <TableCell>Activity</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {projects.map((p) => {
                return (
                  <TableRow hover className={classes.row} key={p.id}>
                    <TableCell className={classes.cell}>{p.name}</TableCell>
                    <TableCell className={classes.cell}>
                      <Button
                        variant="outlined"
                        className={classes.activityButton}
                        onClick={this.dblabInstancesButtonHandler(org, p.alias)}
                      >
                        Database Lab instances
                      </Button>
                      <Button
                        variant="outlined"
                        className={classes.activityButton}
                        onClick={this.joeInstancesButtonHandler(org, p.alias)}
                      >
                        Joe instances
                      </Button>
                      <Button
                        variant="outlined"
                        className={classes.activityButton}
                        onClick={this.checkupReportsButtonHandler(org, p.alias)}
                      >
                        Checkup reports
                      </Button>
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </HorizontalScrollContainer>
      )
    }

    let onboarding = null
    if (
      this.state.data &&
      this.state.data.userProfile &&
      this.state.data.userProfile.data &&
      this.state.data.userProfile.data.orgs &&
      this.state.data.userProfile.data.orgs[org] &&
      this.state.data.userProfile.data.orgs[org].onboarding_text
    ) {
      onboarding = (
        <div>
          <Grid container spacing={2} id="usefulContainer">
            <Grid item xs={12} sm={6}>
              <div className={classes.onboardingCard}>
                <h1>Getting started</h1>
                <ReactMarkdown
                  className={classes.onboarding}
                  children={
                    this.state.data.userProfile.data.orgs[org].onboarding_text
                  }
                  rehypePlugins={[rehypeRaw]}
                  remarkPlugins={[remarkGfm]}
                  components={{
                    a: (props) => {
                      const { href, target, children } = props
                      return (
                        <Link to={href as string} target={target}>
                          {children as string}
                        </Link>
                      )
                    },
                  }}
                />
              </div>
            </Grid>
            <Grid item xs={12} sm={6}>
              <div className={classes.onboardingCard}>
                <h1>Useful links</h1>
                <ReactMarkdown
                  className={classes.onboarding}
                  children={
                    this.state.data.userProfile.data.platform_onboarding_text
                  }
                  rehypePlugins={[rehypeRaw]}
                  remarkPlugins={[remarkGfm]}
                  components={{
                    a: (props) => {
                      const { href, target, children } = props
                      return (
                        <Link to={href as string} target={target}>
                          {children as string}
                        </Link>
                      )
                    },
                  }}
                />
              </div>
            </Grid>
          </Grid>
        </div>
      )
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {onboarding}

        {table}
      </div>
    )
  }

  renderOrgs() {
    const { classes } = this.props
    const profile =
      this.state && this.state.data ? this.state.data.userProfile : null
    const useDemoData =
      this.state && this.state.data ? this.state.data.useDemoData : null
    const profileUpdateInitAfterDemo =
      this.state && this.state.data && this.state.data.dashboard
        ? this.state.data.dashboard.profileUpdateInitAfterDemo
        : null

    // Show organizations.
    if (this.state && this.state.data.projects?.error) {
      return (
        <div>
          <ErrorWrapper />
        </div>
      )
    }

    if (
      !profile ||
      profile.isProcessing ||
      (profile && !profile.data) ||
      !useDemoData ||
      useDemoData.isProcessing ||
      (useDemoData.isProcessed && !profileUpdateInitAfterDemo)
    ) {
      return (
        <>
          <PageSpinner />
        </>
      )
    }

    const useDemoDataButton = (
      <ConsoleButtonWrapper
        variant="contained"
        color="primary"
        onClick={this.useDemoDataButtonHandler}
        id="useDemoDataButton"
        title=""
      >
        Join demo organization
      </ConsoleButtonWrapper>
    )

    const createOrgButton = (
      <ConsoleButtonWrapper
        variant="outlined"
        color="primary"
        onClick={this.addOrgButtonHandler}
        id="createOrgButton"
        title=""
      >
        Create new organization
      </ConsoleButtonWrapper>
    )

    const orgsPlaceholder = (
      <StubContainer>
        <ProductCardWrapper
          inline
          title={'Create or join an organization'}
          actions={[
            {
              id: 'useDemoDataButton',
              content: useDemoDataButton,
            },
            {
              id: 'createOrgButton',
              content: createOrgButton,
            },
          ]}
        >
          <p>
            An organization represents a workspace for you and your colleagues.
            Organizations allow you to manage users and collaborate across
            multiple projects.
          </p>
          <p>
            You can create a new organization, join the demo organization or ask
            existing members of your organization to invite you.
          </p>
        </ProductCardWrapper>
      </StubContainer>
    )

    const pageActions = []
    if (!profile.data?.orgs || !profile.data?.orgs[settings.demoOrgAlias]) {
      pageActions.push(useDemoDataButton)
    }
    pageActions.push(createOrgButton)

    return (
      <div className={classes.root}>
        <ConsolePageTitle
          top={true}
          title="Your organizations"
          information="Your own organizations and organizations of which you are a member"
          actions={pageActions}
        />

        {profile.data?.orgs && Object.keys(profile.data?.orgs).length > 0 ? (
          <HorizontalScrollContainer>
            <Table className={classes.table} id="orgsTable">
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell className={classes.nameColumn}>
                    Organization
                  </TableCell>
                  <TableCell>Projects count</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {Object.keys(profile.data?.orgs).map((index) => {
                  return (
                    <TableRow
                      hover
                      className={classes.row}
                      key={profile.data?.orgs[index].id}
                      onClick={(event) =>
                        this.handleClick(event, profile.data?.orgs[index].alias)
                      }
                      style={{ cursor: 'pointer' }}
                      data-org-id={profile.data?.orgs[index].id}
                      data-org-alias={profile.data?.orgs[index].alias}
                    >
                      <TableCell className={classes.nameColumn}>
                        <NavLink to={'/' + profile.data?.orgs[index].alias}>
                          {profile.data?.orgs[index].name}
                        </NavLink>
                      </TableCell>
                      <TableCell className={classes.cell}>
                        <NavLink
                          to={
                            '/' + profile.data?.orgs[index].alias + '/projects'
                          }
                        >
                          {profile.data?.orgs[index].projects
                            ? Object.keys(profile.data?.orgs[index]?.projects)
                                .length
                            : '0'}
                        </NavLink>
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>
        ) : (
          orgsPlaceholder
        )}
      </div>
    )
  }
}

export default Dashboard
