/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Snackbar from '@material-ui/core/Snackbar';
import CloseIcon from '@material-ui/icons/Close';
import IconButton from '@material-ui/core/IconButton';
import ErrorOutlineIcon from '@material-ui/icons/ErrorOutline';
import WarningIcon from '@material-ui/icons/Warning';
import InfoIcon from '@material-ui/icons/Info';
import CheckCircleIcon from '@material-ui/icons/CheckCircle';

import Store from '../stores/store';
import Actions from '../actions/actions';

const styles = () => ({
  defaultNotification: {
  },
  errorNotification: {
    '& > div.MuiSnackbarContent-root': {
      backgroundColor: '#f44336!important'
    }
  },
  informationNotification: {
    '& > div.MuiSnackbarContent-root': {
      backgroundColor: '#2196f3!important'
    }
  },
  warningNotification: {
    '& > div.MuiSnackbarContent-root': {
      backgroundColor: '#ff9800!important'
    }
  },
  successNotification: {
    '& > div.MuiSnackbarContent-root': {
      backgroundColor: '#4caf50!important'
    }
  },
  svgIcon: {
    marginBottom: -2,
    marginRight: 5
  }
});

class Notification extends Component {
  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });
    });
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  closeNotification = (event, reason) => {
    if (reason === 'clickaway') {
      return;
    }

    Actions.hideNotification();
  };

  render() {
    const { classes } = this.props;

    if (this.state && this.state.data && this.state.data.notification &&
      this.state.data.notification.message) {
      let className;
      let message;

      switch (this.state.data.notification.type) {
      case 'error':
        className = classes.errorNotification;
        message = (
          <span>
            <ErrorOutlineIcon className={classes.svgIcon}/>
            {this.state.data.notification.message}
          </span>
        );
        break;
      case 'warning':
        // Warning
        className = classes.warningNotification;
        message = (
          <span>
            <WarningIcon className={classes.svgIcon}/>
            {this.state.data.notification.message}
          </span>
        );
        break;
      case 'information':
        // Info
        className = classes.informationNotification;
        message = (
          <span>
            <InfoIcon className={classes.svgIcon}/>
            {this.state.data.notification.message}
          </span>
        );
        break;
      case 'success':
        // CheckCircle
        className = classes.successNotification;
        message = (
          <span>
            <CheckCircleIcon className={classes.svgIcon}/>
            {this.state.data.notification.message}
          </span>
        );
        break;
      default:
        className = classes.defaultNotification;
        message = this.state.data.notification.message;
      }

      return (
        <Snackbar
          open={true}
          onClose={this.closeNotification}
          message={message}
          className={className}
          autoHideDuration={this.state.data.notification.duration}
          action={
            <React.Fragment>
              <IconButton
                size='small'
                aria-label='close'
                color='inherit'
                onClick={this.closeNotification}
              >
                <CloseIcon fontSize='small' />
              </IconButton>
            </React.Fragment>
          }
        />
      );
    }

    return null;
  }
}

Notification.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(Notification);
