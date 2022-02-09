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
import Typography from '@material-ui/core/Typography';
import Paper from '@material-ui/core/Paper';
import Breadcrumbs from '@material-ui/core/Breadcrumbs';
import clsx from 'clsx';

import { colors } from '@postgres.ai/shared/styles/colors';

import { Head, createTitle as createTitleBase } from 'components/Head';

import Store from '../stores/store';
import Actions from '../actions/actions';
import Urls from '../utils/urls';


const styles = () => ({
  pointerLink: {
    cursor: 'pointer'
  },
  breadcrumbsLink: {
    maxWidth: 150,
    textOverflow: 'ellipsis',
    overflow: 'hidden',
    display: 'block',
    cursor: 'pointer',
    whiteSpace: 'nowrap',
    fontSize: '12px',
    lineHeight: '14px',
    textDecoration: 'none',
    color: colors.consoleFadedFont
  },
  breadcrumbsItem: {
    fontSize: '12px',
    lineHeight: '14px',
    color: colors.consoleFadedFont
  },
  breadcrumbsActiveItem: {
    fontSize: '12px',
    lineHeight: '14px',
    color: '#000000'
  },
  breadcrumbPaper: {
    '& a, & a:visited': {
      color: colors.consoleFadedFont
    },
    'padding-bottom': '8px',
    'marginTop': '-10px',
    'font-size': '12px',
    'borderRadius': 0
  },
  breadcrumbPaperWithDivider: {
    borderBottom: `1px solid ${colors.consoleStroke}`
  }
});

const createTitle = (parts) => {
  const filteredParts = parts.filter(part => part !== 'Organizations');
  return createTitleBase(filteredParts);
};

class ConsoleBreadcrumbs extends Component {
  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });
    });
    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  render() {
    const { classes, hasDivider = false } = this.props;
    const breadcrumbs = this.props.breadcrumbs;
    const org = this.props.org ? this.props.org : null;
    const project = this.props.project ? this.props.project : null;
    const orgs = this.state && this.state.data && this.state.data.userProfile.data &&
      this.state.data.userProfile.data.orgs ? this.state.data.userProfile.data.orgs : null;
    const paths = [];
    let lastUrl = '';

    if (!breadcrumbs.length || Urls.isSharedUrl()) {
      return null;
    }

    if (org && orgs && orgs[org]) {
      if (orgs[org].name) {
        paths.push({ name: 'Organizations', url: '/' });
        paths.push({ name: orgs[org].name, url: '/' + org });
        lastUrl = '/' + org;
      }

      if (project && orgs[org].projects && orgs[org].projects[project]) {
        paths.push({ name: orgs[org].projects[project].name, url: null });
        lastUrl = '/' + org + '/' + project;
      }
    }

    for (let i = 0; i < breadcrumbs.length; i++) {
      if (breadcrumbs[i].url && breadcrumbs[i].url.indexOf('/') === -1) {
        breadcrumbs[i].url = lastUrl + '/' + breadcrumbs[i].url;
        lastUrl = breadcrumbs[i].url;
      }
      breadcrumbs[i].isLast = (i === breadcrumbs.length - 1);
      paths.push(breadcrumbs[i]);
    }

    return (
      <>
        <Head title={createTitle(paths.map(path => path.name))} />
        <Paper
          elevation={0}
          className={clsx(
            classes.breadcrumbPaper, hasDivider && classes.breadcrumbPaperWithDivider
          )}
        >
          <Breadcrumbs aria-label='breadcrumb'>
            {paths.map((b) => {
              return (
                <span key={b}>
                  {b.url ? (
                    <NavLink color='inherit' to={b.url} className={classes.breadcrumbsLink}>
                      {b.name}
                    </NavLink>
                  ) : (
                    <Typography
                      color='textPrimary'
                      className={b.isLast ? classes.breadcrumbsActiveItem : classes.breadcrumbsItem}
                    >
                      {b.name}
                    </Typography>
                  )}
                </span>
              );
            })}
          </Breadcrumbs>
        </Paper>
      </>
    );
  }
}

ConsoleBreadcrumbs.propTypes = {
  hasDivider: PropTypes.bool,
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  org: PropTypes.string,
  project: PropTypes.string,
  breadcrumbs: PropTypes.arrayOf(
    PropTypes.shape({
      name: PropTypes.string.isRequired,
      url: PropTypes.string
    }).isRequired
  ).isRequired
};

export default withStyles(styles, { withTheme: true })(ConsoleBreadcrumbs);
