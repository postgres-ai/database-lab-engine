/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import {
  Table, TableBody, TableCell, TableHead,
  TableRow, TextField, Menu, MenuItem, Tooltip, IconButton
} from '@material-ui/core';
import WarningIcon from '@material-ui/icons/Warning';
import MoreVertIcon from '@material-ui/icons/MoreVert';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { StubContainer } from '@postgres.ai/shared/components/StubContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import ProductCard from 'components/ProductCard';

import Actions from '../actions/actions';
import Aliases from '../utils/aliases';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsoleButton from './ConsoleButton';
import ConsolePageTitle from './ConsolePageTitle';
import Error from './Error';
import Link from './Link';
import messages from '../assets/messages';
import Store from '../stores/store';
import Urls from '../utils/urls';
import Utils from '../utils/utils';


const getStyles = () => ({
  root: {
    ...styles.root,
    paddingBottom: '20px',
    display: 'flex',
    flexDirection: 'column'
  },
  filterSelect: {
    ...styles.inputField,
    width: 150
  },
  cell: {
    '& > a': {
      color: 'black',
      textDecoration: 'none'
    },
    '& > a:hover': {
      color: 'black',
      textDecoration: 'none'
    }
  },
  warningIcon: {
    color: colors.state.warning,
    fontSize: '1.2em',
    position: 'absolute',
    marginLeft: 5
  },
  toolTip: {
    fontSize: '10px!important'
  }
});

class JoeInstances extends Component {
  componentDidMount() {
    const that = this;
    let orgId = this.props.orgId ? this.props.orgId : null;
    let projectId = this.props.projectId ? this.props.projectId : null;

    if (!projectId) {
      projectId = this.props.match && this.props.match.params &&
        this.props.match.params.projectId ?
        this.props.match.params.projectId : null;
    }

    if (projectId) {
      Actions.setJoeInstancesProject(orgId, projectId);
    } else {
      Actions.setJoeInstancesProject(orgId, 0);
    }

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const joeInstances = this.data && this.data.joeInstances ? this
        .data.joeInstances : null;
      const projects = this.data && this.data.projects ? this.data.projects :
        null;

      if (auth && auth.token && !joeInstances.isProcessing &&
        !joeInstances.error && !that.state) {
        Actions.getJoeInstances(auth.token, orgId, projectId);
      }

      if (auth && auth.token && !projects.isProcessing && !projects.error &&
        !that.state) {
        Actions.getProjects(auth.token, orgId);
      }

      that.setState({ data: this.data });
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleClick = (event, id) => {
    const url = Urls.linkJoeInstance(this.props, id);

    if (url) {
      this.props.history.push(url);
    }
  };

  handleChangeProject = (event) => {
    const org = this.props.org ? this.props.org : null;
    const orgId = this.props.orgId ? this.props.orgId : null;
    const projectId = event.target.value;
    const project = Aliases.getProjectAliasById(this.state.data.userProfile.data
      .orgs, projectId);
    const props = { org, orgId, projectId, project };

    Actions.setJoeInstancesProject(orgId, event.target.value);
    this.props.history.push(Urls.linkJoeInstances(props));
  };

  registerButtonHandler = () => {
    this.props.history.push(Urls.linkJoeInstanceAdd(this.props));
  };

  openMenu = (event) => {
    event.stopPropagation();
    this.setState({ anchorEl: event.currentTarget });
  };

  closeMenu = () => {
    this.setState({ anchorEl: null });
  };

  menuHandler = (event, action) => {
    const anchorEl = this.state.anchorEl;

    this.closeMenu();

    setTimeout(() => {
      const auth = this.state.data && this.state.data.auth ? this.state.data
        .auth : null;

      if (anchorEl) {
        let instanceId = anchorEl.getAttribute('instanceid');
        if (!instanceId) {
          return;
        }

        switch (action) {
        case 'destroy':
          /* eslint no-alert: 0 */
          if (window.confirm('Are you sure you want to remove this' +
              ' Joe Bot instance?') === true) {
            Actions.destroyJoeInstance(auth.token, instanceId);
          }

          break;
        default:
          break;
        }
      }
    }, 100);
  };

  render() {
    const { classes, orgPermissions, orgId } = this.props;
    const data = this.state && this.state.data &&
      this.state.data.joeInstances ? this.state .data.joeInstances : null;
    const projects = this.state && this.state.data &&
      this.state.data.projects ? this.state.data.projects : null;
    let projectId = this.props.projectId ? this.props.projectId : null;
    const menuOpen = Boolean(this.state && this.state.anchorEl);
    const title = 'Joe Bot instances';

    if (!projectId) {
      projectId = this.props.match && this.props.match.params && this.props.match
        .params.projectId ?
        this.props.match.params.projectId : null;
    }

    const createPermitted = !orgPermissions || orgPermissions.joeInstanceCreate;
    const deletePermitted = !orgPermissions || orgPermissions.joeInstanceDelete;

    let projectFilter = null;
    if (projects && projects.data && data) {
      projectFilter = (
        <div>
          <TextField
            value={data.projectId}
            onChange={event => this.handleChangeProject(event)}
            select
            label='Project'
            inputProps={{
              name: 'project',
              id: 'project-filter'
            }}
            InputLabelProps={{
              shrink: true,
              style: styles.inputFieldLabel
            }}
            FormHelperTextProps={{
              style: styles.inputFieldHelper
            }}
            variant='outlined'
            className={classes.filterSelect}
          >
            <MenuItem value={0}>All</MenuItem>

            {projects.data.map(p => {
              return (
                <MenuItem value={p.id} key={p.id} >{p.name}</MenuItem>
              );
            })}
          </TextField>
        </div>
      );
    }

    let breadcrumbs = (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[
          { name: 'SQL Optimization', url: null },
          { name: title }
        ]}
      />
    );

    if (this.state && this.state.data.joeInstances.error) {
      return (
        <div>
          {breadcrumbs}

          <ConsolePageTitle title={title}/>

          <Error/>
        </div>
      );
    }

    if (!data || (data && data.isProcessing) ||
      (data.orgId !== orgId) || (data.projectId !== (projectId ? projectId : 0))) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          <ConsolePageTitle title={title}/>

          <PageSpinner />
        </div>
      );
    }

    const addInstanceButton = (
      <ConsoleButton
        disabled={ !createPermitted }
        variant='contained'
        color='primary'
        key='add_instance'
        onClick={this.registerButtonHandler}
        title={ createPermitted ? 'Add a new Joe Bot instance' : messages.noPermission }
      >
        Add instance
      </ConsoleButton>
    );

    const emptyListTitle = projectId ?
      'There are no Joe Bot instances in this project yet' : 'There are no Joe Bot instances yet';

    let table = (
      <StubContainer>
        <ProductCard
          inline
          title={ emptyListTitle }
          actions={[{
            id: 'addInstanceButton',
            content: addInstanceButton
          }]}
          icon= { icons.joeLogo }
        >
          <p>
            Joe Bot is a virtual DBA for SQL Optimization. Joe helps engineers
            quickly troubleshoot and optimize SQL. Joe runs on top of the
            Database Lab Engine.
            (
            <Link
              link='https://postgres.ai/docs/joe-bot'
              target='_blank'
            >
              Learn more
            </Link>
            ).
          </p>
        </ProductCard>
      </StubContainer>
    );

    if (data.data && Object.keys(data.data).length > 0) {
      table = (
        <HorizontalScrollContainer>
          <Table className={classes.table} id='joeInstancesTable'>
            <TableHead>
              <TableRow className={classes.row}>
                <TableCell>Project</TableCell>
                <TableCell>URL</TableCell>
                <TableCell>&nbsp;</TableCell>
              </TableRow>
            </TableHead>

            <TableBody>
              {Object.keys(data.data).map(i => {
                return (
                  <TableRow
                    hover
                    className={classes.row}
                    key={i.id}
                    onClick={event => this.handleClick(event,
                      data.data[i].id, data.data[i].project_id)}
                    style={{ cursor: 'pointer' }}
                    key={i}
                  >
                    <TableCell className={classes.cell}>
                      {data.data[i].project_name}
                    </TableCell>

                    <TableCell className={classes.cell}>
                      {data.data[i].url ? data.data[i].url : ''}
                      {!Utils.isHttps(data.data[i].url) && !data.data[i].use_tunnel ? (
                        <Tooltip
                          title='The connection to Joe Bot API is not secure'
                          aria-label='add'
                          classes={{ tooltip: classes.toolTip }}
                        >
                          <WarningIcon className={classes.warningIcon}/>
                        </Tooltip>) : null}
                    </TableCell>

                    <TableCell align={'right'}>
                      <IconButton
                        aria-label='more'
                        aria-controls='instance-menu'
                        aria-haspopup='true'
                        onClick={this.openMenu}
                        instanceid={data.data[i].id}
                      >
                        <MoreVertIcon />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </HorizontalScrollContainer>
      );
    }

    const menu = (
      <Menu
        id='instance-menu'
        anchorEl={this.state.anchorEl}
        keepMounted
        open={menuOpen}
        onClose={this.closeMenu}
        PaperProps={{
          style: {
            width: 200
          }
        }}
      >
        <MenuItem
          disabled={ !deletePermitted }
          key={1}
          onClick={event => this.menuHandler(event, 'destroy')}
        >
          Remove
        </MenuItem>
      </Menu>
    );

    return (
      <div className={classes.root}>
        {breadcrumbs}

        <ConsolePageTitle
          title={title}
          actions={[ addInstanceButton ]}
        />

        {projectFilter}

        {table}

        {menu}
      </div>
    );
  }
}

JoeInstances.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(JoeInstances);
