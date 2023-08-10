/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'
import Button from '@material-ui/core/Button'
import TextField from '@material-ui/core/TextField'
import {
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@material-ui/core'

import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'

interface LoginDialogProps {
  classes: ClassesType
}

interface LoginDialogState {
  open: boolean
  emailError: boolean
  passwordError: boolean
  isProcessing: boolean
  email: string | null
  password: string | null
  data: {
    auth: {
      isProcessing: boolean
      error: boolean
      status: string | null
      errorMessage: string | null
    }
  } | null
}

class LoginDialog extends React.Component<LoginDialogProps, LoginDialogState> {
  state = {
    data: {
      auth: {
        status: null,
        isProcessing: false,
        error: false,
        errorMessage: null,
      },
    },
    open: true,
    email: null,
    password: null,
    emailError: false,
    passwordError: false,
    isProcessing: false,
  }

  unsubscribe: Function
  componentDidMount() {
    const that = this

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      that.setState({ data: this.data })
    })
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  handleNext = () => {
    const state = this.state

    if (!state.email) {
      this.setState({ emailError: true })
    } else {
      this.setState({ emailError: false })
    }

    if (!state.password) {
      this.setState({ passwordError: true })
    } else {
      this.setState({ passwordError: false })
    }

    if (state.email && state.password) {
      this.setState({ isProcessing: true })
      this.setState({ emailError: false, passwordError: false })
      Actions.doAuth(state.email, state.password)
    }
  }

  handleCancel = () => {
    this.setState({ open: false })
    window.location.href = '/'
  }

  render() {
    const { classes } = this.props
    const auth =
      this.state && this.state.data && this.state.data.auth
        ? this.state.data.auth
        : { isProcessing: false, status: null, error: null }
    let errorMessage = null

    if (
      auth &&
      auth.error &&
      auth.errorMessage !== 'wrong_reply' &&
      auth.errorMessage !== 'empty_request'
    ) {
      errorMessage = (
        <div>
          <DialogContentText>
            <div className={classes.error}>{auth.errorMessage}</div>
          </DialogContentText>
        </div>
      )
    }

    let password = (
      <div>
        <TextField
          required
          id="password-input"
          label="Password"
          name="password"
          type="password"
          placeholder="Input password here"
          fullWidth
          margin="normal"
          onChange={(e) => {
            this.setState({
              password: e.target.value,
            })
          }}
          disabled={auth.isProcessing}
          error={this.state.passwordError}
        />
      </div>
    )

    return (
      <div>
        <Dialog open={this.state.open} aria-labelledby="form-dialog-title">
          <DialogTitle id="form-dialog-title">Login</DialogTitle>
          <DialogContent>
            <DialogContentText>
              To enter to dashboard, please enter your email address here. We
              will send you password.
            </DialogContentText>
            <TextField
              required
              autoFocus
              margin="dense"
              id="email"
              label="Email Address"
              type="email"
              name="email"
              fullWidth
              placeholder="Input login here"
              onChange={(e) => {
                this.setState({
                  email: e.target.value,
                })
              }}
              error={this.state.emailError}
              disabled={auth.isProcessing || auth.status === 'password_sent'}
            />
            {password}
            {errorMessage}
          </DialogContent>
          <DialogActions>
            <Button
              onClick={this.handleCancel}
              color="primary"
              disabled={auth.isProcessing}
            >
              Cancel
            </Button>
            <Button
              onClick={this.handleNext}
              color="primary"
              disabled={auth.isProcessing}
            >
              Login
            </Button>
          </DialogActions>
        </Dialog>
      </div>
    )
  }
}

export default LoginDialog
