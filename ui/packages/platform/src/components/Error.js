/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Paper from '@material-ui/core/Paper';
import Typography from '@material-ui/core/Typography';


const styles = theme => ({
  paper: theme.mixins.gutters({
    paddingTop: 16,
    paddingBottom: 16,
    marginTop: 0
  }),
  errorHead: {
    color: '#c00111',
    fontWeight: 'bold!important',
    fontSize: '16px!important'
  },
  errorText: {
    color: '#c00111'
  }
});

class Error extends Component {
  render() {
    const { classes } = this.props;

    return (
      <div>
        <Paper className={classes.paper}>
          <Typography
            variant='headline'
            component='div'
            className={classes.errorHead}
          >
            ERROR {this.props.code ? this.props.code : null}
          </Typography>

          <br/>

          <Typography
            component='p'
            className={classes.errorText}
          >
            {this.props.message ? this.props.message :
              'Unknown error occurred. Please try again later.'}
          </Typography>
        </Paper>
      </div>
    );
  }
}

Error.propTypes = {
  classes: PropTypes.object.isRequired
};

export default withStyles(styles)(Error);
