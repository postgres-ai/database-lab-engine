/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import {
  Table, TableBody, TableCell, TableHead, TableRow, Button
} from '@material-ui/core';
import ExpansionPanel from '@material-ui/core/ExpansionPanel';
import ExpansionPanelSummary from '@material-ui/core/ExpansionPanelSummary';
import ExpansionPanelDetails from '@material-ui/core/ExpansionPanelDetails';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';
import TextField from '@material-ui/core/TextField';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { styles } from '@postgres.ai/shared/styles/styles';

import Actions from '../actions/actions';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import Error from './Error';
import Store from '../stores/store';
import Warning from './Warning';
import messages from '../assets/messages';
import format from '../utils/format';

const PAGE_SIZE = 20;

const getStyles = theme => ({
  root: {
    ...styles.root,
    display: 'flex',
    flexDirection: 'column',
    paddingBottom: '20px'
  },
  container: {
    display: 'flex',
    flexWrap: 'wrap'
  },
  textField: {
    marginLeft: theme.spacing(1),
    marginRight: theme.spacing(1),
    width: '80%'
  },
  dense: {
    marginTop: 16
  },
  menu: {
    width: 200
  },
  updateButtonContainer: {
    marginTop: 20,
    textAlign: 'right'
  },
  errorMessage: {
    color: 'red'
  },
  orgsHeader: {
    position: 'relative'
  },
  newOrgBtn: {
    position: 'absolute',
    top: 0,
    right: 10
  },
  timeCell: {
    verticalAlign: 'top',
    minWidth: 200
  },
  expansionPanel: {
    boxShadow: 'none',
    background: 'transparent',
    fontSize: '12px',
    marginBottom: '5px'
  },
  expansionPanelSummary: {
    'display': 'inline-block',
    'padding': '0px',
    'minHeight': '22px',
    '& .MuiExpansionPanelSummary-content': {
      margin: '0px',
      display: 'inline-block'
    },
    '&.Mui-expanded': {
      minHeight: '22px'
    },
    '& .MuiExpansionPanelSummary-expandIcon': {
      display: 'inline-block',
      padding: '0px'
    }
  },
  expansionPanelDetails: {
    padding: '0px',
    [theme.breakpoints.down('md')]: {
      display: 'block'
    }
  },
  actionDescription: {
    marginBottom: '5px'
  },
  code: {
    'width': '100%',
    'margin-top': 0,
    '& > div': {
      paddingTop: 8,
      padding: 8
    },
    'background-color': 'rgb(246, 248, 250)',
    '& > div > textarea': {
      fontFamily: '"Menlo", "DejaVu Sans Mono", "Liberation Mono", "Consolas",' +
        ' "Ubuntu Mono", "Courier New", "andale mono", "lucida console", monospace',
      color: 'black',
      fontSize: '12px'
    }
  },
  showMoreContainer: {
    marginTop: 20,
    textAlign: 'center'
  },
  data: {
    width: '50%',
    [theme.breakpoints.up('md')]: {
      width: '50%',
      marginRight: '10px'
    }
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

const auditTitle = 'Audit log';

class Audit extends Component {
  componentDidMount() {
    const that = this;
    const orgId = this.props.orgId;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const auditLog = this.data && this.data.auditLog ? this.data.auditLog :
        null;

      that.setState({ data: this.data });

      if (auth && auth.token && auditLog &&
        !auditLog.isProcessing && !auditLog.error && !that.state) {
        Actions.getAuditLog(
          auth.token,
          {
            orgId,
            limit: PAGE_SIZE
          }
        );
      }
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

  handleChange = event => {
    const name = event.target.name;
    const value = event.target.value;

    this.setState({
      [name]: value
    });
  }

  buttonAddHandler = () => {
    const org = this.props.org ? this.props.org : null;

    this.props.history.push('/' + org + '/members/add');
  }

  showMore() {
    const { orgId } = this.props;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const logs = this.state && this.state.data ? this.state.data.auditLog : [];
    let lastId = null;

    if (logs && logs.data && logs.data.length) {
      lastId = logs.data[logs.data.length - 1].id;
    }

    if (auth && auth.token && !logs.isProcessing && lastId) {
      Actions.getAuditLog(
        auth.token,
        {
          orgId,
          limit: PAGE_SIZE,
          lastId
        }
      );
    }
  }

  formatAction = (r) => {
    const { classes } = this.props;
    let acted = r.action;
    let actor = r.actor;
    let actorSrc = '';
    let rows = 'row';

    if (!actor) {
      actor = 'Unknown';
      actorSrc = ' (changed directly in database) ';
    }

    if (r.action_data && r.action_data.processed_rows_count) {
      rows = r.action_data.processed_rows_count + ' ' +
        (r.action_data.processed_rows_count > 1 ? 'rows' : 'row');
    }

    switch (r.action) {
    case 'insert':
      acted = ' added ' + rows + ' to';
      break;
    case 'delete':
      acted = ' deleted ' + rows + ' from';
      break;
    default:
      acted = ' updated ' + rows + ' in';
    }

    return (
      <div className={classes.actionDescription}>
        <strong>{actor}</strong>{actorSrc} {acted} table&nbsp;<strong>{r.table_name}</strong>
      </div>
    );
  }

  getDataSectionTitle = (r, before) => {
    switch (r.action) {
    case 'insert':
      return '';
    case 'delete':
      return '';
    default:
      return before ? 'Before:' : 'After:';
    }
  }

  getChangesTitle = (r) => {
    let displayedCount = r.data_before ? r.data_before.length : r.data_after.length;
    let objCount = r.action_data && r.action_data.processed_rows_count ?
      r.action_data.processed_rows_count : null;

    if (displayedCount && objCount > displayedCount) {
      return 'Changes (displayed ' + displayedCount + ' rows out of ' + objCount + ')';
    }

    return 'Changes';
  }

  render() {
    const { classes, orgPermissions, orgId } = this.props;
    const data = this.state && this.state.data ? this.state.data.auditLog : null;
    const logsStore = this.state && this.state.data &&
      this.state.data.auditLog || null;
    const logs = logsStore && logsStore.data || [];

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[{ name: auditTitle }]}
      />
    );

    const pageTitle = (
      <ConsolePageTitle
        title={ auditTitle }
        actions={ [] }
      />
    );

    if (orgPermissions && !orgPermissions.auditLogView) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <Warning>{ messages.noPermissionPage }</Warning>
        </div>
      );
    }

    if (!logsStore || !logsStore.data || (logsStore && logsStore.orgId !== orgId)) {
      return (
        <div className={classes.root}>
          { breadcrumbs }

          { pageTitle }

          <PageSpinner />
        </div>
      );
    }

    if (logsStore.error) {
      return (
        <div className={classes.root}>
          <Error
            message={logsStore.errorMessage}
            code={logsStore.errorCode}
          />
        </div>
      );
    }

    return (
      <div className={classes.root}>
        { breadcrumbs }

        { pageTitle }

        { logs && logs.length > 0 ? (
          <div>
            <HorizontalScrollContainer>
              <Table className={classes.table}>
                <TableHead>
                  <TableRow className={classes.row}>
                    <TableCell>Action</TableCell>
                    <TableCell>Time</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {logs.map(r => {
                    return (
                      <TableRow
                        hover
                        className={classes.row}
                        key={r.id}
                      >
                        <TableCell className={classes.cell}>
                          {this.formatAction(r)}
                          {(r.data_before || r.data_after) && (
                            <div>
                              <ExpansionPanel className={classes.expansionPanel}>
                                <ExpansionPanelSummary
                                  expandIcon={<ExpandMoreIcon />}
                                  aria-controls='panel1b-content'
                                  id='panel1b-header'
                                  className={classes.expansionPanelSummary}
                                >
                                  {this.getChangesTitle(r)}
                                </ExpansionPanelSummary>
                                <ExpansionPanelDetails className={classes.expansionPanelDetails}>
                                  {r.data_before && (
                                    <div className={classes.data}>
                                      {this.getDataSectionTitle(r, true)}
                                      <TextField
                                        variant='outlined'
                                        id={'before_data_' + r.id}
                                        multiline
                                        fullWidth
                                        value={JSON.stringify(r.data_before, null, 4)}
                                        className={classes.code}
                                        margin='normal'
                                        variant='outlined'
                                        InputProps={{
                                          readOnly: true
                                        }}
                                      />
                                    </div>
                                  )}
                                  {r.data_after && (
                                    <div className={classes.data}>
                                      {this.getDataSectionTitle(r, false)}
                                      <TextField
                                        variant='outlined'
                                        id={'after_data_' + r.id}
                                        multiline
                                        fullWidth
                                        value={JSON.stringify(r.data_after, null, 4)}
                                        className={classes.code}
                                        margin='normal'
                                        variant='outlined'
                                        InputProps={{
                                          readOnly: true
                                        }}
                                      />
                                    </div>
                                  )}
                                </ExpansionPanelDetails>
                              </ExpansionPanel>
                            </div>
                          )}
                        </TableCell>
                        <TableCell className={classes.timeCell}>
                          {format.formatTimestampUtc(r.created_at)}
                        </TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </HorizontalScrollContainer>
            <div className={classes.showMoreContainer}>
              {data && data.isProcessing &&
                <Spinner className={classes.progress} />}
              {data && !data.isProcessing && !data.isComplete &&
                <Button
                  ref='showMoreBtn'
                  variant='outlined'
                  color='secondary'
                  className={classes.button}
                  onClick={() => this.showMore()}
                  disabled={data && data.isProcessing}
                >
                  Show more
                </Button>
              }
            </div>
          </div>) : 'Audit log records not found'
        }

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

Audit.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(Audit);
