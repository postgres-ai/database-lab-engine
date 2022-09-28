/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Brightness1Icon from '@material-ui/icons/Brightness1';
import Tooltip from '@material-ui/core/Tooltip';

import { colors } from '@postgres.ai/shared/styles/colors';
import { icons } from '@postgres.ai/shared/styles/icons';

import Format from '../utils/format';

const styles = () => ({
  cloneReadyStatus: {
    color: colors.state.ok,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  cloneCreatingStatus: {
    color: colors.state.processing,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  cloneResettingStatus: {
    color: colors.state.processing,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  cloneDeletingStatus: {
    color: colors.state.warning,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  cloneFatalStatus: {
    color: colors.state.error,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }

  },
  instanceReadyStatus: {
    color: colors.state.ok,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  instanceWarningStatus: {
    color: colors.state.warning,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  instanceNoResponseStatus: {
    color: colors.state.error,
    fontSize: '1.1em',
    verticalAlign: 'middle',
    ['& svg']: {
      marginTop: '-3px'
    }
  },
  toolTip: {
    fontSize: '10px!important'
  },
  sessionPassedStatus: {
    'display': 'inline-block',
    'border': '1px solid ' + colors.state.ok,
    'fontSize': '12px',
    'color': '#FFFFFF',
    'backgroundColor': colors.state.ok,
    'padding': '3px',
    'paddingLeft': '5px',
    'paddingRight': '5px',
    'borderRadius': 3,
    'lineHeight': '14px',
    '& svg': {
      width: 10,
      height: 10,
      marginBottom: '-1px',
      marginRight: '5px'
    }
  },
  sessionFailedStatus: {
    'display': 'inline-block',
    'border': '1px solid ' + colors.state.error,
    'fontSize': '12px',
    'color': '#FFFFFF',
    'backgroundColor': colors.state.error,
    'padding': '3px',
    'paddingLeft': '5px',
    'paddingRight': '5px',
    'borderRadius': 3,
    'lineHeight': '14px',
    '& svg': {
      width: 10,
      height: 10,
      marginBottom: '-1px',
      marginRight: '5px'
    }
  },
  sessionProcessingStatus: {
    'display': 'inline-block',
    'border': '1px solid ' + colors.state.processing,
    'fontSize': '12px',
    'color': '#FFFFFF',
    'backgroundColor': colors.state.processing,
    'padding': '3px',
    'paddingLeft': '5px',
    'paddingRight': '5px',
    'borderRadius': 3,
    'lineHeight': '14px',
    '& svg': {
      width: 10,
      height: 10,
      marginBottom: '-1px',
      marginRight: '5px'
    }
  }
});

class DbLabStatus extends Component {
  getCloneStatus = (clone, onlyText, showDescription) => {
    const { classes } = this.props;
    let className = classes.cloneReadyStatus;

    if (!clone.status) {
      return null;
    }

    switch (clone.status.code) {
    case 'OK':
      className = classes.cloneReadyStatus;
      break;
    case 'CREATING':
      className = classes.cloneCreatingStatus;
      break;
    case 'DELETING':
      className = classes.cloneDeletingStatus;
      break;
    case 'RESETTING':
      className = classes.cloneResettingStatus;
      break;
    case 'FATAL':
      className = classes.cloneFatalStatus;
      break;
    default:
      break;
    }

    if (onlyText && showDescription) {
      return (
        <span
        >
          <span style={{ display: 'block' }} className={className}>
            <Brightness1Icon className={className}/>
            &nbsp;
            {Format.formatStatus(clone.status.code)}
          </span>
          <span style={{ display: 'block' }}>
            {clone.status.message && clone.status.message.length > 100 ? (
              <Tooltip
                classes={{ tooltip: classes.toolTip }}
              >
                <span>{Format.limitStr(clone.status.message, 100)}</span>
              </Tooltip>
            ) : clone.status.message}
          </span>
        </span>
      );
    }

    if (onlyText && !showDescription) {
      return (
        <span
          className={className}
          title={clone.status.message}
        >
          <Tooltip
            title={clone.status.message}
            classes={{ tooltip: classes.toolTip }}
          >
            <Brightness1Icon className={className}/>
          </Tooltip>
          &nbsp;
          {Format.formatStatus(clone.status.code)}
        </span>
      );
    }

    return (
      <Tooltip
        title={clone.status.message}
        classes={{ tooltip: classes.toolTip }}
      >
        <Brightness1Icon className={className}/>
      </Tooltip>
    );
  };

  getInstanceStatus = (instance, onlyText) => {
    const { classes } = this.props;
    let className = classes.instanceReadyStatus;

    if (!instance.state) {
      return null;
    }

    if (!instance.state.status) {
      return null;
    }
    switch (instance.state.status.code) {
    case 'OK':
      className = classes.instanceReadyStatus;
      break;
    case 'WARNING':
      className = classes.instanceWarningStatus;
      break;
    case 'NO_RESPONSE':
      className = classes.instanceNoResponseStatus;
      break;
    default:
      break;
    }

    if (onlyText) {
      return (
        <span
          className={className}
          title={instance.state.status.message}
        >
          <Tooltip
            title={instance.state.status.message}
            classes={{ tooltip: classes.toolTip }}
          >
            <Brightness1Icon className={className}/>
          </Tooltip>
          &nbsp;
          {Format.formatStatus(instance.state.status.code)}
        </span>
      );
    }

    return (
      <Tooltip
        title={instance.state.status.message}
        classes={{ tooltip: classes.toolTip }}
      >
        <Brightness1Icon className={className}/>
      </Tooltip>
    );
  };

  getSessionStatus = (session) => {
    const { classes } = this.props;
    let icon = null;
    let className = null;
    let label = session.status;
    if (session.status.length) {
      label = session.status.charAt(0).toUpperCase() + session.status.slice(1);
    }

    switch (session.status) {
    case 'passed':
      icon = icons.okIconWhite;
      className = classes.sessionPassedStatus;
      break;
    case 'failed':
      icon = icons.failedIconWhite;
      className = classes.sessionFailedStatus;
      break;
    default:
      icon = icons.processingIconWhite;
      className = classes.sessionProcessingStatus;
    }

    return (
      <div className={className}>
        <nobr>{icon}{label}</nobr>
      </div>
    );
  };

  render() {
    const onlyText = this.props.onlyText;
    const showDescription = this.props.showDescription;
    const instance = this.props.instance;
    const clone = this.props.clone;
    const session = this.props.session;

    if (clone) {
      return this.getCloneStatus(clone, onlyText, showDescription);
    }

    if (instance) {
      return this.getInstanceStatus(instance, onlyText);
    }

    if (session) {
      return this.getSessionStatus(session);
    }

    return null;
  }
}

DbLabStatus.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(DbLabStatus);
