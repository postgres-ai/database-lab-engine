/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import { NavLink } from 'react-router-dom';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Table from '@material-ui/core/Table';
import TableBody from '@material-ui/core/TableBody';
import TableCell from '@material-ui/core/TableCell';
import TableHead from '@material-ui/core/TableHead';
import TableRow from '@material-ui/core/TableRow';
import Typography from '@material-ui/core/Typography';
import Button from '@material-ui/core/Button';

import {
  HorizontalScrollContainer
} from '@postgres.ai/shared/components/HorizontalScrollContainer';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { styles } from '@postgres.ai/shared/styles/styles';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Error from './Error';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import Urls from '../utils/urls';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

const getStyles = () => ({
  root: {
    ...styles.root,
    flex: '1 1 100%',
    display: 'flex',
    flexDirection: 'column'
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
  horizontalMenuItem: {
    'display': 'inline-block',
    'marginRight': 10,
    'font-weight': 'normal',
    'color': 'black',
    '&:visited': {
      color: 'black'
    }
  },
  activeHorizontalMenuItem: {
    'display': 'inline-block',
    'marginRight': 10,
    'font-weight': 'bold',
    'color': 'black'
  },
  fileTypeMenu: {
    marginBottom: 10
  },
  bottomSpace: {
    ...styles.bottomSpace
  }
});

class Report extends Component {
  componentDidMount() {
    const that = this;
    const reportId = this.props.match.params.reportId;
    const type = this.props.match.params.type ?
      this.props.match.params.type : 'md';
    const orgId = this.props.orgId;

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const reports = this.data && this.data.reports ?
        this.data.reports : null;
      const report = this.data && this.data.report ?
        this.data.report : null;

      if (auth && auth.token && !reports.isProcessing &&
        !that.state) {
        Actions.getCheckupReports(auth.token, orgId, 0, reportId);
      }

      if (auth && auth.token && report.reportId !== reportId &&
        !report.isProcessing && !report.error &&
        !reports.error && reports.data && reports.data[0].id) {
        Actions.getCheckupReportFiles(auth.token, reportId, type,
          'filename', 'asc');
      }

      that.setState({ data: this.data });
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  componentDidUpdate(prevProps) {
    const prevType = prevProps.match.params.type ?
      prevProps.match.params.type : 'md';
    const curType = this.props.match.params.type ?
      this.props.match.params.type : 'md';
    const reportId = this.props.match.params.reportId;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;

    if (prevType !== curType) {
      const report = this.state.data && this.state.data.report ?
        this.state.data.report : null;

      if (auth && auth.token &&
        (report.reportId !== reportId || report.type !== curType) &&
        !report.isProcessing && !report.error) {
        Actions.getCheckupReportFiles(auth.token, reportId, curType,
          'filename', 'asc');
      }
    }
  }

  handleClick = (event, id, type) => {
    this.props.history.push(this.getReportFileLink(id, type));
  }

  getReportFileLink(id, type) {
    const reportId = this.props.match.params.reportId;

    return Urls.linkReportFile(this.props, reportId, id, type);
  }

  downloadJsonFiles = () => {
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const reportId = this.props.match.params.reportId;

    Actions.downloadReportJsonFiles(auth.token, reportId);
  }

  render() {
    const { classes } = this.props;
    const data = this.state && this.state.data && this.state.data.report ?
      this.state.data.report : null;
    let reportId = this.props.match.params.reportId;
    const type = this.props.match.params.type ? this.props.match.params.type :
      'md';

    let breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: 'Reports', url: 'reports' },
          { name: 'Report #' + reportId }
        ]}
      />
    );

    let menu = null;
    if (type === 'md') {
      menu = (
        <div>
          <div className={classes.fileTypeMenu}>
            <Typography
              className={classes.activeHorizontalMenuItem}
              color='textPrimary'
            >
              Markdown files
            </Typography>
            <NavLink
              className={classes.horizontalMenuItem}
              to={Urls.linkReport(this.props, reportId, 'json')}
            >
              JSON files
            </NavLink>
          </div>
        </div>
      );
    }

    if (type === 'json') {
      menu = (
        <div>
          <div className={classes.fileTypeMenu}>
            <NavLink
              className={classes.horizontalMenuItem}
              to={Urls.linkReport(this.props, reportId)}
            >
              Markdown files
            </NavLink>
            <Typography
              className={classes.activeHorizontalMenuItem}
              color='textPrimary'
            >
              JSON files
            </Typography>
          </div>

          <Button
            variant='contained'
            color='primary'
            onClick={this.downloadJsonFiles}
            disabled={data && data.isDownloading}
          >
            Download all
            {data && data.isDownloading ? (
              <span>
                &nbsp;
                <Spinner size='sm' />
              </span>
            ) : '' }
          </Button>
        </div>
      );
    }

    let errorWidget = null;
    if (this.state && this.state.data.reports.error) {
      errorWidget = (
        <Error
          message={this.state.data.reports.errorMessage}
          code={this.state.data.reports.errorCode}
        />
      );
    }

    if (this.state && this.state.data && this.state.data.report.error) {
      errorWidget = (
        <Error/>
      );
    }

    if (errorWidget) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {errorWidget}
        </div>
      );
    }

    if (!data || (data && data.isProcessing) || (data && data.reportId !== reportId)) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          <PageSpinner />
        </div>
      );
    }

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {menu}

        {data.data && data.data.length > 0 ? (
          <HorizontalScrollContainer>
            <Table className={classes.table}>
              <TableHead>
                <TableRow className={classes.row}>
                  <TableCell>Filename</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Created</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data.data.map(f => {
                  return (
                    <TableRow
                      hover
                      className={classes.row}
                      key={f.id}
                      onClick={event => this.handleClick(event, f.id, f.type)}
                      style={{ cursor: 'pointer' }}
                    >
                      <TableCell className={classes.cell}>
                        <NavLink to={this.getReportFileLink(f.id, f.type)}>
                          {f.filename}
                        </NavLink>
                      </TableCell>
                      <TableCell className={classes.cell}>{f.type}</TableCell>
                      <TableCell className={classes.cell}>{f.created_formatted}</TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </HorizontalScrollContainer>) : 'This report does not contain any files.'
        }

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

Report.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(Report);
