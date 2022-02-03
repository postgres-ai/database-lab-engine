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
  TableHead, TableRow, Button
} from '@material-ui/core';
import * as timeago from 'timeago.js';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { StubContainer } from '@postgres.ai/shared/components/StubContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import ProductCard from 'components/ProductCard';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import format from '../utils/format';
import DbLabStatus from './DbLabStatus';


const PAGE_SIZE = 20;

const getStyles = () => ({
  root: {
    ...styles.root,
    paddingBottom: '20px',
    display: 'flex',
    flexDirection: 'column'
  },
  tableHead: {
    ...styles.tableHead,
    textAlign: 'left'
  },
  tableCell: {
    textAlign: 'left'
  },
  showMoreContainer: {
    marginTop: 20,
    textAlign: 'center'
  }
});

class DbLabSessions extends Component {
  componentDidMount() {
    const that = this;
    const { orgId } = this.props;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const sessions = this.data && this.data.dbLabSessions ?
        this.data.dbLabSessions : null;

      if (auth && auth.token && !sessions.isProcessing && !sessions.error &&
        !that.state) {
        Actions.getDbLabSessions(auth.token, { orgId, limit: PAGE_SIZE });
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

  componentWillUnmount() {
    this.unsubscribe();
  }

  onSessionClick(event, sessionId) {
    const { org } = this.props;

    this.props.history.push(
      '/' + org +
      '/observed-sessions/' + sessionId
    );
  }

  formatStatus(status) {
    const { classes } = this.props;
    let icon = null;
    let className = null;
    let label = status;
    if (status.length) {
      label = status.charAt(0).toUpperCase() + status.slice(1);
    }

    switch (status) {
    case 'passed':
      icon = icons.okIcon;
      className = classes.passedStatus;
      break;
    case 'failed':
      icon = icons.failedIcon;
      className = classes.failedStatus;
      break;
    default:
      icon = icons.processingIcon;
      className = classes.processingStatus;
    }

    return (
      <div className={className}>
        <nobr>{icon}&nbsp;{label}</nobr>
      </div>
    );
  }

  showMore() {
    const { orgId } = this.props;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const sessions = this.state.data && this.state.data.dbLabSessions ?
      this.state.data.dbLabSessions : [];
    let lastId = null;

    if (sessions && sessions.data && sessions.data.length) {
      lastId = sessions.data[sessions.data.length - 1].id;
    }

    if (auth && auth.token && !sessions.isProcessing && lastId) {
      Actions.getDbLabSessions(
        auth.token,
        {
          orgId,
          limit: PAGE_SIZE,
          lastId
        }
      );
    }
  }

  render() {
    const { classes, org } = this.props;

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        org={org}
        breadcrumbs={[
          { name: 'Database Lab observed sessions', url: null }
        ]}
      />
    );

    const pageTitle = (
      <ConsolePageTitle title='Database Lab observed sessions' label={'Experimental'}/>
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

    const sessionsStore = this.state.data && this.state.data.dbLabSessions || null;
    const sessions = sessionsStore && sessionsStore.data || [];

    if ((sessionsStore && sessionsStore.error)) {
      return (
        <div>
          {breadcrumbs}

          {pageTitle}

          <Error/>
        </div>
      );
    }

    if (!sessionsStore || !sessionsStore.data) {
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

        {sessions && sessions.length > 0 ? (
          <div>
            <HorizontalScrollContainer>
              <Table className={classes.table}>
                <TableHead>
                  <TableCell className={classes.tableHead}>
                    Status
                  </TableCell>
                  <TableCell className={classes.tableHead}>
                    Session
                  </TableCell>
                  <TableCell className={classes.tableHead}>
                    Project/Instance
                  </TableCell>
                  <TableCell className={classes.tableHead}>
                    Commit
                  </TableCell>
                  <TableCell className={classes.tableHead}>
                    Checklist
                  </TableCell>
                  <TableCell className={classes.tableHead}>
                    &nbsp;
                  </TableCell>
                </TableHead>

                <TableBody>
                  {sessions.map(s => {
                    if (s) {
                      return (
                        <TableRow
                          hover={false}
                          className={classes.row}
                          key={s.id}
                          onClick={event => {
                            this.onSessionClick(
                              event,
                              s.id
                            );
                            return false;
                          }}
                          style={{ cursor: 'pointer' }}
                        >
                          <TableCell className={classes.tableCell}>
                            <DbLabStatus session={{ status: s.result && s.result.status ?
                              s.result.status : 'processing' }}/>
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            #{s.id}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            {s.tags && s.tags.project_id ? s.tags.project_id : '-'}/
                            {s.tags && s.tags.instance_id ? s.tags.instance_id : '-'}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            {icons.branch}&nbsp;{s.tags && s.tags.branch && s.tags.revision ?
                              s.tags.branch + '/' + s.tags.revision : '-'}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            {s.result && s.result.summary && s.result.summary.checklist ? (
                              <div>
                                {Object.keys(s.result.summary.checklist).map(function (key) {
                                  return (
                                    <span title={format.formatDbLabSessionCheck(key)}>
                                      {s.result.summary.checklist[key] ? icons.okLargeIcon :
                                        icons.failedLargeIcon}&nbsp;
                                    </span>
                                  );
                                })}
                              </div>
                            ) : (icons.processingLargeIcon)}
                          </TableCell>
                          <TableCell className={classes.tableCell}>
                            <div>
                              {s.duration > 0 ||
                                (s.result && s.result.summary && s.result.summary.elapsed) ? (
                                  <span>
                                    {icons.timer}&nbsp;
                                    {s.result && s.result.summary && s.result.summary.elapsed ?
                                      s.result.summary.elapsed :
                                      format.formatSeconds(s.duration, 0, '')
                                    }
                                  </span>
                                ) : '-'}
                            </div>
                            <div>
                              {icons.calendar}&nbsp;created&nbsp;
                              {timeago.format(s.started_at + ' UTC')}
                              {s.tags && s.tags.launched_by ? (
                                <span> by {s.tags.launched_by}</span>
                              ) : ''}
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
              {sessionsStore && sessionsStore.isProcessing &&
                <Spinner className={classes.progress} />}
              {sessionsStore && !sessionsStore.isProcessing && !sessionsStore.isComplete &&
                <Button
                  ref='showMoreBtn'
                  variant='outlined'
                  color='secondary'
                  className={classes.button}
                  onClick={() => this.showMore()}
                  disabled={sessionsStore && sessionsStore.isProcessing}
                >
                  Show more
                </Button>
              }
            </div>
          </div>) : (
          <>
            { sessions && sessions.length === 0 && sessionsStore.isProcessed && <StubContainer>
              <ProductCard
                inline
                title={'There are no Database Lab observed sessions yet.'}
                icon= { icons.databaseLabLogo }
              />
            </StubContainer> }
          </>
        )}
      </div>
    );
  }
}


DbLabSessions.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(DbLabSessions);
