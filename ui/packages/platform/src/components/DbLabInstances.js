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
  TableRow, TextField, IconButton, Menu, MenuItem, Tooltip
} from '@material-ui/core';
import MoreVertIcon from '@material-ui/icons/MoreVert';
import WarningIcon from '@material-ui/icons/Warning';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { StubContainer } from '@postgres.ai/shared/components/StubContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import Actions from '../actions/actions';
import Aliases from '../utils/aliases';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsoleButton from './ConsoleButton';
import ConsolePageTitle from './ConsolePageTitle';
import DbLabStatus from './DbLabStatus';
import Error from './Error';
import Link from './Link';
import messages from '../assets/messages';
import ProductCard from './ProductCard';
import Store from '../stores/store';
import Urls from '../utils/urls';
import Utils from '../utils/utils';
import Warning from './Warning';


const getStyles = () => ({
  root: {
    ...styles.root,
    display: 'flex',
    flexDirection: 'column'
  },
  stubContainer: {
    marginTop: '10px'
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
  inTableProgress: {
    width: '30px!important',
    height: '30px!important'
  },
  warningIcon: {
    color: colors.state.warning,
    fontSize: '1.2em',
    position: 'absolute',
    marginLeft: 5
  },
  tooltip: {
    fontSize: '10px!important'
  }
});

class DbLabInstances extends Component {
  componentDidMount() {
    const that = this;
    let orgId = this.props.orgId ? this.props.orgId : null;
    let projectId = this.props.projectId ? this.props.projectId : null;

    if (!projectId) {
      projectId = this.props.match && this.props.match.params && this.props.match
        .params.projectId ?
        this.props.match.params.projectId : null;
    }

    if (projectId) {
      Actions.setDbLabInstancesProject(orgId, projectId);
    } else {
      Actions.setDbLabInstancesProject(orgId, 0);
    }

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const dbLabInstances = this.data && this.data.dbLabInstances ? this
        .data.dbLabInstances : null;
      const projects = this.data && this.data.projects ? this.data.projects :
        null;

      if (auth && auth.token && !dbLabInstances.isProcessing &&
        !dbLabInstances.error && !that.state) {
        Actions.getDbLabInstances(auth.token, orgId, projectId);
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
    const url = Urls.linkDbLabInstance(this.props, id);

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

    Actions.setDbLabInstancesProject(orgId, event.target.value);
    this.props.history.push(Urls.linkDbLabInstances(props));
  };

  registerButtonHandler = () => {
    this.props.history.push(Urls.linkDbLabInstanceAdd(this.props));
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
      const data = this.state.data && this.state.data.dbLabInstances ?
        this.state.data.dbLabInstances : null;

      if (anchorEl) {
        let instanceId = anchorEl.getAttribute('instanceid');
        if (!instanceId) {
          return;
        }

        let project = '';
        if (data.data) {
          for (let i in data.data) {
            if (parseInt(data.data[i].id, 10) === parseInt(instanceId, 10)) {
              project = data.data[i].project_alias;
            }
          }
        }

        switch (action) {
        case 'addclone':
          let props = { org: this.props.org, project: project };
          this.props.history.push(Urls.linkDbLabCloneAdd(props,
            instanceId));

          break;

        case 'destroy':
          /* eslint no-alert: 0 */
          if (window.confirm('Are you sure you want to remove this' +
              ' Database Lab instance?') === true) {
            Actions.destroyDbLabInstance(auth.token, instanceId);
          }

          break;

        case 'refresh':
          Actions.getDbLabInstanceStatus(auth.token, instanceId);

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
     this.state.data.dbLabInstances ? this.state .data.dbLabInstances : null;
    const projects = this.state && this.state.data &&
      this.state.data.projects ? this.state.data.projects : null;
    let projectId = this.props.projectId ? this.props.projectId : null;
    const menuOpen = Boolean(this.state && this.state.anchorEl);
    const title = 'Database Lab Instances';
    const addPermitted = !orgPermissions || orgPermissions.dblabInstanceCreate;
    const deletePermitted = !orgPermissions || orgPermissions.dblabInstanceDelete;
    const addInstanceButton = (
      <ConsoleButton
        disabled={ !addPermitted }
        variant='contained'
        color='primary'
        key='add_dblab_instance'
        onClick={ this.registerButtonHandler }
        title={ addPermitted ? 'Add a new Database Lab instance' : messages.noPermission }
      >
        Add instance
      </ConsoleButton>
    );
    const pageTitle = (<ConsolePageTitle
      title={title}
      actions={[ addInstanceButton ]}
    />);

    if (!projectId) {
      projectId = this.props.match && this.props.match.params && this.props.match
        .params.projectId ?
        this.props.match.params.projectId : null;
    }

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
          { name: title }
        ]}
      />
    );

    if (orgPermissions && !orgPermissions.dblabInstanceList) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <Warning>{ messages.noPermissionPage }</Warning>
        </div>
      );
    }

    if (this.state && this.state.data.dbLabInstances.error) {
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

    const emptyListTitle = projectId ?
      'There are no Database Lab instances in this project yet' :
      'There are no Database Lab instances yet';

    let table = (
      <StubContainer className={classes.stubContainer}>
        <ProductCard
          inline
          title={ emptyListTitle }
          actions={[{
            id: 'addInstanceButton',
            content: addInstanceButton
          }]}
          icon= { icons.databaseLabLogo }
        >
          <p>
            Clone multi-terabyte databases in seconds and use them to
            test your database migrations, optimize SQL, or deploy full-size
            staging apps. Start here to work with all Database Lab tools.
            Setup
            <Link
              link='https://postgres.ai/docs/database-lab'
              target='_blank'
            >
              documentation here
            </Link>
            .
          </p>
        </ProductCard>
      </StubContainer>
    );

    let menu = null;
    if (data.data && Object.keys(data.data).length > 0) {
      table = (
        <HorizontalScrollContainer>
          <Table className={classes.table} id='dblabInstancesTable'>
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
              {Object.keys(data.data).map(i => {
                return (
                  <TableRow
                    hover
                    className={classes.row}
                    key={i.id}
                    onClick={event => this.handleClick(event,
                      data.data[i].id, data.data[i].project_id)}
                    style={{ cursor: 'pointer' }}
                  >
                    <TableCell className={classes.cell}>
                      {data.data[i].project_name}
                    </TableCell>

                    <TableCell className={classes.cell}>
                      {data.data[i].state && data.data[i].url ? data.data[i].url : ''}
                      {!Utils.isHttps(data.data[i].url) && !data.data[i].use_tunnel ? (
                        <Tooltip
                          title='The connection to Database Lab API is not secure'
                          aria-label='add'
                          classes={{ tooltip: classes.tooltip }}
                        >
                          <WarningIcon className={classes.warningIcon}/>
                        </Tooltip>) : null}
                    </TableCell>

                    <TableCell className={classes.cell}>
                      <DbLabStatus instance={data.data[i]} onlyText={false}/>
                    </TableCell>

                    <TableCell className={classes.cell}>
                      {
                        data.data[i]?.state?.cloning?.numClones ??
                        data.data[i]?.state?.clones?.length ??
                        ''
                      }
                    </TableCell>

                    <TableCell align={'right'}>
                      {(data.data[i].isProcessing) ||
                        (this.state.data.dbLabInstanceStatus.instanceId === i &&
                        this.state.data.dbLabInstanceStatus.isProcessing) ? (
                          <Spinner className={classes.inTableProgress} />
                        ) : null}
                      <IconButton
                        aria-label='more'
                        aria-controls='instance-menu'
                        aria-haspopup='true'
                        onClick={this.openMenu}
                        instanceId={data.data[i].id}
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

      menu = (
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
            key={1}
            onClick={event => this.menuHandler(event, 'addclone')}
          >
            Create clone
          </MenuItem>
          <MenuItem
            key={2}
            onClick={event => this.menuHandler(event, 'refresh')}
          >
            Refresh
          </MenuItem>
          <MenuItem
            disabled={ !deletePermitted }
            key={3}
            onClick={event => this.menuHandler(event, 'destroy')}
          >
            Remove
          </MenuItem>
        </Menu>
      );
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        {projectFilter}

        {table}

        {menu}
      </div>
    );
  }
}

DbLabInstances.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(DbLabInstances);
