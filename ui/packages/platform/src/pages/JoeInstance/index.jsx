/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import {
  Button,
  TextField,
  MenuItem,
  ExpansionPanel,
  ExpansionPanelSummary,
  ExpansionPanelDetails,
  Typography
} from '@material-ui/core';
import ExpandMoreIcon from '@material-ui/icons/ExpandMore';

import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';
import { colors } from '@postgres.ai/shared/styles/colors';
import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import Store from 'stores/store';
import Actions from 'actions/actions';
import { ErrorWrapper } from 'components/Error/ErrorWrapper';
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper';
import ConsolePageTitle from 'components/ConsolePageTitle';

import { getSystemMessages } from './utils';
import { Messages } from './Messages';
import { Command } from './Command';

import './styles.scss';

const VERIFY_MESSAGES_TIMEOUT = 60 * 1000;
class JoeInstance extends Component {
  state = {
    command: '',
    channelId: null,
    artifacts: {}
  };

  componentDidMount() {
    const that = this;
    const instanceId = this.props.match.params.instanceId;
    const orgId = this.props.orgId;


    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const joeInstance = this.data && this.data.joeInstance &&
        this.getJoeInstance(instanceId) ? this.getJoeInstance(instanceId) : null;

      if (auth && auth.token && !joeInstance.isProcessing &&
        !that.state.data) {
        Actions.getJoeInstance(auth.token, orgId, 0, instanceId);
      }

      if (auth && auth.token && !joeInstance.isChannelsProcessing &&
        !joeInstance.isChannelsProcessed && joeInstance.isProcessed &&
        !joeInstance.channelsError && joeInstance.data && joeInstance.data.id) {
        Actions.getJoeInstanceChannels(auth.token, instanceId);
      }

      that.setState({
        data: this.data
      });

      if (!that.state.channelId && Object.keys(joeInstance.channels).length > 0) {
        that.setState({
          channelId: joeInstance.channels[Object.keys(joeInstance.channels)[0]].channelId
        }, () => {
          that.sendHelpCommand();
        });
      }
    });

    Actions.refresh();
  }

  sendHelpCommand = () => {
    const channelId = this.state.channelId;
    const instanceId = this.props.match.params.instanceId;
    const instance = this.state.data && this.state.data.joeInstance &&
      this.state.data.joeInstance.instances[instanceId] ?
      this.state.data.joeInstance.instances[instanceId] : null;
    const messages = instance && instance.messages &&
      instance.messages[channelId] ?
      instance.messages[channelId] : null;

    if (messages && Object.keys(messages).length > 0) {
      return;
    }

    this.sendCommand('help');
  };

  sendCommand = (cmd) => {
    const instanceId = this.props.match.params.instanceId;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const instance = this.state.data && this.state.data.joeInstance &&
      this.state.data.joeInstance.instances[instanceId] ?
      this.state.data.joeInstance.instances[instanceId] : null;

    // Send command.
    Actions.sendJoeInstanceCommand(
      auth.token, instanceId,
      this.state.channelId,
      cmd,
      instance.channels[this.state.channelId].sessionId
    );
  };

  handleChangeChannel = (event) => {
    this.setState({ channelId: event.target.value }, () => {
      this.sendHelpCommand();
    });
  };

  handleChangeCommand = (event) => {
    this.setState({
      command: event.target.value
    });
  };

  componentWillUnmount() {
    const instanceId = this.props.match.params.instanceId;

    this.unsubscribe();
    if (this.updateInterval) {
      clearInterval(this.updateInterval);
    }

    if (this.domObserver) {
      this.domObserver.disconnect();
    }

    Actions.closeJoeWebSocketConnection(instanceId);
  }

  onSend = (value) => {
    const instanceId = this.props.match.params.instanceId;
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const instance = this.state.data && this.state.data.joeInstance &&
      this.state.data.joeInstance.instances[instanceId] ?
      this.state.data.joeInstance.instances[instanceId] : null;

    this.sendCommand(value);

    if (!this.updateInterval) {
      this.updateInterval = setInterval(() => {
        Actions.getJoeInstanceMessages(
          auth.token,
          instanceId,
          this.state.channelId,
          instance.channels[this.state.channelId].sessionId
        );
      }, VERIFY_MESSAGES_TIMEOUT);
    }
  };

  loadMessageArtifacts = (event, messageId) => {
    const auth = this.state.data && this.state.data.auth ?
      this.state.data.auth : null;
    const channelId = this.state.channelId;
    const instanceId = this.props.match.params.instanceId;
    const instance = this.state.data && this.state.data.joeInstance &&
      this.state.data.joeInstance.instances[instanceId] ?
      this.state.data.joeInstance.instances[instanceId] : null;
    const messages = instance && instance.messages &&
      instance.messages[channelId] ?
      instance.messages[channelId] : null;

    if (messages && messages[messageId].artifacts &&
      messages[messageId].artifacts.files) {
      // Artifacts already loaded.
      return true;
    }

    Actions.getJoeMessageArtifacts(
      auth.token,
      instanceId,
      channelId,
      messageId
    );

    return true;
  };

  preformatJoeMessageStatus = (status) => {
    const { classes } = this.props;
    let icon;
    let text;
    let color;

    switch (status) {
    case 'error':
      icon = icons.failedIcon;
      text = 'Failed';
      color = colors.state.error;
      break;
    case 'ok':
      icon = icons.okIcon;
      text = 'Completed';
      color = colors.state.ok;
      break;
    case 'running':
      icon = icons.hourGlassIcon;
      text = 'Processing';
      color = colors.state.processing;
      break;
    default:
      return null;
    }

    return (
      <div className={classes.messageStatus} style={{ color: color }}>
        <span className={classes.messageStatusIcon}>{icon}</span> {text}
      </div>
    );
  };

  markdownLink = (linkProps, messageId) => {
    const { classes } = this.props;
    const { href, target, children } = linkProps;
    const instanceId = this.props.match.params.instanceId;
    const channelId = this.state.channelId;
    const instance = this.state.data && this.state.data.joeInstance &&
      this.state.data.joeInstance.instances[instanceId] ?
      this.state.data.joeInstance.instances[instanceId] : null;
    const messages = instance && instance.messages &&
      instance.messages[channelId] ?
      instance.messages[channelId] : null;
    let artifactId = false;
    let artifacts = messages[messageId] && messages[messageId].artifacts ?
      messages[messageId].artifacts : null;


    let match = href.match(/\/artifact\/(\d*)/);
    if (match && match.length > 1) {
      artifactId = match[1];
    }

    if (href.startsWith('/artifact/') && artifactId) {
      return (
        <div className={'artifact-container'}>
          <ExpansionPanel
            className={classes.messageArtifacts}
            expanded={this.state.artifacts[artifactId]}
            TransitionProps={{
              enter: false,
              exit: false,
              appear: false,
              mountOnEnter: false,
              timeout: 0
            }}
          >
            <ExpansionPanelSummary
              expandIcon={<ExpandMoreIcon/>}
              onClick={
                event => {
                  this.setState({ artifacts: { [artifactId]: !this.state.artifacts[artifactId] } });
                  this.loadMessageArtifacts(event, messages[messageId].id);
                }
              }
              aria-controls={'message-' + messageId + '-artifacts-content'}
              id={'message-' + messageId + '-artifacts-header'}
              className={classes.advancedExpansionPanelSummary}
            >
              <Typography className={classes.heading}>
                <nobr>{children}</nobr>
              </Typography>
            </ExpansionPanelSummary>
            <ExpansionPanelDetails
              className={classes.advancedExpansionPanelDetails}
            >
              {(!artifacts || (artifacts && artifacts.isProcessing)) ? (
                <Spinner />) : ''}
              {artifacts && artifacts.files ? (
                <div className={classes.messageArtifactsContainer}>
                  <div
                    key={artifacts.files[artifactId].id}
                    className={'artifact'}
                  >
                    <TextField
                      variant='outlined'
                      id={'artifact' + artifacts.files[artifactId].id}
                      multiline
                      fullWidth
                      value={artifacts.files[artifactId].content}
                      className={classes.code}
                      margin='normal'
                      InputProps={{
                        readOnly: true
                      }}
                    />
                  </div>
                </div>
              ) : null}
            </ExpansionPanelDetails>
          </ExpansionPanel>
        </div>
      );
    }

    return (
      <a className={classes.link} href={href} target={target}>{children}</a>
    );
  };

  render() {
    const { classes } = this.props;
    const instanceId = this.props.match.params.instanceId;
    const channelId = this.state.channelId;
    const instance = this.state.data && this.state.data.joeInstance &&
      this.state.data.joeInstance.instances[instanceId] ?
      this.state.data.joeInstance.instances[instanceId] : null;
    const messages = instance && instance.messages &&
      instance.messages[channelId] ?
      instance.messages[channelId] : null;

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        org={this.props.org}
        project={this.props.project}
        breadcrumbs={[
          { name: 'SQL Optimization', url: null },
          { name: 'Joe Instances', url: 'joe-instances' },
          { name: 'Instance #' + instanceId, url: null }
        ]}
      />
    );

    if (!this.state.data ||
      (instance && (instance.isChannelsProcessing || instance.isProcessing))) {
      return (
        <>
          {breadcrumbs}

          <PageSpinner />
        </>
      );
    }

    if (instance.error) {
      return (
        <>
          {breadcrumbs}

          <ErrorWrapper
            message={instance.errorMessage}
            code={instance.errorCode}
          />
        </>
      );
    }

    if (instance.channelsError) {
      return (
        <>
          {breadcrumbs}

          <ErrorWrapper
            message={instance.channelsErrorMessage}
          />
        </>
      );
    }

    return (
      <>
        {breadcrumbs}

        <ConsolePageTitle title='Ask Joe'/>

        { instance && instance.channelsErrorMessage ? (
          <div>
            <span className={classes.errorMsg}>{instance.channelsErrorMessage}</span>
          </div>
        ) : null }

        <div className={classes.actions}>
          <TextField
            value={channelId}
            onChange={event => this.handleChangeChannel(event)}
            select
            label='Project and database'
            inputProps={{
              name: 'channel',
              id: 'channel'
            }}
            InputLabelProps={{
              shrink: true,
              style: styles.inputFieldLabel
            }}
            FormHelperTextProps={{
              style: styles.inputFieldHelper
            }}
            variant='outlined'
            className={classes.channelsList}
          >
            {Object.keys(instance.channels).map(c => {
              return (
                <MenuItem
                  value={instance.channels[c].channelId}
                  key={instance.channels[c].channelId}
                >
                  {instance.channels[c].channelId}
                </MenuItem>
              );
            })}
          </TextField>
          <Button
            className={classes.clearChatButton}
            variant='outlined'
            disabled={!messages || (messages && messages.length === 0)}
            onClick={() => Actions.clearJoeInstanceChannelMessages(instanceId, channelId)}
          >
            <nobr>Clear chat</nobr>
          </Button>
        </div>

        <Messages
          classes={classes}
          messages={messages}
          markdownLink={this.markdownLink}
          sendCommand={this.sendCommand}
          loadMessageArtifacts={this.loadMessageArtifacts}
          preformatJoeMessageStatus={this.preformatJoeMessageStatus}
          systemMessages={getSystemMessages(instance, channelId)}
        />

        <Command
          isDisabled={instance && instance.channelsError}
          onSend={this.onSend}
        />
      </>
    );
  }
}

JoeInstance.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default JoeInstance
