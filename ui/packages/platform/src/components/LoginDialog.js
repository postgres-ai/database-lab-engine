/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react';
import PropTypes from 'prop-types';
import Button from '@material-ui/core/Button';
import TextField from '@material-ui/core/TextField';
import { withStyles } from '@material-ui/core/styles';
import {
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle
} from '@material-ui/core';

import Store from '../stores/store';
import Actions from '../actions/actions';


const styles = () => ({
  error: {
    color: 'red'
  }
});

class LoginDialog extends React.Component {
  state = {
    open: true,
    email: null,
    password: null,
    emailError: false,
    passwordError: false,
    isProcessing: false
  };

  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });
    });
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleNext = () => {
    const state = this.state;

    if (!state.email) {
      this.setState({ emailError: true });
    } else {
      this.setState({ emailError: false });
    }

    if (!state.password) {
      this.setState({ passwordError: true });
    } else {
      this.setState({ passwordError: false });
    }

    if (state.email && state.password) {
      this.setState({ isProcessing: true });
      this.setState({ emailError: false, passwordError: false });
      Actions.doAuth(state.email, state.password);
    }
  };

  handleCancel = () => {
    this.setState({ open: false });
    window.location = '/';
  };

  handleChange = event => {
    this.setState({
      [event.target.name]: event.target.value
    }, function () {});
  };

  render() {
    const { classes } = this.props;
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : { isProcessing: false, status: null, error: null };
    let errorMessage = null;

    if (auth && auth.error && auth.errorMessage !== 'wrong_reply' &&
      auth.errorMessage !== 'empty_request') {
      errorMessage = (
        <div>
          <DialogContentText>
            <div className={classes.error}>
              {auth.errorMessage}
            </div>
          </DialogContentText>
        </div>
      );
    }

    let password = (
      <div>
        <TextField
          required
          id='password-input'
          label='Password'
          name='password'
          type='password'
          placeholder='Input password here'
          fullWidth
          margin='normal'
          onChange={this.handleChange}
          disabled = {auth.isProcessing}
          error = {this.state.passwordError}
        />
      </div>
    );

    return (
      <div>
        <Dialog
          open={this.state.open}
          onClose={this.handleClose}
          aria-labelledby='form-dialog-title'
        >
          <DialogTitle id='form-dialog-title'>Login</DialogTitle>
          <DialogContent>
            <DialogContentText>
              To enter to dashboard, please enter your email address here.
              We will send you password.
            </DialogContentText>
            <TextField
              required
              autoFocus
              margin='dense'
              id='email'
              label='Email Address'
              type='email'
              name='email'
              fullWidth
              placeholder='Input login here'
              onChange={this.handleChange}
              error={this.state.emailError}
              disabled={auth.isProcessing || auth.status === 'password_sent'}
            />
            {password}
            {errorMessage}
          </DialogContent>
          <DialogActions>
            <Button onClick={this.handleCancel} color='primary' disabled={auth.isProcessing}>
              Cancel
            </Button>
            <Button onClick={this.handleNext} color='primary' disabled={auth.isProcessing}>
              Login
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    );
  }
}

LoginDialog.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(LoginDialog);
