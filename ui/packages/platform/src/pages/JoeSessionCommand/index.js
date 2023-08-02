/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import Dialog from '@material-ui/core/Dialog';
import Button from '@material-ui/core/Button';
import AppBar from '@material-ui/core/AppBar';
import Toolbar from '@material-ui/core/Toolbar';
import IconButton from '@material-ui/core/IconButton';
import Typography from '@material-ui/core/Typography';
import CloseIcon from '@material-ui/icons/Close';
import Slide from '@material-ui/core/Slide';
import Tabs from '@material-ui/core/Tabs';
import Tab from '@material-ui/core/Tab';
import { formatDistanceToNowStrict } from 'date-fns';

import { FormattedText } from '@postgres.ai/shared/components/FormattedText';
import { PageSpinner } from '@postgres.ai/shared/components/PageSpinner';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

import Store from 'stores/store';
import Actions from 'actions/actions';
import { ErrorWrapper } from 'components/Error/ErrorWrapper';
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper';
import ConsolePageTitle from 'components/ConsolePageTitle';
import FlameGraph from 'components/FlameGraph';
import { visualizeTypes } from 'assets/visualizeTypes';
import Urls from 'utils/urls';
import Permissions from 'utils/permissions';
import format from 'utils/format';

import { TabPanel } from './TabPanel';

const hashLinkVisualizePrefix = 'visualize-';

function a11yProps(index) {
  return {
    'id': `plan-tab-${index}`,
    'aria-controls': `plan-tabpanel-${index}`
  };
}

const FullScreenDialogTransition = React.forwardRef(function Transition(props, ref) {
  return <Slide direction='up' ref={ref} {...props} />;
});

class JoeSessionCommand extends Component {
  componentDidMount() {
    const that = this;
    const commandId = this.getCommandId();
    const orgId = this.props.orgId ? this.props.orgId : null;

    this.handleChangeTab(null, 0);

    this.unsubscribe = Store.listen(function () {
      const auth = this.data && this.data.auth ? this.data.auth : null;
      const command = this.data && this.data.command ?
        this.data.command : null;

      that.setState({ data: this.data });
      if (auth && auth.token && !command.isProcessing && !command.error &&
        !that.state && orgId) {
        Actions.getJoeSessionCommand(auth.token, orgId, commandId);
      }

      if (!Urls.isSharedUrl() && !this.data.shareUrl.isProcessing &&
        !this.data.sharedUrls['command'][commandId]) {
        Actions.getSharedUrl(this.data.auth.token, orgId, 'command', commandId);
      }

      setTimeout(()=> {
        const commandData = command ? command.data : null;
        if (!!commandData && !this.hashLocationProcessed) {
          this.hashLocationProcessed = true;
          that.handleHashLocation();
        }
      }, 10);
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleHashLocation() {
    const hash = this.props.history.location.hash;
    if (!hash || !hash.includes(hashLinkVisualizePrefix)) {
      return;
    }

    const type = hash.replace('#', '').replace(hashLinkVisualizePrefix, '');

    if (type === visualizeTypes.flame) {
      this.showFlameGraphVisualization();
      return;
    }

    this.showExternalVisualization(type);
  }

  setHashUrl(hashUrl) {
    this.props.history.replace(this.props.history.location.pathname + hashUrl);
  }

  removeHashUrl() {
    this.props.history.replace(this.props.history.location.pathname);
  }

  getCommandId = () => {
    if (this.props.match && this.props.match.params && this.props.match.params.commandId) {
      return parseInt(this.props.match.params.commandId, 10);
    }

    return parseInt(this.props.commandId, 10);
  };

  getCommandData = () => {
    const commandId = this.getCommandId();
    const data = this.state && this.state.data && this.state.data.command &&
      this.state.data.command.data ? this.state.data.command.data : null;

    return data && data.commandId === commandId ? data : null;
  };

  getExternalVisualization = () => {
    return this.state && this.state.data &&
      this.state.data.externalVisualization ? this.state.data.externalVisualization : null;
  };

  isExplain = () => {
    const data = this.getCommandData();
    return !!data && data.command === 'explain';
  };

  showExternalVisualization = (type) => {
    const data = this.getCommandData();

    if (!this.isExplain() || data.planExecJson.length === 0) {
      return;
    }

    this.setHashUrl('#' + hashLinkVisualizePrefix + type);

    Actions.getExternalVisualizationData(
      type,
      {
        json: data.planExecJson,
        text: data.planExecText
      },
      data.query
    );
  };

  closeExternalVisualization = () => {
    this.props.history.replace(this.getCommandId());
    Actions.closeExternalVisualization();
    this.setState({
      showFlameGraph: false
    });
  };

  handleExternalVisualizationClick = (type) => {
    return () => {
      this.showExternalVisualization(type);
    };
  };

  showFlameGraphVisualization = () => {
    this.setHashUrl('#' + hashLinkVisualizePrefix + visualizeTypes.flame);
    this.setState({
      showFlameGraph: true
    });
  };

  handleChangeTab = (event, tabValue) => {
    this.setState({ tab: tabValue });
  };

  showShareDialog = () => {
    const commandId = this.getCommandId();
    const orgId = this.props.orgId ? this.props.orgId : null;

    if (!orgId) {
      return;
    }

    Actions.showShareUrlDialog(
      parseInt(orgId, 10),
      'command',
      commandId,
      'Anyone on the internet with the special link can view query, plan and ' +
      'all parameters. Check that there is no sensitive data.'
    );
  };

  render() {
    const { classes } = this.props;
    const data = this.getCommandData();
    const commandId = this.getCommandId();
    const sessionId = this.props.sessionId || this.props.match.params.sessionId;
    const orgId = this.props.orgId ? this.props.orgId : null;
    let allowShare = false;

    if (this.props.orgData && orgId) {
      let permissions = Permissions.getPermissions(this.props.orgData);
      allowShare = permissions.shareUrl;
    }

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[
          { name: 'SQL Optimization', url: null },
          { name: 'History', url: 'sessions' },
          { name: 'Command #' + commandId },
        ]}
      />
    )

    if (this.state && this.state.data && this.state.data.command.error) {
      return (
        <div className={classes.root}>
          {breadcrumbs}
          <ErrorWrapper
            message={this.state.data.command.errorMessage}
            code={this.state.data.command.errorCode}
          />
        </div>
      );
    }

    if (!data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          <PageSpinner />
        </div>
      );
    }

    let username = 'Unknown';
    if (data.slackUsername && data.slackUid) {
      username = `${data.slackUsername} (${data.slackUid})`;
    } else if (data.username) {
      username = data.username;
    } else if (data.useremail) {
      username = data.useremail;
    }

    const externalVisualization = this.getExternalVisualization();

    const showFlameGraph = this.state.showFlameGraph;
    const disableVizButtons = showFlameGraph ||
      (externalVisualization && externalVisualization.isProcessing);
    const openVizDialog = showFlameGraph || (externalVisualization &&
      externalVisualization.url && externalVisualization.url.length > 0);
    const title = `Command #${commandId} (${data.command}) from session #${sessionId}`;
    const recommends = data.recommends;

    let shareUrlButton = (
      <Button
        variant='contained'
        color='primary'
        key='add_dblab_instance'
        onClick={ this.showShareDialog }
      >
        Share
        {(this.state.data.shareUrl && this.state.data.shareUrl.isProcessing) &&
          <span>
            &nbsp;
            <Spinner size='sm' />
          </span>
        }
      </Button>
    );

    let titleActions = [];
    if (!Urls.isSharedUrl() && allowShare) {
      titleActions.push(shareUrlButton);
    }

    let isShared = this.state.data.sharedUrls && this.state.data.sharedUrls['command'] &&
      this.state.data.sharedUrls['command'][commandId] &&
      this.state.data.sharedUrls['command'][commandId].uuid;

    return (
      <div className={classes.root}>
        { breadcrumbs }

        <ConsolePageTitle
          title={ title }
          label={ isShared ? 'Shared via link' : null}
          actions={ titleActions }
        />

        { this.isExplain() &&
          <div className={classes.actions}>
            <Button
              variant='contained'
              color='primary'
              onClick={this.handleExternalVisualizationClick(visualizeTypes.depesz)}
              disabled={ disableVizButtons }
            >
              Explain Depesz

              {
                disableVizButtons &&
                externalVisualization.type === visualizeTypes.depesz &&
                <span>
                  &nbsp;
                  <Spinner size='sm' />
                </span>
              }
            </Button>

            <Button
              className={classes.nextButton}
              variant='contained'
              color='primary'
              onClick={this.handleExternalVisualizationClick(visualizeTypes.pev2)}
              disabled={ disableVizButtons}
            >
              Explain PEV2

              {
                disableVizButtons &&
                externalVisualization.type === visualizeTypes.pev2 &&
                <span>
                  &nbsp;
                  <Spinner size='sm' />
                </span>
              }
            </Button>

            <Button
              className={ classes.nextButton }
              variant='contained'
              color='primary'
              onClick={ this.showFlameGraphVisualization }
              disabled={ disableVizButtons }
            >
              Explain FlameGraph
            </Button>
          </div>
        }

        <div>
          <h4>Author:</h4>
          <p>
            { username }
          </p>
        </div>

        <h4>Command:</h4>
        <FormattedText value={data.command} />

        { data.query && (
          <React.Fragment>
            <h4>Query:</h4>
            <FormattedText value={data.query} />
          </React.Fragment>
        )}

        {this.isExplain() ? (
          <div>
            <h4>Plan:</h4>
            <Tabs
              value={this.state.tab}
              onChange={this.handleChangeTab}
              aria-label='Plan'
              inputProps={{
                name: 'tab',
                id: 'tab'
              }}
            >
              <Tab label='with execution' {...a11yProps(0)} />
              <Tab label='with execution (JSON)' {...a11yProps(0)} />
              <Tab label='w/o execution' {...a11yProps(1)} />
            </Tabs>

            <TabPanel value={this.state.tab} index={0}>
              <FormattedText value={data.planExecText} />
            </TabPanel>

            <TabPanel value={this.state.tab} index={1}>
              <FormattedText value={data.planExecJson} />
            </TabPanel>

            <TabPanel value={this.state.tab} index={2}>
              <FormattedText value={data.planText} />
            </TabPanel>

            <h4>Recommendations:</h4>
            <FormattedText value={recommends} />

            <h4>Statistics:</h4>
            <FormattedText value={data.stats.trim()} />

            <h4>Query locks:</h4>
            <FormattedText value={data.queryLocks} />

            <h4>Details:</h4>
            <Typography component='p'>
              <span>Uploaded</span>:&nbsp;{
                formatDistanceToNowStrict(new Date(data.createdAt), { addSuffix: true })
              }&nbsp;
              ({ format.formatTimestampUtc(data.createdAt) })
            </Typography>
          </div>
        ) : (
          <div>
            <h4>Response:</h4>
            <FormattedText value={data.response} />
          </div>
        )}

        { data.error && data.error.length > 0 &&
          <FormattedText value={data.error} />
        }

        <div>
          <Dialog
            fullScreen
            open={ openVizDialog }
            onClose={this.handleDialogClose}
            TransitionComponent={FullScreenDialogTransition}
          >
            <AppBar className={classes.appBar}>
              <Toolbar>
                <Typography variant='h2' className={classes.title}>
                  Visualization
                </Typography>
                <IconButton
                  edge='end'
                  color='inherit'
                  onClick={this.closeExternalVisualization}
                  aria-label='close'
                >
                  <CloseIcon />
                </IconButton>
              </Toolbar>
            </AppBar>

            { showFlameGraph &&
              <div className={classes.flameGraphContainer}>
                <h4>Flame Graph (buffers):</h4>
                <FlameGraph
                  data={data.planExecJson}
                  type='buffers'
                  name='chart-b'
                />

                <h4>Flame Graph (timing):</h4>
                <FlameGraph
                  data={data.planExecJson}
                  type='time'
                  name='chart-t'
                />
              </div>
            }

            { externalVisualization.url &&
              <iframe
                id='iframeVisualization'
                title='Visualization'
                src={externalVisualization.url}
                className={classes.visFrame}
              />
            }
          </Dialog>
        </div>

        <div className={classes.bottomSpace}/>
      </div>
    );
  }
}

JoeSessionCommand.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default JoeSessionCommand
