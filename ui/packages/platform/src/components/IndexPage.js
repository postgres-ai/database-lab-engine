/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import { Switch, Route, NavLink, Redirect } from 'react-router-dom';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Drawer from '@material-ui/core/Drawer';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import Divider from '@material-ui/core/Divider';
import List from '@material-ui/core/List';
import Hidden from '@material-ui/core/Hidden';
import IconButton from '@material-ui/core/IconButton';
import { ListItem } from '@material-ui/core';
import { ThemeProvider } from '@material-ui/core/styles';
import Link from '@material-ui/core/Link';
import qs from 'qs';

import { colors } from '@postgres.ai/shared/styles/colors';
import { icons } from '@postgres.ai/shared/styles/icons';
import { theme } from '@postgres.ai/shared/styles/theme';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';

import { ROUTES } from 'config/routes';
import { SideNav } from 'components/SideNav';
import { ContentLayout } from 'components/ContentLayout';
import SignIn from 'pages/SignIn';
import Profile from 'pages/Profile';
import JoeInstance from 'pages/JoeInstance';
import { Instance } from 'pages/Instance';
import { Clone } from 'pages/Clone';
import { CreateClone } from 'pages/CreateClone';
import JoeSessionCommand from 'pages/JoeSessionCommand';

import AccessTokens from './AccessTokens';
import ShareUrlDialog from './ShareUrlDialog';
import Actions from '../actions/actions';
import AddMemberForm from './AddMemberForm';
import Billing from './Billing';
import Audit from './Audit';
import CheckupAgentForm from './CheckupAgentForm';
import Dashboard from './Dashboard';
import ConsoleButton from './ConsoleButton';

import DbLabInstanceForm from './DbLabInstanceForm';
import DbLabInstances from './DbLabInstances';
import DbLabSessions from './DbLabSessions';
import DbLabSession from './DbLabSession';
import ExplainVisualization from './ExplainVisualization';
import JoeConfig from './JoeConfig';
import JoeInstanceForm from './JoeInstanceForm';
import JoeInstances from './JoeInstances';
import JoeHistory from './JoeHistory';
import Notification from './Notification';
import OrgForm from './OrgForm';
import OrgMembers from './OrgMembers';
import Permissions from '../utils/permissions';
import Report from './Report';
import ReportFile from './ReportFile';
import Reports from './Reports';
import SharedUrl from './SharedUrl';
import Store from '../stores/store';
import Urls from '../utils/urls';
import LoginDialog from './LoginDialog';
import { AppUpdateBanner } from './AppUpdateBanner';

import settings from '../utils/settings';

const drawerWidth = 180;

const styles = () => ({
  root: {
    flex: '1 1 0',
    zIndex: 1,
    overflow: 'hidden',
    position: 'relative',
    display: 'flex',
    fontSize: '14px'
  },
  appBar: {
    zIndex: theme.zIndex.drawer + 1,
    backgroundColor: colors.secondary2.darkDark
  },
  drawerPaper: {
    [theme.breakpoints.up('md')]: {
      paddingTop: '40px'
    },
    'position': 'absolute',
    'min-width': drawerWidth,
    'background-color': colors.consoleMenuBackground,
    'border-right-color': colors.consoleStroke,
    '& hr': {
      backgroundColor: colors.consoleStroke
    }
  },
  drawer: {
    minWidth: drawerWidth,
    flexShrink: 0
  },
  drawerContainer: {
    minWidth: drawerWidth
  },
  navIconHide: {
    [theme.breakpoints.down('sm')]: {
      display: 'inline-flex'
    },
    [theme.breakpoints.up('md')]: {
      display: 'none'
    },
    [theme.breakpoints.up('lg')]: {
      display: 'none'
    },
    'marginLeft': '-14px',
    '& svg': {
      marginTop: '-4px'
    }
  },
  rightDivider: {
    marginLeft: 30
  },
  navIconSignOut: {
    position: 'absolute',
    right: 0,
    padding: 8
  },
  navIconArea: {
    position: 'absolute',
    right: 35,
    color: 'white',
    textDecoration: 'none'
  },
  navIconProfile: {
    padding: 8
  },
  toolbar: theme.mixins.toolbar,
  topToolbar: {
    minHeight: 40,
    height: 40,
    paddingLeft: 14,
    paddingRight: 14
  },
  logo: {
    color: 'white',
    textDecoration: 'none',
    fontSize: 16
  },
  userName: {
    position: 'absolute',
    right: 77,
    fontSize: 14,
    [theme.breakpoints.down('sm')]: {
      display: 'none'
    },
    [theme.breakpoints.up('md')]: {
      display: 'block'
    },
    [theme.breakpoints.up('lg')]: {
      display: 'block'
    }
  },
  orgHeaderContainer: {
    position: 'relative',
    height: 40
  },
  orgHeader: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    position: 'absolute',
    left: '15px',
    top: '15px',
    fontStyle: 'normal',
    fontWeight: 'normal',
    fontSize: '10px',
    lineHeight: '12px',
    color: '#000000'
  },
  orgSwitcher: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    position: 'absolute',
    right: '10px',
    top: '10px',
    border: '1px solid #CCD7DA',
    borderRadius: '3px',
    fontStyle: 'normal',
    fontWeight: 'normal',
    fontSize: '10px',
    lineHeight: '12px',
    display: 'flex',
    alignItems: 'center',
    textAlign: 'center',
    color: '#808080',
    padding: 3,
    textTransform: 'capitalize',
    cursor: 'pointer'
  },
  orgNameContainer: {
    paddingLeft: '15px',
    height: '35px',
    position: 'relative'
  },
  orgName: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    fontStyle: 'normal',
    fontWeight: 'bold',
    fontSize: '14px',
    lineHeight: '16px',
    color: '#000000',
    maxWidth: '125px',
    whiteSpace: 'nowrap',
    textOverflow: 'ellipsis',
    overflow: 'hidden',
    display: 'inline-block'
  },
  orgPlan: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    fontStyle: 'normal',
    fontWeight: 'normal',
    fontSize: '10px',
    lineHeight: '11px',
    alignItems: 'center',
    textAlign: 'center',
    color: '#FFFFFF',
    backgroundColor: colors.secondary2.main,
    padding: '1px',
    borderRadius: '4px',
    paddingLeft: '3px',
    paddingRight: '3px',
    marginLeft: 10,
    position: 'absolute',
    top: '2px'
  },
  botTag: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    fontStyle: 'normal',
    fontWeight: 'normal',
    fontSize: '10px',
    lineHeight: '11px',
    alignItems: 'center',
    textAlign: 'center',
    color: '#FFFFFF',
    backgroundColor: colors.pgaiOrange,
    padding: '1px',
    borderRadius: '4px',
    paddingLeft: '3px',
    paddingRight: '3px',
    marginLeft: 10,
    position: 'absolute',
    top: '10px'
  },
  menuSectionHeader: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    fontStyle: 'normal',
    fontWeight: 'bold',
    fontSize: '14px',
    lineHeight: '16px',
    color: '#000000',
    padding: '0px',
    marginTop: '10px'
  },
  bottomFixedMenuItem: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    fontStyle: 'normal',
    fontWeight: 'bold',
    fontSize: '14px',
    lineHeight: '16px',
    color: '#000000',
    padding: '0px',
    marginTop: '0px'
  },
  menuSectionHeaderLink: {
    textDecoration: 'none',
    paddingTop: 12,
    paddingBottom: 12,
    paddingRight: 14,
    width: '100%',
    paddingLeft: '15px',
    color: '#000000'
  },
  menuSectionHeaderActiveLink: {
    textDecoration: 'none',
    paddingTop: 12,
    paddingBottom: 12,
    paddingRight: 14,
    width: '100%',
    paddingLeft: '15px',
    color: '#000000'
  },
  navMenu: {
    padding: '0px',
    marginBottom: '85px'
  },
  menuSectionHeaderIcon: {
    marginRight: '13px'
  },
  menuItem: {
    fontFamily: '"Roboto", "Helvetica", "Arial", sans-serif',
    fontStyle: 'normal',
    fontWeight: 'normal',
    fontSize: '14px',
    lineHeight: '16px',
    color: '#000000',
    padding: '0px',
    position: 'relative'
  },
  menuItemLink: {
    textDecoration: 'none',
    paddingTop: 8,
    paddingBottom: 8,
    paddingRight: 14,
    width: '100%',
    paddingLeft: '43px',
    color: '#000000',
    position: 'relative'
  },
  menuItemActiveLink: {
    textDecoration: 'none',
    paddingTop: 8,
    paddingBottom: 8,
    paddingRight: 14,
    backgroundColor: colors.consoleStroke,
    width: '100%',
    paddingLeft: '43px',
    color: '#000000',
    position: 'relative'
  },
  betaContainer: {
    '& > svg': {
      display: 'block',
      margin: 'auto'
    },
    'font-family': '"Roboto", "Helvetica", "Arial", sans-serif',
    'font-style': 'normal',
    'font-weight': 'normal',
    'font-size': '16px',
    'max-width': '500px',
    'background': '#ffffff',
    'border': '1px solid ' + colors.consoleStroke,
    'margin': 'auto',
    'border-radius': '3px',
    'padding': '40px'
  },
  betaWrapper: {
    'background': colors.consoleMenuBackground,
    'height': '100vh',
    'display': 'flex',
    'align-items': 'center'
  },
  tosContainer: {
    '& > svg': {
      display: 'block',
      margin: 'auto'
    },
    'font-family': '"Roboto", "Helvetica", "Arial", sans-serif',
    'font-style': 'normal',
    'font-weight': 'normal',
    'font-size': '14px',
    'width': '330px',
    'background': '#ffffff',
    'border': '1px solid ' + colors.consoleStroke,
    'margin': 'auto',
    'border-radius': '3px',
    'paddingTop': '35px',
    'paddingBottom': '35px',
    'textAlign': 'center',
    '& > p': {
      marginTop: '0px'
    }
  },
  tosWrapper: {
    'background': colors.consoleMenuBackground,
    'height': '100vh',
    'display': 'flex',
    'align-items': 'center',
    '& a': {
      textDecoration: 'underline'
    }
  },
  tosAgree: {
    marginTop: '15px',
    display: 'inline-block'
  },
  navBottomFixedMenu: {
    width: '100%',
    borderTop: '1px solid',
    borderColor: colors.consoleStroke,
    padding: 0,
    position: 'absolute',
    bottom: 0,
    backgroundColor: colors.consoleMenuBackground
  }
});

function ProjectWrapper(parentProps) {
  const project = parentProps.match.params.project;

  const errorMessage = (
    <div>
      404 Project <strong>{ project }</strong> not found<br/>
    </div>
  );

  if (!project || !parentProps.orgData.projects || !parentProps.orgData.projects[project]) {
    return errorMessage;
  }

  const projectId = parentProps.orgData.projects[project].id;

  const customProps = {
    project, projectId,
    org: parentProps.org,
    orgId: parentProps.orgId,
    orgData: parentProps.orgData,
    orgPermissions: parentProps.orgPermissions,
    userIsOwner: parentProps.userIsOwner,
    env: parentProps.env,
    envData: parentProps.envData,
    auth: parentProps.auth,
    raw: parentProps.raw
  };

  return (
    <Switch>
      <Route
        path='/:org/:project/instances/add'
        render={ (props) => <DbLabInstanceForm {...props} {...customProps} /> }
      />
      <Route exact path={ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.ADD.createPath()}>
        <CreateClone />
      </Route>
      <Route path={ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.CLONES.CLONE.createPath()}>
        <Clone />
      </Route>
      <Route path={ROUTES.ORG.PROJECT.INSTANCES.INSTANCE.createPath()}>
        <Instance />
      </Route>
      <Route
        path='/:org/:project/instances'
        render={ (props) => <DbLabInstances {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/observed-sessions/:sessionId'
        render={ (props) => <DbLabSession {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/observed-sessions'
        render={ (props) => <DbLabSessions {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/joe-instances/add'
        render={ (props) => <JoeInstanceForm {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/joe-instances/:instanceId'
        render={ (props) => <JoeInstance {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/joe-instances'
        render={ (props) => <JoeInstances {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/reports/:reportId/files/:fileId/:fileType'
        render={ (props) => <ReportFile {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/reports/:reportId/:type'
        render={ (props) => <Report {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/reports/:reportId/'
        render={ (props) => <Report {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/reports'
        render={ (props) => <Reports {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/sessions/:sessionId/commands/:commandId'
        render={ (props) => <JoeSessionCommand {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/sessions/:sessionId'
        render={ (props) => <JoeHistory {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project/sessions'
        render={ (props) => <JoeHistory {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project'
        render={ (props) => <Reports {...props} {...customProps} /> }
      />
    </Switch>
  );
}

function OrganizationMenu(parentProps) {
  if (parentProps.env && parentProps.env.data &&
    parentProps.match.params.org && parentProps.env.data.orgs &&
    parentProps.env.data.orgs[parentProps.match.params.org]) {
    let org = parentProps.match.params.org;
    let isBlocked = false;
    let orgData = null;

    let orgPermissions = null;
    if (parentProps.env.data && parentProps.env.data.orgs && parentProps.env.data.orgs[org]) {
      orgData = parentProps.env.data.orgs[org];
      isBlocked = orgData.is_blocked;
      orgPermissions = Permissions.getPermissions(orgData);
    }

    return (
      <div>
        <div>
          <div className={parentProps.classes.orgHeaderContainer}>
            <span className={parentProps.classes.orgHeader}>Organization</span>
            <NavLink to={ROUTES.ROOT.path}>
              <span className={parentProps.classes.orgSwitcher} id='menuSwitch'>switch</span>
            </NavLink>
          </div>
          <div className={parentProps.classes.orgNameContainer}>
            <span className={parentProps.classes.orgName}>
              {parentProps.env.data.orgs[org].name}
            </span>
          </div>
        </div>

        <Divider />

        <List component='nav' className={parentProps.classes.navMenu}>
          <ListItem
            button
            className={parentProps.classes.menuSectionHeader}
            disabled={isBlocked}
            id='menuDashboard'
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
            id='menuDblabTitle'
          >
            <NavLink
              className={parentProps.classes.menuSectionHeaderLink}
              activeClassName={parentProps.classes.menuSectionHeaderActiveLink}
              to={'/' + org + '/instances'}
            >
              <span className={parentProps.classes.menuSectionHeaderIcon}>
                {icons.databaseLabIcon}
              </span>
              Database Lab
            </NavLink>
          </ListItem>
          <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuDblabInstances'
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
            id='menuDblabSessions'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/observed-sessions'}
            >
              Observed sessions
            </NavLink>
          </ListItem>
          <ListItem
            button
            className={parentProps.classes.menuSectionHeader}
            disabled={isBlocked}
            id='menuJoeTitle'
          >
            <NavLink
              className={parentProps.classes.menuSectionHeaderLink}
              activeClassName={parentProps.classes.menuSectionHeaderActiveLink}
              to={'/' + org + '/joe-instances'}
            >
              <span className={parentProps.classes.menuSectionHeaderIcon}>
                {icons.sqlOptimizationIcon}
              </span>
              SQL Optimization
            </NavLink>
          </ListItem>
          <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuJoeInstances'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/joe-instances'}
            >
              Ask Joe<span className={parentProps.classes.botTag}>BOT</span>
            </NavLink>
          </ListItem>
          <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuJoeSessions'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/sessions?'}
            >
              History
            </NavLink>
          </ListItem>
          {false && <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuJoeExplain'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/explain'}
            >
              Plan visualization
            </NavLink>
          </ListItem>}

          <ListItem
            button
            className={parentProps.classes.menuSectionHeader}
            disabled={isBlocked}
            id='menuCheckupTitle'
          >
            <NavLink
              className={parentProps.classes.menuSectionHeaderLink}
              activeClassName={parentProps.classes.menuSectionHeaderActiveLink}
              to={'/' + org + '/reports'}
            >
              <span className={parentProps.classes.menuSectionHeaderIcon}>
                {icons.checkupIcon}
              </span>
              Checkup
            </NavLink>
          </ListItem>
          <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuCheckupReports'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/reports'}
            >
              Reports
            </NavLink>
          </ListItem>

          <ListItem
            button
            className={parentProps.classes.menuSectionHeader}
            disabled={isBlocked}
            id='menuSettingsTitle'
          >
            <NavLink
              className={parentProps.classes.menuSectionHeaderLink}
              activeClassName={parentProps.classes.menuSectionHeaderActiveLink}
              to={orgPermissions && orgPermissions.settingsOrganizationUpdate ?
                '/' + org + '/settings' : '#'}
            >
              <span className={parentProps.classes.menuSectionHeaderIcon}>
                {icons.settingsIcon}
              </span>
              Settings
            </NavLink>
          </ListItem>
          { orgPermissions && orgPermissions.settingsOrganizationUpdate && <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuSettingsGeneral'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/settings'}
            >
              General
            </NavLink>
          </ListItem>}
          <ListItem
            disabled={ isBlocked }
            button
            className={ parentProps.classes.menuItem }
            id='menuSettingsMembers'
          >
            <NavLink
              disabled
              className={ parentProps.classes.menuItemLink }
              activeClassName={ parentProps.classes.menuItemActiveLink }
              to={ '/' + org + '/members' }
            >
              Members
            </NavLink>
          </ListItem>
          <ListItem
            button
            className={parentProps.classes.menuItem}
            disabled={isBlocked}
            id='menuSettingsTokens'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/tokens'}
            >
              Access tokens
            </NavLink>
          </ListItem>
          { Permissions.isAdmin(orgData) && <ListItem
            button
            className={parentProps.classes.menuItem}
            id='menuSettingsBilling'
          >
            <NavLink
              className={parentProps.classes.menuItemLink}
              activeClassName={parentProps.classes.menuItemActiveLink}
              to={'/' + org + '/billing'}
            >
              Billing
            </NavLink>
          </ListItem>}
          { orgPermissions && orgPermissions.auditLogView && <ListItem
            disabled={ (orgPermissions && !orgPermissions.auditLogView) || isBlocked }
            button
            className={ parentProps.classes.menuItem }
            id='menuSettingsAudit'
          >
            <NavLink
              disabled
              className={ parentProps.classes.menuItemLink }
              activeClassName={ parentProps.classes.menuItemActiveLink }
              to={ '/' + org + '/audit' }
            >
              Audit
            </NavLink>
          </ListItem>}
        </List>
      </div>
    );
  }

  return null;
}

function OrganizationWrapper(parentProps) {
  const org = parentProps.match.params.org;
  const envData = parentProps.env && parentProps.env.data ?
    parentProps.env.data : { orgs: null };

  const errorMessage = (
    <div>
      404 Organization <strong>{ org }</strong> not found<br/>
    </div>
  );

  const preloader = (
    <PageSpinner />
  );

  if (org === 'auth' || !envData.orgs) {
    return preloader;
  }

  if (!org || (org && envData.orgs && !envData.orgs[org])) {
    return errorMessage;
  }

  const env = parentProps.env;
  const auth = parentProps.auth;
  const raw = parentProps.raw;
  const orgId = envData.orgs[org].id;
  const orgData = envData.orgs[org];
  const isBlocked = orgData.is_blocked;
  const orgPermissions = Permissions.getPermissions(orgData);
  const userIsOwner = (!env || !auth || !orgId) ? false :
    Permissions.userIsOwner(auth.userId, orgId, env.data);

  const queryParams = qs.parse(parentProps.location.search, { ignoreQueryPrefix: true });
  const {
    session, command, author, fingerprint, project, search, is_favorite
  } = queryParams;

  const customProps = {
    org, orgId, orgData, orgPermissions, userIsOwner, env, envData, auth,
    session, command, author, fingerprint, project, search, is_favorite, raw
  };

  if (isBlocked && Permissions.isAdmin(orgData)) {
    return (
      <Switch>
        <Route
          path='/:org/billing'
          render={ (props) => <Billing {...props} {...customProps} /> }
        />
        <Route
          path={ROUTES.ROOT.path}
          render={ (props) => <Billing {...props} {...customProps} short={true}/> }
        />
      </Switch>
    );
  }

  return (
    <Switch>
      <Route
        path='/:org/instances/add'
        render={ (props) => <DbLabInstanceForm {...props} {...customProps} /> }
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
        path='/:org/instances'
        render={ (props) => <DbLabInstances {...props} {...customProps} /> }
      />
      <Route
        path='/:org/observed-sessions/:sessionId'
        render={ (props) => <DbLabSession {...props} {...customProps} /> }
      />
      <Route
        path='/:org/observed-sessions'
        render={ (props) => <DbLabSessions {...props} {...customProps} /> }
      />
      <Route
        path='/:org/joe-instances/add'
        render={ (props) => <JoeInstanceForm {...props} {...customProps} /> }
      />
      <Route
        path='/:org/joe-instances/:instanceId'
        render={ (props) => <JoeInstance {...props} {...customProps} /> }
      />
      <Route
        path='/:org/joe-instances'
        render={ (props) => <JoeInstances {...props} {...customProps} /> }
      />
      <Route
        path='/:org/members/add'
        render={ (props) => <AddMemberForm {...props} {...customProps} /> }
      />
      <Route
        path='/:org/settings'
        render={ (props) => <OrgForm {...props} {...customProps} /> }
      />
      <Route
        path='/:org/billing'
        render={ (props) => <Billing {...props} {...customProps} /> }
      />
      <Route
        path='/:org/audit'
        render={ (props) => <Audit {...props} {...customProps} /> }
      />
      <Route
        path='/:org/members'
        render={ (props) => <OrgMembers {...props} {...customProps} /> }
      />
      <Route
        path='/:org/reports'
        render={ (props) => <Reports {...props} {...customProps} /> }
      />
      <Route
        path={ROUTES.ORG.TOKENS.createPath()}
        render={ (props) => <AccessTokens {...props} {...customProps} /> }
      />
      <Route
        path='/:org/checkup-config'
        render={ (props) => <CheckupAgentForm {...props} {...customProps} /> }
      />
      <Route
        path='/:org/joe-config'
        render={ (props) => <JoeConfig {...props} {...customProps} /> }
      />
      <Route
        path='/:org/explain'
        render={ (props) => <ExplainVisualization {...props} {...customProps} /> }
      />
      <Route
        path='/:org/sessions'
        render={ (props) => <JoeHistory {...props} {...customProps} /> }
      />
      <Route
        path='/:org/:project'
        render={ (props) => <ProjectWrapper {...props} {...customProps} /> }
      />
      <Route
        path='/:org'
        render={ (props) => <Dashboard onlyProjects {...props} {...customProps} /> }
      />
    </Switch>
  );
}

function SupportMenu(props) {
  return (
    <List component='nav' className={props.classes.navBottomFixedMenu}>
      <ListItem button className={props.classes.bottomFixedMenuItem}>
        <a
          className={props.classes.menuSectionHeaderLink}
          activeClassName={props.classes.menuSectionHeaderLink}
          target='_blank'
          href={settings.rootUrl + '/docs'}
        >
          <span className={props.classes.menuSectionHeaderIcon}>
            {icons.docIcon}
          </span>
          Documentation
        </a>
      </ListItem>
      <ListItem button className={props.classes.bottomFixedMenuItem}>
        <span
          className={props.classes.menuSectionHeaderLink}
          activeClassName={props.classes.menuSectionHeaderActiveLink}
          onClick={() => window.Intercom && window.Intercom('show')}
        >
          <span className={props.classes.menuSectionHeaderIcon}>
            {icons.supportIcon}
          </span>
          Ask support
        </span>
      </ListItem>
    </List>
  );
}

class IndexPage extends Component {
  state = {
    mobileOpen: false
  };

  componentDidMount() {
    const that = this;

    document.getElementsByTagName('html')[0].style.overflow = 'hidden';

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });

      // redirect to new organization alias if need
      const env = this.data ? this.data.userProfile : null;
      const orgProfile = this.data && this.data.orgProfile ? this.data.orgProfile :
        null;
      if (orgProfile && orgProfile.prevAlias && orgProfile.data.alias &&
        env && env.data && env.data.orgs[orgProfile.data.alias]) {
        that.props.history.push('/' + orgProfile.data.alias + '/settings');
      }

      if ((env.isConfirmProcessed || (env.data && env.data.info.is_active)) &&
        Urls.isRequestedPath('confirm')) {
        that.props.history.push(ROUTES.ROOT.path);
      }
    });

    Actions.doAuth(null, null);
  }

  signOut() {
    Actions.signOut();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleDrawerToggle = () => {
    this.setState({ mobileOpen: !this.state.mobileOpen });
  }

  goHrefUrl = event => {
    this.props.history.push(event.target.getAttribute('hrefurl'));
    return false;
  }

  resendCode = () => {
    const auth = this.state.data && this.state.data.auth ? this.state.data.auth :
      null;

    if (auth.token) {
      Actions.sendUserCode(auth.token);
    }
  }

  confirmTosAgreement = () => {
    const auth = this.state.data && this.state.data.auth ? this.state.data.auth :
      null;

    if (auth.token) {
      Actions.confirmTosAgreement(auth.token);
    }
  }

  addOrgButtonHandler = () => {
    this.props.history.push(ROUTES.CREATE_ORG.path);
  }

  render() {
    const { classes } = this.props;
    const auth = this.state.data && this.state.data.auth ? this.state.data.auth :
      null;
    const env = this.state.data ? this.state.data.userProfile : null;
    const raw = this.props && this.props.location && this.props.location.search &&
      this.props.location.search.indexOf('raw') !== -1;

    if (auth && auth.isProcessed && !auth.token && !Urls.isSharedUrl()) {
      if (window.location.pathname === '/login') {
        return (
          <div>
            <LoginDialog/>
          </div>
        );
      }

      if (window.location.pathname !== '/signin') {
        this.signOut();
        return null;
      }

      return <SignIn />;
    }

    const appBarLogo = (
      <Typography color='inherit' noWrap>
        <NavLink
          to={ROUTES.ROOT.path}
          className={classes.logo}
        >
          Database Lab Platform &#946;
        </NavLink>
      </Typography>
    );

    const appBarSignOut = (
      <IconButton
        color='inherit'
        aria-label='Sign out'
        onClick={this.signOut}
        className={classes.navIconSignOut}
      >
        {icons.exitIcon}
      </IconButton>
    );

    const uiUpdate = <AppUpdateBanner />;

    let userName = '';
    if (env && env.data && env.data.info) {
      if (env.data.info.first_name) {
        userName = env.data.info.first_name;
      } else {
        userName = env.data.info.email;
      }
    }

    if (Urls.isSharedUrl()) {
      return (
        <ThemeProvider theme={theme}>
          <div className={classes.root}>
            <AppBar position='absolute' className={classes.appBar}>
              <Toolbar className={classes.topToolbar}>
                {appBarLogo}
                {auth && auth.token && <Typography
                  color='inherit'
                  noWrap
                  className={classes.userName}
                >
                  {userName}
                </Typography>}
                {auth && auth.token ? (<NavLink
                  to={ROUTES.PROFILE.path}
                  className={classes.navIconArea}
                >
                  <IconButton
                    color='inherit'
                    aria-label='Profile'
                    className={classes.navIconProfile}
                  >
                    {icons.userIcon}
                  </IconButton>
                </NavLink>) : null}
                {auth && auth.token && appBarSignOut}
              </Toolbar>
            </AppBar>
            <ContentLayout>
              <Switch>
                <Route
                  path='/shared/:url_uuid'
                  render={(props) => (
                    <SharedUrl
                      {...props}
                    />
                  )}
                />
                <Redirect from='*' to={ROUTES.ROOT.path} />
              </Switch>
            </ContentLayout>
          </div>
        </ThemeProvider>
      );
    }

    if (!env || (env && !env.data)) {
      return (
        <ThemeProvider theme={theme}>
          <PageSpinner />
        </ThemeProvider>
      );
    }

    if (!env.data.info['is_tos_confirmed']) {
      return (
        <ThemeProvider theme={theme}>
          { uiUpdate }

          <div className={classes.tosWrapper}>
            <div className={classes.tosContainer}>
              <p>
                Please, read and agree to our updated<br/>
                <Link href='https://postgres.ai/tos' target='_blank'>Terms of Service</Link>
                &nbsp;and&nbsp;
                <Link href='https://postgres.ai/privacy' target='_blank'>Privacy Policy</Link>
              </p>

              <ConsoleButton
                variant='contained'
                color='primary'
                key='add_dblab_instance'
                disabled={env.isTosAgreementConfirmProcessing}
                onClick={ this.confirmTosAgreement }
                className={classes.tosAgree}
                title='I Agree'
              >
                I Agree
              </ConsoleButton>
            </div>
            <Notification/>
          </div>
        </ThemeProvider>
      );
    }

    if (!env.data.info['is_active']) {
      if (Urls.isRequestedPath('confirm') && Urls.getRequestParam('code')) {
        if (!env.isConfirmProcessing && !env.isConfirmProcessed) {
          Actions.confirmUserEmail(auth.token, Urls.getRequestParam('code'));
        }
        if (!env.isConfirmProcessed) {
          return (
            <ThemeProvider theme={theme}>
              { uiUpdate }
              <PageSpinner />
            </ThemeProvider>
          );
        }

        this.props.history.push(ROUTES.ROOT.path);
      }

      return (
        <ThemeProvider theme={theme}>
          { uiUpdate }
          <AppBar position='absolute' className={classes.appBar}>
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

              <strong>Hi {env.data.info['first_name'] ?
                env.data.info['first_name'] : env.data.info['user_name']}!
              </strong>

              <p>
                Please confirm registration by clicking
                on a link we have just sent to {env.data.info['email']}.
              </p>

              <p>
                If you haven't received the email,&nbsp;
                <Link onClick={this.resendCode} href='#'>click here</Link> to resend it.
              </p>


            </div>
            <Notification/>
          </div>
        </ThemeProvider>
      );
    }

    if (raw) {
      return (
        <ThemeProvider theme={theme}>
          <style>{`
              div.intercom-lightweight-app {
                display: none!important;
              }
            `}
          </style>
          <Switch>
            <Route
              path='/:org'
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
            <Redirect from='*' to={ROUTES.ROOT.path} />
          </Switch>
        </ThemeProvider>
      );
    }

    const drawer = (
      <div onClick={this.handleDrawerToggle} className='menu-pointer'>
        <Divider />
        <Switch>
          <Route exact path={[ROUTES.ROOT.path, ROUTES.PROFILE.path]}>
            <SideNav />
          </Route>

          <Route
            path='/:org'
            render={(props) => (
              <OrganizationMenu
                {...props}
                classes={classes}
                env={env}
              />
            )}
          />
        </Switch>
        <SupportMenu {...this.props} />
      </div>
    );

    return (
      <ThemeProvider theme={theme}>
        { uiUpdate }
        <div className={classes.root}>
          <AppBar position='absolute' className={classes.appBar}>
            <Toolbar className={classes.topToolbar}>
              <IconButton
                color='inherit'
                aria-label='Open menu'
                onClick={this.handleDrawerToggle}
                className={`menu-pointer ${classes.navIconHide}`}
              >
                {icons.menuIcon}
              </IconButton>
              {appBarLogo}
              <Typography color='inherit' noWrap className={classes.userName}>
                {userName}
              </Typography>
              <NavLink
                to={ROUTES.PROFILE.path}
                className={classes.navIconArea}
              >
                <IconButton
                  color='inherit'
                  aria-label='Profile'
                  className={classes.navIconProfile}
                >
                  {icons.userIcon}
                </IconButton>
              </NavLink>
              {appBarSignOut}
            </Toolbar>
          </AppBar>

          <Hidden mdUp>
            <Drawer
              className={classes.drawer}
              variant='temporary'
              anchor='left'
              open={this.state.mobileOpen}
              onClose={this.handleDrawerToggle}
              classes={{
                paper: classes.drawerPaper
              }}
              ModalProps={{
                // Better open performance on mobile.
                keepMounted: true
              }}
            >
              {drawer}
            </Drawer>
          </Hidden>
          <Hidden smDown implementation='css' className={classes.drawerContainer}>
            <Drawer
              variant='permanent'
              open
              classes={{
                paper: classes.drawerPaper
              }}
            >
              {drawer}
            </Drawer>
          </Hidden>
          <ContentLayout>
            <ShareUrlDialog/>
            <Switch>
              <Redirect from='/signin' to={ROUTES.ROOT.path} />
              <Redirect from='/login' to={ROUTES.ROOT.path} />
              <Route path={ROUTES.PROFILE.path} component={Profile} />
              <Route path={ROUTES.ROOT.path}
                exact
                render={(props) => (
                  <Dashboard
                    onlyProjects={false}
                    {...props}
                    env={env}
                  />
                )}
              />
              <Route
                path={ROUTES.CREATE_ORG.path}
                render={
                  (props) => <OrgForm {...props} mode='new' />
                }
              />
              <Route
                path='/:org'
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
            <Notification/>
          </ContentLayout>
        </div>
      </ThemeProvider>
    );
  }
}

IndexPage.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(IndexPage);
