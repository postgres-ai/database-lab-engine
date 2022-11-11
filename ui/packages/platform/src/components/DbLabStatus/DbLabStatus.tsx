/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import Brightness1Icon from '@material-ui/icons/Brightness1'
import Tooltip from '@material-ui/core/Tooltip'

import { icons } from '@postgres.ai/shared/styles/icons'
import { ClassesType } from '@postgres.ai/platform/src/components/types'

import Format from '../../utils/format'
import { Clone } from '@postgres.ai/shared/types/api/entities/clone'
import { DbLabStatusProps } from 'components/DbLabStatus/DbLabStatusWrapper'

export interface DbLabStatusInstance {
  state: {
    status: {
      code: string
      message: string
    }
  }
}
interface DbLabStatusWithStylesProps extends DbLabStatusProps {
  classes: ClassesType
}

class DbLabStatus extends Component<DbLabStatusWithStylesProps> {
  getCloneStatus = (
    clone: Clone,
    onlyText: boolean,
    showDescription: boolean,
  ) => {
    const { classes } = this.props
    let className = classes?.cloneReadyStatus

    if (!clone.status) {
      return null
    }

    switch (clone.status.code) {
      case 'OK':
        className = classes?.cloneReadyStatus
        break
      case 'CREATING':
        className = classes?.cloneCreatingStatus
        break
      case 'DELETING':
        className = classes?.cloneDeletingStatus
        break
      case 'RESETTING':
        className = classes?.cloneResettingStatus
        break
      case 'FATAL':
        className = classes?.cloneFatalStatus
        break
      default:
        break
    }

    if (onlyText && showDescription) {
      return (
        <span>
          <span style={{ display: 'block' }} className={className}>
            <Brightness1Icon className={className} />
            &nbsp;
            {Format.formatStatus(clone.status.code)}
          </span>
          <span style={{ display: 'block' }}>
            {clone.status.message && clone.status.message.length > 100 ? (
              <Tooltip title={''} classes={{ tooltip: classes?.toolTip }}>
                <span>{Format.limitStr(clone.status.message, 100)}</span>
              </Tooltip>
            ) : (
              clone.status.message
            )}
          </span>
        </span>
      )
    }

    if (onlyText && !showDescription) {
      return (
        <span className={className} title={clone.status.message}>
          <Tooltip
            title={clone.status.message}
            classes={{ tooltip: classes?.toolTip }}
          >
            <Brightness1Icon className={className} />
          </Tooltip>
          &nbsp;
          {Format.formatStatus(clone.status.code)}
        </span>
      )
    }

    return (
      <Tooltip
        title={clone.status.message}
        classes={{ tooltip: classes?.toolTip }}
      >
        <Brightness1Icon className={className} />
      </Tooltip>
    )
  }

  getInstanceStatus = (instance: DbLabStatusInstance, onlyText: boolean) => {
    const { classes } = this.props
    let className = classes?.instanceReadyStatus

    if (!instance.state) {
      return null
    }

    if (!instance.state.status) {
      return null
    }
    switch (instance.state.status.code) {
      case 'OK':
        className = classes?.instanceReadyStatus
        break
      case 'WARNING':
        className = classes?.instanceWarningStatus
        break
      case 'NO_RESPONSE':
        className = classes?.instanceNoResponseStatus
        break
      default:
        break
    }

    if (onlyText) {
      return (
        <span className={className} title={instance.state.status.message}>
          <Tooltip
            title={instance.state.status.message}
            classes={{ tooltip: classes?.toolTip }}
          >
            <Brightness1Icon className={className} />
          </Tooltip>
          &nbsp;
          {Format.formatStatus(instance.state.status.code)}
        </span>
      )
    }

    return (
      <Tooltip
        title={instance.state.status.message}
        classes={{ tooltip: classes?.toolTip }}
      >
        <Brightness1Icon className={className} />
      </Tooltip>
    )
  }

  getSessionStatus = (session: { status: string }) => {
    const { classes } = this.props
    let icon = null
    let className = null
    let label = session.status
    if (session.status.length) {
      label = session.status.charAt(0).toUpperCase() + session.status.slice(1)
    }

    switch (session.status) {
      case 'passed':
        icon = icons.okIconWhite
        className = classes?.sessionPassedStatus
        break
      case 'failed':
        icon = icons.failedIconWhite
        className = classes?.sessionFailedStatus
        break
      default:
        icon = icons.processingIconWhite
        className = classes?.sessionProcessingStatus
    }

    return (
      <div className={className}>
        <span style={{ whiteSpace: 'nowrap' }}>
          {icon}
          {label}
        </span>
      </div>
    )
  }

  render() {
    const { onlyText, showDescription, instance, clone, session } = this.props

    if (clone) {
      return this.getCloneStatus(
        clone,
        onlyText as boolean,
        showDescription as boolean,
      )
    }

    if (instance) {
      return this.getInstanceStatus(instance, onlyText as boolean)
    }

    if (session) {
      return this.getSessionStatus(session)
    }

    return null
  }
}

export default DbLabStatus
