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
  Table, TableBody, TableCell,
  TableHead, TableRow, Button, Checkbox, InputAdornment,
  IconButton, OutlinedInput, Tooltip, Box
} from '@material-ui/core';
import dompurify from 'dompurify';
import { formatDistanceToNowStrict } from 'date-fns';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { StubContainer } from '@postgres.ai/shared/components/StubContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import ProductCard from './ProductCard';
import Link from './Link';
import Urls from '../utils/urls';
import format from '../utils/format';
import qs from 'qs';

const PAGE_SIZE = 20;

const getStyles = theme => ({
  root: {
    ...styles.root,
    display: 'flex',
    flexDirection: 'column',
    paddingBottom: '20px'
  },
  filterSelect: {
    ...styles.inputField,
    width: 150,
    marginRight: '15px'
  },
  searchFilter: {
    ...styles.inputField,
    '& input.MuiInputBase-input': {
      paddingLeft: '0px!important'
    }
  },
  checkboxTableCell: {
    width: '30px',
    padding: 0,
    paddingTop: 10,
    verticalAlign: 'top'
  },
  tag: {
    'border': '1px solid ' + colors.consoleStroke,
    'borderRadius': 3,
    'fontSize': 10,
    'lineHeight': '16px',
    'display': 'inline-block',
    'marginRight': 8,
    'backgroundColor': 'white',
    '&:last-child': {
      marginRight: 0
    }
  },
  tagName: {
    fontSize: 10,
    color: colors.pgaiDarkGray,
    borderRight: '1px solid ' + colors.consoleStroke,
    paddingLeft: 3,
    paddingRight: 3,
    backgroundColor: 'white'
  },
  tagValue: {
    fontSize: 10,
    color: colors.secondary2.main,
    paddingLeft: 3,
    paddingRight: 3,
    backgroundColor: 'white'
  },
  twoSideRow: {
    display: 'flex'
  },
  twoSideCol1: {

  },
  twoSideCol2: {
    flexGrow: 1,
    textAlign: 'right'
  },
  timeLabel: {
    lineHeight: '16px',
    fontSize: 12,
    color: colors.pgaiDarkGray
  },
  query: {
    'marginTop': 10,
    'marginBottom': 10,
    'fontSize': 14,
    'lineHeight': '16px',
    'fontFamily': '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
    'color': 'black',
    '& > b': {
      color: colors.secondary2.main
    },
    'height': 32,
    'overflow': 'hidden',
    'text-overflow': 'ellipsis',
    '-webkit-line-clamp': '2',
    '-webkit-box-orient': 'vertical',
    'display': '-webkit-box'
  },
  showMoreContainer: {
    marginTop: 20,
    textAlign: 'center'
  },
  cardCell: {
    paddingLeft: 0,
    paddingRight: 0
  },
  toolTip: {
    fontSize: '10px!important',
    maxWidth: '420px!important'
  },
  searchIcon: {
    marginRight: 0
  },
  filterContainer: {
    display: 'flex',
    [theme.breakpoints.down('sm')]: {
      display: 'block'
    }
  },
  filterButton: {
    height: '32px',
    marginRight: 10,
    marginBottom: 10
  },
  tableHead: {
    ...styles.tableHead,
    fontWeight: 'normal!important',
    padding: 0,
    paddingBottom: 5,
    paddingTop: 5
  },
  tableHeadActions: {
    ...styles.tableHeadActions
  },
  headCheckboxTableCell: {
    width: '30px',
    padding: 0,
    verticalAlign: 'top',
    paddingBottom: 5,
    paddingTop: 5
  },
  centeredBox: {
    display: 'flex',
    justifyContent: 'center',
    height: 'calc(100vh - 250px)',
    mixHeight: '220px'
  },
  centeredProgress: {
    marginTop: 'calc((100vh - 250px) / 2 - 30px)'
  }
});

function Tag(props) {
  return (
    <div
      className={props.classes.tag}
      onClick={
        (event) => {
          if (props.onClick) {
            props.onClick(event);
          }
          event.stopPropagation();
          return false;
        }
      }
    >
      {props.name && (<span className={props.classes.tagName}>
        {props.name}
      </span>)}
      {props.tooltip ? (
        <Tooltip title={props.tooltip} classes={{ tooltip: props.classes.toolTip }}>
          <span className={props.classes.tagValue}>
            {props.value}
          </span>
        </Tooltip>) : (
        <span className={props.classes.tagValue}>
          {props.value}
        </span>
      )}
    </div>
  );
}

class JoeHistory extends Component {
  buildFilter() {
    const { session, project, command, fingerprint, author, search, is_favorite } = this.props;
    let filters = [];
    let filter = '';

    if (author) {
      filters.push('author:' + format.quoteStr(author));
    }
    if (command) {
      filters.push('command:' + command);
    }
    if (fingerprint) {
      filters.push('fingerprint:' + fingerprint);
    }
    if (project) {
      filters.push('project:' + format.quoteStr(project));
    }
    if (session) {
      filters.push('session:' + session);
    }
    if (search) {
      filters.push(search);
    }
    if (is_favorite) {
      filters.push('is:favorite');
    }

    filter = filters.join(' ');
    setTimeout(() => this.setState({ searchFilter: filter, selectedRows: {} }), 1);

    return filter;
  }

  addInstance() {
    this.props.history.push(Urls.linkJoeInstanceAdd(this.props));
  }

  componentDidMount() {
    const that = this;
    const { session, project, command, fingerprint, author, orgId,
      search, is_favorite } = this.props;
    const dataAvailabilityFrom = null;

    this.buildFilter();

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const commands = this.data && this.data.commands ?
        this.data.commands : null;
      const projects = this.data && this.data.projects ?
        this.data.projects : null;

      if (auth && auth.token && !commands.isProcessing && !commands.error &&
        !that.state) {
        Actions.getJoeSessionCommands(
          auth.token,
          {
            orgId,
            session,
            fingerprint,
            command,
            project,
            author,
            search,
            isFavorite: is_favorite,
            startAt: dataAvailabilityFrom,
            limit: PAGE_SIZE
          }
        );
      }

      if (auth && auth.token && !projects.isProcessing && !projects.error &&
        !that.state) {
        Actions.getProjects(auth.token, orgId);
      }

      that.setState({ data: this.data });
    });

    let contentContainer = document.getElementById('content-container');
    if (contentContainer) {
      contentContainer.addEventListener('scroll', () => {
        if (contentContainer.scrollTop >=
          (contentContainer.scrollHeight - contentContainer.offsetHeight)) {
          if (that.refs.showMoreBtn) {
            that.refs.showMoreBtn.click();
          }
        }
      });
    }

    Actions.refresh();
  }

  showMore() {
    const { orgId, session, project, command, fingerprint, author,
      is_favorite, search } = this.props;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const commands = this.state.data && this.state.data.commands ?
      this.state.data.commands : null;
    const dataAvailabilityFrom = null;
    let lastId = null;

    if (commands && commands.data && commands.data.length) {
      lastId = commands.data[commands.data.length - 1].id;
    }

    if (auth && auth.token && !commands.isProcessing && lastId) {
      Actions.getJoeSessionCommands(
        auth.token,
        {
          orgId,
          session,
          fingerprint,
          command,
          project,
          author,
          search,
          isFavorite: is_favorite,
          startAt: dataAvailabilityFrom,
          limit: PAGE_SIZE,
          lastId
        }
      );
    }
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  componentDidUpdate(prevProps) {
    if (JSON.stringify(prevProps) !== JSON.stringify(this.props)) {
      let filter = this.buildFilter();
      this.applyFilter(filter, false);
    }
  }

  onCommandClick(event, project, sessionId, id) {
    const { org } = this.props;

    this.props.history.push(
      '/' + org +
      '/' + project +
      '/sessions/' + sessionId +
      '/commands/' + id
    );
  }

  setFilter(value) {
    this.setState({
      searchFilter: value
    });

    this.applyFilter(value);
  }

  getQueryParams(filterValue) {
    let queryParams = {};
    let filters = filterValue.split(/[;,(\s)]/);

    for (let f in filters) {
      if (!filters.hasOwnProperty(f)) {
        continue;
      }

      let filter = filters[f].split(/[:=]/);

      if (filter.length > 1 && filter[1].length) {
        if (filter[0].trim().toLowerCase().startsWith('is')) {
          queryParams[filter[0].trim() + '_' + filter[1]] = true;
        } else {
          queryParams[filter[0].trim()] = filter[1];
        }
      }
    }

    return queryParams;
  }

  applyFilter(value, changeUrl = true) {
    const { orgId } = this.props;
    let filterValue = typeof value !== 'undefined' ? value : this.state.searchFilter;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const dataAvailabilityFrom = null;
    let queryParams = {};

    let filters = filterValue.split('"');
    if (Array.isArray(filters) && filters.length > 1) {
      for (let i = 0; i < filters.length; i++) {
        if (i > 0 && (filters[i - 1].endsWith('=') || filters[i - 1].endsWith(':'))) {
          let paramName = filters[i - 1].split(/[;,(\s)]/);
          if (Array.isArray(paramName)) {
            paramName = paramName[paramName.length - 1].trim('');
            paramName = paramName.substring(0, paramName.length - 1);
          }
          queryParams[paramName] = filters[i];
        } else {
          queryParams = {
            ...queryParams,
            ...this.getQueryParams(filters[i])
          };
        }
      }
    } else {
      queryParams = this.getQueryParams(filterValue);
    }


    let search = filterValue;
    for (let f in queryParams) {
      if (!queryParams.hasOwnProperty(f)) {
        continue;
      }
      if (f.startsWith('is')) {
        search = search.split(f.replace('_', ':')).join('');
      } else {
        search = search.split(f + ':' + format.quoteStr(queryParams[f])).join('');
        search = search.split(f + ':"' + queryParams[f] + '"').join('');
        search = search.split(f + '=' + format.quoteStr(queryParams[f])).join('');
        search = search.split(f + '="' + queryParams[f] + '"').join('');
      }
    }
    queryParams.search = search.trim();

    Object.keys(queryParams).forEach((key) =>
      (queryParams[key] === null || queryParams[key] === '') &&
      delete queryParams[key]);

    if (changeUrl) {
      let url = '';
      let org = this.props.org;
      url = '/' + org + '/sessions/';
      url = url + '?' + qs.stringify(queryParams);
      this.props.history.push(url);
    }

    if (auth && auth.token) {
      Actions.getJoeSessionCommands(auth.token, {
        ...queryParams,
        orgId: orgId,
        isFavorite: queryParams['is_favorite'],
        startAt: dataAvailabilityFrom,
        limit: PAGE_SIZE
      });
    }
  }

  clearFilter() {
    this.setState({ searchFilter: '' });
    this.applyFilter('');
  }

  clearFilters() {
    const { auth, org, orgId } = this.props;
    const dataAvailabilityFrom = null;

    const url = '/' + org + '/sessions/';
    this.props.history.push(url);

    if (auth && auth.token) {
      Actions.getJoeSessionCommands(auth.token, {
        orgId,
        startAt: dataAvailabilityFrom,
        limit: PAGE_SIZE
      });
    }
  }

  getAuthor(command) {
    if (command['slack_username']) {
      return command['slack_username'];
    }

    if (command.username) {
      return command.username;
    }

    return command.useremail;
  }

  getCurrentUser() {
    let userName = [];
    if (this.state.data.userProfile.data.info.first_name) {
      userName.push(this.state.data.userProfile.data.info.first_name);
    }
    if (this.state.data.userProfile.data.info.last_name) {
      userName.push(this.state.data.userProfile.data.info.last_name);
    }

    userName = userName.join(' ').trim();
    if (userName) {
      return userName;
    }

    return this.state.data.userProfile.data.info.email;
  }

  getSessionId(command) {
    return command['joe_session_id'];
  }

  getProject(command) {
    return command['project_name'];
  }

  getChannel(command) {
    return command['slack_channel'];
  }

  onSelectRow(event, rowId) {
    let selectedRows = this.state.selectedRows;

    if (selectedRows[rowId] && !event.target.checked) {
      delete selectedRows[rowId];
    } else {
      selectedRows[rowId] = event.target.checked;
    }

    this.setState({ selectedRows: selectedRows });
  }

  onCheckBoxClick(event) {
    event.stopPropagation();
  }

  onSelectAllClick(event, commands) {
    if (!event.target.checked) {
      this.setState({ selectedRows: {} });
      return;
    }

    let selectedRows = {};
    for (let i in commands) {
      if (commands.hasOwnProperty(i)) {
        selectedRows[commands[i].id] = true;
      }
    }

    this.setState({ selectedRows: selectedRows });
  }

  deleteCommands() {
    const count = Object.keys(this.state.selectedRows).length;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;

    /* eslint no-alert: 0 */
    if (window.confirm('Are you sure you want to delete ' + count +
      ' command(s)?') === true) {
      let commands = [];
      for (let i in this.state.selectedRows) {
        if (this.state.selectedRows.hasOwnProperty(i)) {
          commands.push(parseInt(i, 10));
        }
      }

      Actions.deleteJoeCommands(auth.token, commands);
      this.setState({ selectedRows: {} });
    }
  }

  render() {
    const that = this;
    const { classes, auth, org, orgId } = this.props;

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        org={org}
        breadcrumbs={[
          { name: 'SQL optimization history', url: null }
        ]}
      />
    );

    const pageTitle = (
      <ConsolePageTitle title='SQL optimization history'/>
    );

    if (!this.state || !this.state.data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}
          {pageTitle}

          <PageSpinner/>
        </div>
      );
    }

    const commandStore = this.state.data.commands || null;
    const commands = commandStore.data || [];

    const isFilterAvailable = (commandStore && commandStore.isHistoryExists &&
      commandStore.isHistoryExists[orgId]) ||
      ((commands.length > 0) ||
      (commands.length === 0 &&
      (this.state.searchFilter ? this.state.searchFilter : '') !== ''));
    const searchFilter = (
      <div className={classes.filterContainer}>
        <Button
          variant='outlined'
          className={classes.filterButton}
          disabled={commandStore && commandStore.isProcessing}
          onClick={() => this.setFilter('author:' + format.quoteStr(this.getCurrentUser()))}
        >
          <nobr>My history</nobr>
        </Button>
        {<Button
          variant='outlined'
          className={classes.filterButton}
          disabled={commandStore && commandStore.isProcessing}
          onClick={() => this.setFilter('is:favorite')}
        >
          <nobr>Favorites</nobr>
        </Button>}
        <OutlinedInput
          value={that.state.searchFilter}
          onChange={event => this.setState({ searchFilter: event.target.value })}
          onKeyDown={event => {
            if (event.keyCode === 13) {
              this.applyFilter();
            }
          }}
          inputProps={{
            name: 'searchFilter',
            id: 'searchFilter'
          }}
          endAdornment={that.state.searchFilter && that.state.searchFilter.length ?
            (<InputAdornment position='end'>
              <IconButton
                onClick={event => this.clearFilter(event)}
                onMouseDown={event => this.clearFilter(event)}
                edge='end'
              >
                {icons.closeIcon}
              </IconButton>
            </InputAdornment>) : null}
          startAdornment={<InputAdornment position='start' className={classes.searchIcon}>
            <IconButton
              edge='start'
              disabled
              className={classes.searchIcon}
            >
              {icons.searchIcon}
            </IconButton>
          </InputAdornment>}
          fullWidth
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel
          }}
          FormHelperTextProps={{
            style: styles.inputFieldHelper
          }}
          variant='outlined'
          className={classes.searchFilter}
        />
      </div>
    );

    if ((commandStore && commandStore.error)) {
      return (
        <div>
          {breadcrumbs}
          {pageTitle}
          {isFilterAvailable && searchFilter}
          <Error/>
        </div>
      );
    }

    if (!commandStore || !commandStore.data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}
          {pageTitle}

          <PageSpinner />
        </div>
      );
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}
        {pageTitle}

        <div>
          {isFilterAvailable && searchFilter}
        </div>

        {commandStore && commandStore.isProcessing &&
          <div>
            <Box className={classes.centeredBox}>
              <Spinner size='lg' className={classes.centeredProgress} />
            </Box>
          </div>
        }

        {commands && commands.length > 0 ? (
          <div>
            <HorizontalScrollContainer>
              <Table className={classes.table}>
                {this.props.userIsOwner && <TableHead>
                  <TableRow className={classes.row}>
                    {this.props.userIsOwner &&
                      <TableCell className={classes.headCheckboxTableCell}>
                        <Checkbox
                          indeterminate={Object.keys(this.state.selectedRows).length > 0}
                          checked={Object.keys(this.state.selectedRows).length === commands.length}
                          onChange={event => this.onSelectAllClick(event, commands)}
                          onClick={event => {
                            this.onCheckBoxClick(event);
                            event.stopPropagation();
                          }}
                        />
                      </TableCell>}
                    <TableCell className={classes.tableHead}>
                      {this.props.userIsOwner && (
                        <div>
                          { commandStore.isDeleting ? (<span>Processing...</span>) : (
                            <div>
                              {Object.keys(this.state.selectedRows).length > 0 ? (
                                <span>
                                  Selected: {Object.keys(this.state.selectedRows).length} rows
                                </span>
                              ) : 'Select table rows to process them' }
                            </div>
                          )}
                          <div className={classes.tableHeadActions}>
                            <Button
                              variant='contained'
                              color='primary'
                              disabled={Object.keys(this.state.selectedRows).length === 0
                                || commandStore.isDeleting}
                              onClick={event => this.deleteCommands(event)}
                            >
                              Delete
                              {commandStore && commandStore.isDeleting ? (
                                <span>
                                  &nbsp;
                                  <Spinner size='sm' />
                                </span>
                              ) : '' }
                            </Button>
                          </div>
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                </TableHead> }

                <TableBody>
                  {commands.map(c => {
                    if (c) {
                      return (
                        <TableRow
                          hover={false}
                          className={classes.row}
                          key={c.id}
                          onClick={event => {
                            this.onCommandClick(
                              event,
                              this.getProject(c),
                              this.getSessionId(c), c.id
                            );
                            return false;
                          }}
                          style={{ cursor: 'pointer' }}
                        >
                          {this.props.userIsOwner &&
                            <TableCell className={classes.checkboxTableCell}>
                              <Checkbox
                                checked={!!this.state.selectedRows[c.id]}
                                onChange={event => this.onSelectRow(event, c.id)}
                                onClick={event => {
                                  this.onCheckBoxClick(event);
                                  event.stopPropagation();
                                }}
                              />
                            </TableCell>
                          }
                          <TableCell className={classes.cardCell}>
                            <div className={classes.twoSideRow}>
                              <div className={classes.twoSideCol1}>
                                {c.command &&
                                  <Tag
                                    classes={classes}
                                    name='command'
                                    value={c.command.substring(0, 30)}
                                    onClick={() => this.setFilter('command:' + c.command)}
                                  />
                                }

                                {this.getAuthor(c) &&
                                  <Tag
                                    classes={classes}
                                    name='author'
                                    value={this.getAuthor(c)}
                                    onClick={() =>
                                      this.setFilter('author:' + format.quoteStr(this.getAuthor(c)))
                                    }
                                  />
                                }

                                <Tooltip
                                  title={format.formatTimestampUtc(c['created_at'])}
                                  classes={{ tooltip: classes.toolTip }}
                                >
                                  <span className={classes.timeLabel}>
                                    {
                                      formatDistanceToNowStrict(
                                        new Date(c.created_at),
                                        { addSuffix: true }
                                      )
                                    }
                                  </span>
                                </Tooltip>
                              </div>
                              {<div className={classes.twoSideCol2}>
                                {!!c.is_favorite && <span
                                  onClick={
                                    (event) => {
                                      Actions.joeCommandFavorite(auth.token, c.id, !c.is_favorite);
                                      event.stopPropagation();
                                    }}
                                >
                                  {icons.favoriteOnIcon}
                                </span>}
                                {!c.is_favorite && <span
                                  onClick={
                                    (event) => {
                                      Actions.joeCommandFavorite(auth.token, c.id, !c.is_favorite);
                                      event.stopPropagation();
                                    }}
                                >
                                  {icons.favoriteOffIcon}
                                </span>}
                              </div>}
                            </div>

                            {c.query &&
                              <div
                                className={classes.query}
                                dangerouslySetInnerHTML={
                                  { __html: dompurify.sanitize(format.formatSql(c.query)) }
                                }
                              />
                            }

                            <div className={classes.twoSideRow}>
                              <div className={classes.twoSideCol1}>
                                <Tag
                                  classes={classes}
                                  name='project'
                                  value={this.getProject(c)}
                                  onClick={() => this.setFilter('project:' + this.getProject(c))}
                                />

                                {this.getChannel(c) &&
                                  <Tag
                                    classes={classes}
                                    name='channel'
                                    value={this.getChannel(c)}
                                    onClick={() => this.setFilter('channel:' + this.getChannel(c))}
                                  />
                                }

                                <Tag
                                  classes={classes}
                                  name='session'
                                  value={'#' + this.getSessionId(c)}
                                  onClick={() => this.setFilter('session:' + this.getSessionId(c))}
                                />
                              </div>
                              <div className={classes.twoSideCol2}>
                                {c.fingerprint &&
                                  <Tag
                                    classes={classes}
                                    value='find similar'
                                    tooltip={'fingerprint: ' + c.fingerprint}
                                    onClick={() => this.setFilter('fingerprint:' + c.fingerprint)}
                                  />}
                              </div>
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    }

                    return null;
                  })}
                </TableBody>
              </Table>
            </HorizontalScrollContainer>
            <div className={classes.showMoreContainer}>
              {commandStore && commandStore.isProcessing &&
                <Spinner size='lg' className={classes.progress} />}
              {commandStore && !commandStore.isProcessing && !commandStore.isComplete &&
                <Button
                  ref='showMoreBtn'
                  variant='outlined'
                  color='secondary'
                  className={classes.button}
                  onClick={() => this.showMore()}
                  disabled={commandStore && commandStore.isProcessing}
                >
                  Show more
                </Button>
              }
            </div>
          </div>) : null
        }

        { commands && commands.length === 0 && commandStore.isProcessed &&
          <StubContainer>
            <ProductCard
              inline
              title={ this.state.searchFilter === '' ?
                'There is no Joe Bot history yet' : 'No commands matching the filters.' }
              actions={this.state.searchFilter !== '' ? [{
                id: 'clearFiltersButton',
                content: <Button
                  variant='contained'
                  color='primary'
                  disabled={commandStore && commandStore.isProcessing}
                  onClick={() => this.clearFilter()}
                >
                  <nobr>Clear filters</nobr>
                </Button>
              }] : [{
                id: 'addInstanceButton',
                content: <Button
                  variant='contained'
                  color='primary'
                  disabled={commandStore && commandStore.isProcessing}
                  onClick={() => this.addInstance()}
                >
                  <nobr>Add instance</nobr>
                </Button>
              }]}
              icon= { icons.joeHistoryLogo }
            >
              {this.state.searchFilter === '' ? (
                <p>
                  Joe Bot is a virtual DBA for SQL Optimization. Joe helps engineers
                  quickly troubleshoot and optimize SQL. Joe runs on top of the
                  Database Lab Engine.
                  (
                  <Link
                    link='https://postgres.ai/docs/joe'
                    target='_blank'
                  >
                    Learn more
                  </Link>
                  ).
                </p>) : (
                <p>
                  We couldn't find any commands matching your filters, try another.
                </p>)}
            </ProductCard>
          </StubContainer> }
      </div>
    );
  }
}

JoeHistory.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(JoeHistory);
