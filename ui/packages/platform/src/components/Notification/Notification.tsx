/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react'
import { Snackbar, IconButton } from '@material-ui/core'
import CloseIcon from '@material-ui/icons/Close'
import ErrorOutlineIcon from '@material-ui/icons/ErrorOutline'
import WarningIcon from '@material-ui/icons/Warning'
import InfoIcon from '@material-ui/icons/Info'
import CheckCircleIcon from '@material-ui/icons/CheckCircle'

import { ClassesType } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'

interface NotificationProps {
  classes: ClassesType
}

interface NotificationState {
  data: {
    notification: {
      type: string
      message: string
      duration: number
    }
  }
}

class Notification extends Component<NotificationProps, NotificationState> {
  componentDidMount() {
    const that = this

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data })
    })
  }

  unsubscribe: () => void
  componentWillUnmount() {
    this.unsubscribe()
  }

  closeNotification = (
    _: React.SyntheticEvent<HTMLButtonElement>,
    reason: string,
  ) => {
    if (reason === 'clickaway') {
      return
    }

    Actions.hideNotification()
  }

  render() {
    const { classes } = this.props

    if (
      this.state &&
      this.state.data &&
      this.state.data.notification &&
      this.state.data.notification.message
    ) {
      let className
      let message

      switch (this.state.data.notification.type) {
        case 'error':
          className = classes.errorNotification
          message = (
            <span>
              <ErrorOutlineIcon className={classes.svgIcon} />
              {this.state.data.notification.message}
            </span>
          )
          break
        case 'warning':
          // Warning
          className = classes.warningNotification
          message = (
            <span>
              <WarningIcon className={classes.svgIcon} />
              {this.state.data.notification.message}
            </span>
          )
          break
        case 'information':
          // Info
          className = classes.informationNotification
          message = (
            <span>
              <InfoIcon className={classes.svgIcon} />
              {this.state.data.notification.message}
            </span>
          )
          break
        case 'success':
          // CheckCircle
          className = classes.successNotification
          message = (
            <span>
              <CheckCircleIcon className={classes.svgIcon} />
              {this.state.data.notification.message}
            </span>
          )
          break
        default:
          className = classes.defaultNotification
          message = this.state.data.notification.message
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
              <IconButton size="small" aria-label="close" color="inherit">
                <CloseIcon fontSize="small" />
              </IconButton>
            </React.Fragment>
          }
        />
      )
    }

    return null
  }
}

export default Notification
