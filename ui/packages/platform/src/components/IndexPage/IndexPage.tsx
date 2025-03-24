/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component, useState } from 'react'
import { Switch, Route, NavLink, Redirect } from 'react-router-dom'
import {
  AppBar,
  Toolbar,
  Typography,
  Divider,
  IconButton,
  ListItem,
  List,
  Drawer,
  Collapse,
  Box,
  ListItemText
} from '@material-ui/core'
import qs from 'qs'

import { icons } from '@postgres.ai/shared/styles/icons'
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner'
import { GatewayLink } from '@postgres.ai/shared/components/GatewayLink'
import {
  OrganizationWrapperProps,
  OrganizationMenuProps,
  ProjectWrapperProps,
  QueryParamsType,
  UserProfile,
  OrgPermissions,
  ClassesType,
  RefluxTypes,
} from '@postgres.ai/platform/src/components/types'
import { ROUTES } from 'config/routes'
import { Instance } from 'pages/Instance'
import { Clone } from 'pages/Clone'
import { CreateClone } from 'pages/CreateClone'
import { ProfileWrapper } from 'pages/Profile/ProfileWrapper'
import { SignInWrapper } from 'pages/SignIn/SignInWrapper'

import { SideNav } from 'components/SideNav'
import { ContentLayout } from 'components/ContentLayout'
import { ConsoleButtonWrapper } from 'components/ConsoleButton/ConsoleButtonWrapper'
import { DbLabInstanceFormWrapper } from 'components/DbLabInstanceForm/DbLabInstanceFormWrapper'
import { AddDbLabInstanceFormWrapper } from 'components/AddDbLabInstanceFormWrapper/AddDbLabInstanceFormWrapper'
import { DbLabInstancesWrapper } from 'components/DbLabInstances/DbLabInstancesWrapper'
import { DbLabInstanceFormInstallWrapper } from 'components/DbLabInstanceInstallForm/DbLabInstanceInstallFormWrapper'
import { DbLabSessionWrapper } from 'components/DbLabSession/DbLabSessionWrapper'
import { DbLabSessionsWrapper } from 'components/DbLabSessions/DbLabSessionsWrapper'
import { JoeInstanceFormWrapper } from 'components/JoeInstanceForm/JoeInstanceFormWrapper'
import { JoeInstanceWrapper } from 'pages/JoeInstance/JoeInstanceWrapper'
import { JoeInstancesWrapper } from 'components/JoeInstances/JoeInstancesWrapper'
import { ReportFileWrapper } from 'components/ReportFile/ReportFileWrapper'
import { ReportWrapper } from 'components/Report/ReportWrapper'
import { ReportsWrapper } from 'components/Reports/ReportsWrapper'
import { JoeSessionCommandWrapper } from 'pages/JoeSessionCommand/JoeSessionCommandWrapper'
import { JoeHistoryWrapper } from 'components/JoeHistory/JoeHistoryWrapper'
import { BillingWrapper } from 'components/Billing/BillingWrapper'
import { AddMemberFormWrapper } from 'components/AddMemberForm/AddMemberFormWrapper'
import { OrgFormWrapper } from 'components/OrgForm/OrgFormWrapper'
import { AuditWrapper } from 'components/Audit/AuditWrapper'
import { OrgMembersWrapper } from 'components/OrgMembers/OrgMembersWrapper'
import { AccessTokensWrapper } from 'components/AccessTokens/AccessTokensWrapper'
import { CheckupAgentFormWrapper } from 'components/CheckupAgentForm/CheckupAgentFormWrapper'
import { ExplainVisualizationWrapper } from 'components/ExplainVisualization/ExplainVisualizationWrapper'
import { DashboardWrapper } from 'components/Dashboard/DashboardWrapper'
import { LoginDialogWrapper } from 'components/LoginDialog/LoginDialogWrapper'
import { NotificationWrapper } from 'components/Notification/NotificationWrapper'
import { SharedUrlWrapper } from 'components/SharedUrl/SharedUrlWrapper'
import { ShareUrlDialogWrapper } from 'components/ShareUrlDialog/ShareUrlDialogWrapper'
import { BotWrapper } from "pages/Bot/BotWrapper";
import { ConsultingWrapper } from "pages/Consulting/ConsultingWrapper";

import Actions from '../../actions/actions'
import JoeConfig from '../JoeConfig'
import Permissions from '../../utils/permissions'
import Store from '../../stores/store'
import Urls from '../../utils/urls'
import settings from '../../utils/settings'
import { AppUpdateBanner } from '../AppUpdateBanner'
import { IndexPageProps } from 'components/IndexPage/IndexPageWrapper'
import { PostgresClusterWrapper } from 'components/PostgresClusterForm/PostgresClusterWrapper'
import { PostgresClusterInstallWrapper } from 'components/PostgresClusterInstallForm/PostgresClusterInstallWrapper'
import { PostgresClustersWrapper } from 'components/PostgresClusters/PostgresClustersWrapper'
import cn from "classnames";
import { BotSettingsFormWrapper } from "../BotSettingsForm/BotSettingsFormWrapper";
import { AuditSettingsFormWrapper } from "../AuditSettingsForm/AuditSettingsFormWrapper";
import { ExpandLess, ExpandMore } from "@material-ui/icons";
import { DBLabSettingsFormWrapper } from "../DBLabSettingsForm/DBLabSettingsFormWrapper";


interface IndexPageWithStylesProps extends IndexPageProps {
  classes: ClassesType
}

interface IndexPageState {
  data: {
    auth: {
      token: string
      userId: number
      isProcessed: boolean
    } | null
    userProfile: UserProfile | null
  } | null
  mobileOpen: boolean
}

function ProjectWrapper(parentProps: Omit<ProjectWrapperProps, 'classes'>) {
  const project = parentProps.match.params.project
  const queryParams = qs.parse(parentProps.location.search, {
    ignoreQueryPrefix: true,
  })
  const { session, command, author, fingerprint, search, is_favorite } =
    queryParams

  const errorMessage = (
    <div>
      404 Project <strong>{project}</strong> not found
      <br />
    </div>
  )

  if (
    !project ||
    !parentProps.orgData?.projects ||
    !parentProps.orgData.projects[project]
  ) {
    return errorMessage
  }

  const projectId = parentProps.orgData.projects[project].id

  const customProps = {
    project,
    projectId,
    org: parentProps.org,
    orgId: parentProps.orgId,
    orgData: parentProps.orgData,
    orgPermissions: parentProps.orgPermissions,
    userIsOwner: parentProps.userIsOwner,
    env: parentProps.env,
    envData: parentProps.envData,
    auth: parentProps.auth,
    raw: parentProps.raw,
  }

  const queryProps = {
    session,
    command,
    author,
    fingerprint,
    search,
    is_favorite,
  } as QueryParamsType

  return (
    <Switch>
      <Route
        path="/:org/:project/instances/add"
        render={(props) => (
          <AddDbLabInstanceFormWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/pg/create"
        render={(props) => (
          <PostgresClusterWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/pg/install"
        render={(props) => (
          <PostgresClusterInstallWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/instances/create"
        render={(props) => (
          <DbLabInstanceFormWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/instances/install"
        render={(props) => (
          <DbLabInstanceFormInstallWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/instances/edit/:instanceId"
        render={(props) => (
          <AddDbLabInstanceFormWrapper edit {...props} {...customProps} />
        )}
      />
      <Route
        exact
        path={ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.ADD.createPath()}
      >
        <CreateClone />
      </Route>
      <Route
        path={ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.CLONE.createPath()}
      >
        <Clone />
      </Route>
      <Route path={ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.createPath()}>
        <Instance />
      </Route>
      <Route
        path="/:org/:project/instances"
        render={(props) => (
          <DbLabInstancesWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/pg"
        render={(props) => (
          <PostgresClustersWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/observed-sessions/:sessionId"
        render={(props) => <DbLabSessionWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/observed-sessions"
        render={(props) => <DbLabSessionsWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/joe-instances/add"
        render={(props) => (
          <JoeInstanceFormWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/joe-instances/:instanceId"
        render={(props) => <JoeInstanceWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/bot/:instanceId"
        render={(props) => <JoeInstanceWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/joe-instances"
        render={(props) => <JoeInstancesWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/reports/:reportId/files/:fileId/:fileType"
        render={(props) => (
          <ReportFileWrapper
            fileType={props.match.params.fileType}
            reportId={props.match.params.reportId}
            {...props}
            {...customProps}
          />
        )}
      />
      <Route
        path="/:org/:project/reports/:reportId/:type"
        render={(props) => <ReportWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/reports/:reportId/"
        render={(props) => <ReportWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/reports"
        render={(props) => <ReportsWrapper {...props} {...customProps} />}
      />
      <Route
        path="/:org/:project/sessions/:sessionId/commands/:commandId"
        render={(props) => (
          <JoeSessionCommandWrapper {...props} {...customProps} />
        )}
      />
      <Route
        path="/:org/:project/sessions/:sessionId"
        render={(props) => (
          <JoeHistoryWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/:project/sessions"
        render={(props) => (
          <JoeHistoryWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/:project"
        render={(props) => <ReportsWrapper {...props} {...customProps} />}
      />
    </Switch>
  )
}

function OrganizationMenu(parentProps: OrganizationMenuProps) {
  const [activeMenuItems, setActiveMenuItems] = useState<Set<string>>(new Set());

  const handleOpenMenuItem = (e: React.MouseEvent<HTMLDivElement, MouseEvent>, value: string) => {
    e.stopPropagation()
    setActiveMenuItems((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(value)) {
        newSet.delete(value);
      } else {
        newSet.add(value);
      }
      return newSet;
    });
  };

  if (
    parentProps.env &&
    parentProps.env.data &&
    parentProps.match.params.org &&
    parentProps.env.data.orgs &&
    parentProps.env.data.orgs[parentProps.match.params.org]
  ) {
    let org = parentProps.match.params.org
    let isBlocked = false
    let orgData = null

    let orgPermissions: OrgPermissions = {}
    if (
      parentProps.env.data &&
      parentProps.env.data.orgs &&
      parentProps.env.data.orgs[org]
    ) {
      orgData = parentProps.env.data.orgs[org]
      isBlocked = orgData.is_blocked
      orgPermissions = Permissions.getPermissions(orgData)
    }

    return (
      <div className={parentProps.classes.menuContainer}>
        <div>
          <div className={parentProps.classes.orgHeaderContainer}>
            <span className={parentProps.classes.orgHeader}>Organization</span>
            <NavLink to={ROUTES.ROOT.path}>
              <span className={parentProps.classes.orgSwitcher} id="menuSwitch">
                switch
              </span>
            </NavLink>
          </div>
          <div className={parentProps.classes.orgNameContainer}>
            <span className={parentProps.classes.orgName}>
              {parentProps.env.data.orgs[org].name}
            </span>
          </div>
        </div>

        <Divider />
        <div className={parentProps.classes.navMenuContainer}>
          <List component="nav" className={parentProps.classes.navMenu}>
            <ListItem
              button
              className={parentProps.classes.menuSectionHeader}
              disabled={isBlocked}
              id="menuDashboard"
            >
              <NavLink
                className={parentProps.classes.menuSectionHeaderLink}
                activeClassName={parentProps.classes.menuSectionHeaderActiveLink}
                to={'/' + org}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.dashboardIcon}
                </span>
                Dashboard
              </NavLink>
            </ListItem>

            <ListItem
              button
              className={parentProps.classes.menuSectionHeader}
              disabled={isBlocked}
              id="menuAiBotTitle"
            >
              <NavLink
                className={parentProps.classes.menuSectionHeaderLink}
                activeClassName={cn(parentProps.classes.menuSectionHeaderActiveLink, parentProps.classes.menuSingleSectionHeaderActiveLink)}
                to={'/' + org + '/assistant'}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.aiBotIcon}
                </span>
                AI Assistant<span className={cn(parentProps.classes.menuItemLabel, parentProps.classes.headerLinkMenuItemLabel)}>NEW</span>
              </NavLink>
            </ListItem>
            <ListItem
              button
              className={cn(parentProps.classes.menuSectionHeader, parentProps.classes.menuSectionHeaderCollapsible)}
              disabled={isBlocked}
              id="menuDblabTitle"
              onClick={(e) => handleOpenMenuItem(e, 'dblab')}
            >
              <ListItemText
                className={cn(
                  parentProps.classes.menuSectionHeaderLink,
                  parentProps.classes.menuSectionHeaderLinkCollapsible,
                  {[parentProps.classes.menuSectionHeaderActiveLink]: activeMenuItems.has('dblab')}
                )}
                classes={{primary: parentProps.classes.menuSectionHeaderLinkText}}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.databaseLabIcon}
                </span>
                Database Lab
                {activeMenuItems.has('dblab')
                  ? <ExpandLess className={parentProps.classes.menuSectionHeaderExpandIcon} />
                  : <ExpandMore className={parentProps.classes.menuSectionHeaderExpandIcon} />}
              </ListItemText>
            </ListItem>
            <Collapse
              in={activeMenuItems.has('dblab')}
              timeout="auto"
              unmountOnExit
            >
              <List component="div" disablePadding>
                <ListItem
                  button
                  className={parentProps.classes.menuItem}
                  disabled={isBlocked}
                  id="menuDblabInstances"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/instances'}
                  >
                    Instances
                  </NavLink>
                </ListItem>
                <ListItem
                  button
                  className={parentProps.classes.menuItem}
                  disabled={isBlocked}
                  id="menuDblabSessions"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/observed-sessions'}
                  >
                    Observed sessions
                  </NavLink>
                </ListItem>
              </List>
            </Collapse>
            <ListItem
              button
              className={cn(parentProps.classes.menuSectionHeader, parentProps.classes.menuSectionHeaderCollapsible)}
              disabled={isBlocked}
              id="menuJoeTitle"
              onClick={(e) => handleOpenMenuItem(e, 'sqlOptimization')}
            >
              <ListItemText
                className={cn(
                  parentProps.classes.menuSectionHeaderLink,
                  parentProps.classes.menuSectionHeaderLinkCollapsible,
                  {[parentProps.classes.menuSectionHeaderActiveLink]: activeMenuItems.has('sqlOptimization')}
                )}
                classes={{primary: parentProps.classes.menuSectionHeaderLinkText}}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.sqlOptimizationIcon}
                </span>
                Joe bot
                {activeMenuItems.has('sqlOptimization')
                  ? <ExpandLess className={parentProps.classes.menuSectionHeaderExpandIcon} />
                  : <ExpandMore className={parentProps.classes.menuSectionHeaderExpandIcon} />}
              </ListItemText>
            </ListItem>
            <Collapse
              in={activeMenuItems.has('sqlOptimization')}
              timeout="auto"
              unmountOnExit
            >
              <List component="div" disablePadding>
                <ListItem
                  button
                  className={parentProps.classes.menuItem}
                  disabled={isBlocked}
                  id="menuJoeInstances"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/joe-instances'}
                  >
                    Ask Joe
                  </NavLink>
                </ListItem>
                <ListItem
                  button
                  className={parentProps.classes.menuItem}
                  disabled={isBlocked}
                  id="menuJoeSessions"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/sessions?'}
                  >
                    History
                  </NavLink>
                </ListItem>
                {false && (
                  <ListItem
                    button
                    className={parentProps.classes.menuItem}
                    disabled={isBlocked}
                    id="menuJoeExplain"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/explain'}
                    >
                      Plan visualization
                    </NavLink>
                  </ListItem>
                )}
              </List>
            </Collapse>
            {/* <ListItem
              button
              className={parentProps.classes.menuSectionHeader}
              disabled={isBlocked}
              id="menuPostgresTitle"
            >
              <NavLink
                className={parentProps.classes.menuSectionHeaderLink}
                activeClassName={parentProps.classes.menuSectionHeaderActiveLink}
                to={'/' + org + '/pg'}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.postgresSQLIcon}
                </span>
                Postgres
              </NavLink>
            </ListItem>
            <ListItem
              button
              className={parentProps.classes.menuItem}
              disabled={isBlocked}
              id="menuDblabClusters"
            >
              <NavLink
                className={parentProps.classes.menuItemLink}
                activeClassName={parentProps.classes.menuItemActiveLink}
                to={'/' + org + '/pg'}
              >
                Clusters
              </NavLink>
            </ListItem> */}

            <ListItem
              button
              className={cn(parentProps.classes.menuSectionHeader, parentProps.classes.menuSectionHeaderCollapsible)}
              disabled={isBlocked}
              id="menuCheckupTitle"
              onClick={(e) => handleOpenMenuItem(e, 'checkup')}
            >
              <ListItemText
                className={cn(
                  parentProps.classes.menuSectionHeaderLink,
                  parentProps.classes.menuSectionHeaderLinkCollapsible,
                  {[parentProps.classes.menuSectionHeaderActiveLink]: activeMenuItems.has('checkup')}
                )}
                classes={{primary: parentProps.classes.menuSectionHeaderLinkText}}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.checkupIcon}
                </span>
                Checkup
                {activeMenuItems.has('checkup')
                  ? <ExpandLess className={parentProps.classes.menuSectionHeaderExpandIcon} />
                  : <ExpandMore className={parentProps.classes.menuSectionHeaderExpandIcon} />}
              </ListItemText>
            </ListItem>
            <Collapse
              in={activeMenuItems.has('checkup')}
              timeout="auto"
              unmountOnExit
            >
              <List component="div" disablePadding>
                <ListItem
                  button
                  className={parentProps.classes.menuItem}
                  disabled={isBlocked}
                  id="menuCheckupReports"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/reports'}
                  >
                    Reports
                  </NavLink>
                </ListItem>
              </List>
            </Collapse>
            <ListItem
              button
              className={parentProps.classes.menuSectionHeader}
              disabled={isBlocked}
              id="menuConsultingTitle"
            >
              <NavLink
                className={parentProps.classes.menuSectionHeaderLink}
                activeClassName={cn(parentProps.classes.menuSectionHeaderActiveLink, parentProps.classes.menuSingleSectionHeaderActiveLink)}
                to={'/' + org + '/consulting'}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.consultingIcon}
                </span>
                Consulting
              </NavLink>
            </ListItem>
            <ListItem
              button
              className={cn(parentProps.classes.menuSectionHeader, parentProps.classes.menuSectionHeaderCollapsible)}
              disabled={isBlocked}
              id="menuSettingsTitle"
              onClick={(e) => handleOpenMenuItem(e, 'settings')}
            >
              <ListItemText
                className={cn(
                  parentProps.classes.menuSectionHeaderLink,
                  parentProps.classes.menuSectionHeaderLinkCollapsible,
                  {[parentProps.classes.menuSectionHeaderActiveLink]: activeMenuItems.has('settings')}
                )}
                classes={{primary: parentProps.classes.menuSectionHeaderLinkText}}
              >
                <span className={parentProps.classes.menuSectionHeaderIcon}>
                  {icons.settingsIcon}
                </span>
                Manage
                {activeMenuItems.has('settings')
                  ? <ExpandLess className={parentProps.classes.menuSectionHeaderExpandIcon} />
                  : <ExpandMore className={parentProps.classes.menuSectionHeaderExpandIcon} />}
              </ListItemText>
            </ListItem>
            <Collapse
              in={activeMenuItems.has('settings')}
              timeout="auto"
              unmountOnExit
            >
              <List component="div" disablePadding>
                {orgPermissions && orgPermissions.settingsOrganizationUpdate && (
                  <ListItem
                    button
                    className={parentProps.classes.menuItem}
                    disabled={isBlocked}
                    id="menuSettingsGeneral"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/settings'}
                    >
                      General settings
                    </NavLink>
                  </ListItem>
                )}
                {orgPermissions && orgPermissions.settingsOrganizationUpdate && (
                  <ListItem
                    disabled={isBlocked}
                    button
                    className={parentProps.classes.menuItem}
                    id="menuSettingsBot"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/assistant-settings'}
                    >
                      AI Assistant settings
                    </NavLink>
                  </ListItem>
                )}
                <ListItem
                  disabled={isBlocked}
                  button
                  className={parentProps.classes.menuItem}
                  id="menuSettingsMembers"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/members'}
                  >
                    Members
                  </NavLink>
                </ListItem>
                <ListItem
                  button
                  className={parentProps.classes.menuItem}
                  disabled={isBlocked}
                  id="menuSettingsTokens"
                >
                  <NavLink
                    className={parentProps.classes.menuItemLink}
                    activeClassName={parentProps.classes.menuItemActiveLink}
                    to={'/' + org + '/tokens'}
                  >
                    Access tokens
                  </NavLink>
                </ListItem>
                {orgData !== null && Permissions.isAdmin(orgData) && (
                  <ListItem
                    button
                    className={parentProps.classes.menuItem}
                    id="menuSettingsBilling"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/billing'}
                    >
                      Billing
                    </NavLink>
                  </ListItem>
                )}
                {orgData !== null && Permissions.isAdmin(orgData) && (
                  <ListItem
                    button
                    className={parentProps.classes.menuItem}
                    id="menuSettingsDBLab"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/dblab-settings'}
                    >
                      DBLab settings
                    </NavLink>
                  </ListItem>
                )}
                {orgPermissions && orgPermissions.auditLogView && (
                  <ListItem
                    disabled={
                      (orgPermissions && !orgPermissions.auditLogView) || isBlocked
                    }
                    button
                    className={parentProps.classes.menuItem}
                    id="menuSettingsAuditLogs"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/audit'}
                    >
                      Audit logs
                    </NavLink>
                  </ListItem>
                )}
                {orgData !== null && orgPermissions && Permissions.isAdmin(orgData) && orgPermissions.auditLogView && (
                  <ListItem
                    disabled={
                      (orgPermissions && !orgPermissions.auditLogView) || isBlocked
                    }
                    button
                    className={parentProps.classes.menuItem}
                    id="menuSettingsAuditSettings"
                  >
                    <NavLink
                      className={parentProps.classes.menuItemLink}
                      activeClassName={parentProps.classes.menuItemActiveLink}
                      to={'/' + org + '/audit-settings'}
                    >
                      Audit settings
                    </NavLink>
                  </ListItem>
                )}
              </List>
            </Collapse>
          </List>
        </div>
      </div>
    )
  }

  return null
}

function OrganizationWrapper(parentProps: OrganizationWrapperProps) {
  const org = parentProps.match.params.org
  const envData =
    parentProps.env && parentProps.env.data
      ? parentProps.env.data
      : { orgs: null }

  const errorMessage = (
    <div>
      404 Organization <strong>{org}</strong> not found
      <br />
    </div>
  )

  const preloader = <PageSpinner />

  if (org === 'auth' || !envData.orgs) {
    return preloader
  }

  if (!org || (org && envData.orgs && !envData.orgs[org])) {
    return errorMessage
  }

  const env = parentProps.env
  const auth = parentProps.auth
  const raw = parentProps.raw
  const orgId = envData.orgs[org].id
  const orgData = envData.orgs[org]
  const isBlocked = orgData.is_blocked
  const orgPermissions = Permissions.getPermissions(orgData)
  const userIsOwner =
    !env || !auth || !orgId
      ? false
      : Permissions.userIsOwner(auth.userId, orgId, env.data)

  const queryParams = qs.parse(parentProps.location.search, {
    ignoreQueryPrefix: true,
  })
  const {
    session,
    command,
    author,
    fingerprint,
    project,
    search,
    is_favorite,
  } = queryParams
  const projectId = project && orgData.projects[project.toString()]?.id

  const customProps = {
    project,
    projectId,
    org,
    orgId,
    orgData,
    orgPermissions,
    userIsOwner,
    env,
    envData,
    auth,
    raw,
  }

  const queryProps = {
    session,
    command,
    author,
    fingerprint,
    search,
    is_favorite,
  } as QueryParamsType

  if (isBlocked && Permissions.isAdmin(orgData)) {
    return (
      <Switch>
        <Route
          path="/:org/billing"
          render={(props) => (
            <BillingWrapper
              short={false}
              {...props}
              {...customProps}
              {...queryProps}
            />
          )}
        />
        <Route
          path={ROUTES.ROOT.path}
          render={(props) => (
            <BillingWrapper
              {...props}
              {...customProps}
              {...queryProps}
              short={true}
            />
          )}
        />
      </Switch>
    )
  }

  return (
    <Switch>
      <Route
        path="/:org/instances/add"
        render={(props) => (
          <AddDbLabInstanceFormWrapper
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route
        path="/:org/pg/create"
        render={(props) => (
          <PostgresClusterWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/pg/install"
        render={(props) => (
          <PostgresClusterInstallWrapper
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route
        path="/:org/instances/create"
        render={(props) => (
          <DbLabInstanceFormWrapper
            userID={auth?.userId}
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route
        path="/:org/instances/install"
        render={(props) => (
          <DbLabInstanceFormInstallWrapper
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route exact path={ROUTES.ORG.INSTANCES.INSTANCE.CLONES.ADD.createPath()}>
        <CreateClone />
      </Route>
      <Route path={ROUTES.ORG.INSTANCES.INSTANCE.CLONES.CLONE.createPath()}>
        <Clone />
      </Route>
      <Route path={ROUTES.ORG.INSTANCES.INSTANCE.createPath()}>
        <Instance />
      </Route>
      <Route
        path="/:org/instances"
        render={(props) => (
          <DbLabInstancesWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/pg"
        render={(props) => (
          <PostgresClustersWrapper
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route
        path="/:org/observed-sessions/:sessionId"
        render={(props) => (
          <DbLabSessionWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/observed-sessions"
        render={(props) => (
          <DbLabSessionsWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/joe-instances/add"
        render={(props) => (
          <JoeInstanceFormWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/joe-instances/:instanceId"
        render={(props) => (
          <JoeInstanceWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/assistant/:threadId"
        exact
        render={(props) => (
          <BotWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/assistant"
        exact
        render={(props) => (
          <BotWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/bot/:threadId"
        exact
        render={(props) => {
          const { org, threadId } = props.match.params;
          return <Redirect to={`/${org}/assistant/${threadId}`} />;
        }}
      />
      <Route
        path="/:org/bot"
        exact
        render={(props) => {
          const { org } = props.match.params;
          return <Redirect to={`/${org}/assistant`} />;
        }}
      />
      <Route
        path="/:org/consulting"
        exact
        render={(props) => (
          <ConsultingWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/joe-instances"
        render={(props) => (
          <JoeInstancesWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/members/add"
        render={(props) => (
          <AddMemberFormWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/settings"
        render={(props) => (
          <OrgFormWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/assistant-settings"
        render={(props) => (
          <BotSettingsFormWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/audit-settings"
        render={(props) => (
          <AuditSettingsFormWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/dblab-settings"
        render={(props) => (
          <DBLabSettingsFormWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/billing"
        render={(props) => (
          <BillingWrapper
            short={false}
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route
        path="/:org/audit"
        render={(props) => (
          <AuditWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/members"
        render={(props) => (
          <OrgMembersWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/reports"
        render={(props) => (
          <ReportsWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path={ROUTES.ORG.TOKENS.createPath()}
        render={(props) => (
          <AccessTokensWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/checkup-config"
        render={(props) => (
          <CheckupAgentFormWrapper
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
      <Route
        path="/:org/joe-config"
        render={(props) => (
          <JoeConfig {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/explain"
        render={() => <ExplainVisualizationWrapper />}
      />
      <Route
        path="/:org/sessions"
        render={(props) => (
          <JoeHistoryWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org/:project"
        render={(props) => (
          <ProjectWrapper {...props} {...customProps} {...queryProps} />
        )}
      />
      <Route
        path="/:org"
        render={(props) => (
          <DashboardWrapper
            onlyProjects
            {...props}
            {...customProps}
            {...queryProps}
          />
        )}
      />
    </Switch>
  )
}

function SupportMenu(props: { classes: ClassesType }) {
  return (
    <List component="nav" className={props.classes.navBottomFixedMenu}>
      <ListItem button className={props.classes.bottomFixedMenuItem}>
        <a
          className={props.classes.menuSectionHeaderLink}
          target="_blank"
          href={settings.rootUrl + '/docs'}
          rel="noreferrer"
        >
          <span className={props.classes.menuSectionHeaderIcon}>
            {icons.docIcon}
          </span>
          Documentation
        </a>
      </ListItem>
      <ListItem button className={props.classes.bottomFixedMenuItem}>
        <a
          className={props.classes.menuSectionHeaderLink}
          target="_blank"
          href={settings.rootUrl + '/contact'}
          rel="noreferrer"
        >
          <span className={props.classes.menuSectionHeaderIcon}>
            {icons.supportIcon}
          </span>
          Ask support
        </a>
      </ListItem>
    </List>
  )
}

class IndexPage extends Component<IndexPageWithStylesProps, IndexPageState> {
  state = {
    data: {
      auth: {
        token: '',
        userId: 0,
        isProcessed: false,
      },
      userProfile: {
        data: {
          orgs: {},
          info: {
            first_name: '',
            user_name: '',
            email: '',
            is_tos_confirmed: false,
            is_active: false,
            id: null,
          },
        },
        isTosAgreementConfirmProcessing: false,
        isConfirmProcessing: false,
        isConfirmProcessed: false,
      },
    },
    mobileOpen: false,
  }

  unsubscribe: Function
  componentDidMount() {
    const that = this

    document.getElementsByTagName('html')[0].style.overflow = 'hidden'

    this.unsubscribe = (Store.listen as RefluxTypes['listen'])(function () {
      that.setState({ data: this.data })

      // redirect to new organization alias if need
      const env = this.data ? this.data.userProfile : null
      const orgProfile =
        this.data && this.data.orgProfile ? this.data.orgProfile : null
      if (
        orgProfile &&
        orgProfile.prevAlias &&
        orgProfile.data.alias &&
        env &&
        env.data &&
        env.data.orgs[orgProfile.data.alias]
      ) {
        window.location.href = '/' + orgProfile.data.alias + '/settings'
      }

      if (
        (env.isConfirmProcessed || (env.data && env.data.info.is_active)) &&
        Urls.isRequestedPath('confirm')
      ) {
        window.location.href = ROUTES.ROOT.path
      }
    })

    Actions.doAuth(null, null)
  }

  signOut() {
    Actions.signOut()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  handleDrawerToggle = () => {
    window.innerWidth <= 768 &&
      this.setState({ mobileOpen: !this.state.mobileOpen })
  }

  goHrefUrl = (event: {
    target: { getAttribute: (attribute: string) => string }
  }) => {
    this.props.history.push(event.target.getAttribute('hrefurl'))
    return false
  }

  resendCode = () => {
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null

    if (auth?.token) {
      Actions.sendUserCode(auth.token)
    }
  }

  confirmTosAgreement = () => {
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null

    if (auth?.token) {
      Actions.confirmTosAgreement(auth.token)
    }
  }

  addOrgButtonHandler = () => {
    this.props.history.push(ROUTES.CREATE_ORG.path)
  }

  render() {
    const { classes } = this.props
    const auth =
      this.state.data && this.state.data.auth ? this.state.data.auth : null
    const env = this.state.data ? this.state.data.userProfile : null
    const raw =
      this.props &&
      this.props.location &&
      this.props.location.search &&
      this.props.location.search.indexOf('raw') !== -1

    if (auth && auth.isProcessed && !auth.token && !Urls.isSharedUrl()) {
      if (window.location.pathname === '/login') {
        return (
          <div>
            <LoginDialogWrapper />
          </div>
        )
      }

      if (window.location.pathname !== '/signin') {
        this.signOut()
        return null
      }

      return <SignInWrapper />
    }

    const appBarLogo = (
      <Typography color="inherit" noWrap>
        <NavLink to={ROUTES.ROOT.path} className={classes.logo}>
          Postgres.AI Console
        </NavLink>
      </Typography>
    )

    const appBarSignOut = (
      <IconButton
        color="inherit"
        aria-label="Sign out"
        onClick={this.signOut}
        className={classes.navIconSignOut}
      >
        {icons.exitIcon}
      </IconButton>
    )

    const uiUpdate = <AppUpdateBanner />

    let userName = ''
    if (env && env.data && env.data.info) {
      if (env.data.info.first_name) {
        userName = env.data.info.first_name
      } else {
        userName = env.data.info.email
      }
    }

    if (Urls.isSharedUrl()) {
      return (
        <div className={classes.root}>
          <AppBar position="absolute" className={classes.appBar}>
            <Toolbar className={classes.topToolbar}>
              {appBarLogo}
              {auth && auth.token && (
                <Typography color="inherit" noWrap className={classes.userName}>
                  {userName}
                </Typography>
              )}
              {auth && auth.token ? (
                <NavLink
                  to={ROUTES.PROFILE.path}
                  className={classes.navIconArea}
                >
                  <IconButton
                    color="inherit"
                    aria-label="Profile"
                    className={classes.navIconProfile}
                  >
                    {icons.userIcon}
                  </IconButton>
                </NavLink>
              ) : null}
              {auth && auth.token && appBarSignOut}
            </Toolbar>
          </AppBar>
          <ContentLayout>
            <Switch>
              <Route
                path="/shared/:url_uuid"
                render={(props) => <SharedUrlWrapper {...props} />}
              />
              <Redirect from="*" to={ROUTES.ROOT.path} />
            </Switch>
          </ContentLayout>
        </div>
      )
    }

    if (!env || (env && !env.data)) {
      return <PageSpinner />
    }

    if (!env.data.info['is_tos_confirmed']) {
      return (
        <>
          {uiUpdate}

          <div className={classes.tosWrapper}>
            <div className={classes.tosContainer}>
              <p>
                Please, read and agree to our updated
                <br />
                <GatewayLink href="https://postgres.ai/tos" target="_blank">
                  Terms of Service
                </GatewayLink>
                &nbsp;and&nbsp;
                <GatewayLink href="https://postgres.ai/privacy" target="_blank">
                  Privacy Policy
                </GatewayLink>
              </p>

              <ConsoleButtonWrapper
                variant="contained"
                color="primary"
                key="add_dblab_instance"
                disabled={env.isTosAgreementConfirmProcessing}
                onClick={this.confirmTosAgreement}
                className={classes.tosAgree}
                title="I Agree"
              >
                I Agree
              </ConsoleButtonWrapper>
            </div>
            <NotificationWrapper />
          </div>
        </>
      )
    }

    if (!env.data.info['is_active']) {
      if (Urls.isRequestedPath('confirm') && Urls.getRequestParam('code')) {
        if (!env.isConfirmProcessing && !env.isConfirmProcessed) {
          Actions.confirmUserEmail(auth?.token, Urls.getRequestParam('code'))
        }
        if (!env.isConfirmProcessed) {
          return (
            <>
              {uiUpdate}
              <PageSpinner />
            </>
          )
        }

        this.props.history.push(ROUTES.ROOT.path)
      }

      return (
        <>
          {uiUpdate}
          <AppBar position="absolute" className={classes.appBar}>
            <Toolbar className={classes.topToolbar}>
              {appBarLogo}
              {appBarSignOut}
            </Toolbar>
          </AppBar>

          <div className={classes.betaWrapper}>
            <div className={classes.betaContainer}>
              <div style={{ marginBottom: '10px', textAlign: 'center' }}>
                {icons.logo}
              </div>

              <strong>
                Hi{' '}
                {env.data.info['first_name']
                  ? env.data.info['first_name']
                  : env.data.info['user_name']}
                !
              </strong>

              <p>
                Please confirm registration by clicking on a link we have just
                sent to {env.data.info['email']}.
              </p>

              <p>
                If you haven't received the email,&nbsp;
                <GatewayLink onClick={this.resendCode} href="#">
                  click here
                </GatewayLink>
                &nbsp;to resend it.
              </p>
            </div>
            <NotificationWrapper />
          </div>
        </>
      )
    }

    if (raw) {
      return (
        <>
          <Switch>
            <Route
              path="/:org"
              render={(props) => (
                <OrganizationWrapper
                  {...props}
                  env={env}
                  auth={auth}
                  raw={raw}
                  classes={classes}
                />
              )}
            />
            <Redirect from="*" to={ROUTES.ROOT.path} />
          </Switch>
        </>
      )
    }

    const drawer = (
      <div onClick={this.handleDrawerToggle} className={classes.menuPointer}>
        <Divider />
        <Switch>
          <Route exact path={[ROUTES.ROOT.path, ROUTES.PROFILE.path]}>
            <SideNav />
          </Route>

          <Route
            path="/:org"
            render={(props) => (
              <OrganizationMenu {...props} classes={classes} env={env} />
            )}
          />
        </Switch>
        <SupportMenu {...this.props} />
      </div>
    )

    return (
      <>
        {uiUpdate}
        <div className={classes.root}>
          <AppBar position="absolute" className={classes.appBar}>
            <Toolbar className={classes.topToolbar}>
              <IconButton
                color="inherit"
                aria-label="Open menu"
                onClick={this.handleDrawerToggle}
                className={`menu-pointer ${classes.navIconHide}`}
              >
                {icons.menuIcon}
              </IconButton>
              {appBarLogo}
              <Typography color="inherit" noWrap className={classes.userName}>
                {userName}
              </Typography>
              <NavLink to={ROUTES.PROFILE.path} className={classes.navIconArea}>
                <IconButton
                  color="inherit"
                  aria-label="Profile"
                  className={classes.navIconProfile}
                >
                  {icons.userIcon}
                </IconButton>
              </NavLink>
              {appBarSignOut}
            </Toolbar>
          </AppBar>
          <Box
            sx={{
              display: { xs: 'block', sm: 'none', md: 'none' },
            }}
          >
            <Drawer
              className={classes.drawer}
              variant="temporary"
              anchor="left"
              open={this.state.mobileOpen}
              onClose={this.handleDrawerToggle}
              classes={{
                paper: classes.drawerPaper,
              }}
              ModalProps={{
                // Better open performance on mobile.
                keepMounted: true,
              }}
            >
              {drawer}
            </Drawer>
          </Box>
          <Box sx={{ display: { xs: 'none', sm: 'none', md: 'block' } }}>
            <Drawer
              className={classes.drawerContainer}
              variant="permanent"
              open
              classes={{
                paper: classes.drawerPaper,
              }}
            >
              {drawer}
            </Drawer>
          </Box>
          <ContentLayout>
            <ShareUrlDialogWrapper />
            <Switch>
              <Redirect from="/signin" to={ROUTES.ROOT.path} />
              <Redirect from="/login" to={ROUTES.ROOT.path} />
              <Route
                path={ROUTES.PROFILE.path}
                render={() => <ProfileWrapper />}
              />
              <Route
                path={ROUTES.ROOT.path}
                exact
                render={(props) => {
                  if (env.data && env.data?.orgs) {
                    return <DashboardWrapper onlyProjects={false} {...props} />
                  } else {
                    return <OrgFormWrapper mode={'new'} {...props} />
                  }
                }}
              />
              <Route
                path={ROUTES.CREATE_ORG.path}
                render={(props) => <OrgFormWrapper mode={'new'} {...props} />}
              />
              <Route
                path="/:org"
                render={(props) => (
                  <OrganizationWrapper
                    {...props}
                    env={env}
                    auth={auth}
                    classes={classes}
                  />
                )}
              />
            </Switch>
            <NotificationWrapper />
          </ContentLayout>
        </div>
      </>
    )
  }
}

export default IndexPage
