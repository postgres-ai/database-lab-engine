/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';

import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';


const styles = theme => ({
  root: {
    width: '100%',
    minHeight: '100%',
    zIndex: 1,
    position: 'relative',
    [theme.breakpoints.down('sm')]: {
      maxWidth: '100vw'
    },
    [theme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 200px)'
    },
    [theme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 200px)'
    }
  }
});

class JoeConfig extends Component {
  render() {
    const breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: 'SQL Optimization' },
          { name: 'Configuration' }
        ]}
      />
    );

    return (
      <div>
        {breadcrumbs}
        <br/>
        See configuration guides in&nbsp;
        <a href='https://gitlab.com/postgres-ai/joe' target='blank'>
            postgres-ai/joe
        </a>
        &nbsp;repository.
      </div>
    );
  }
}

JoeConfig.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(JoeConfig);
