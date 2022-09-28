/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import AppBar from '@material-ui/core/AppBar';
import Button from '@material-ui/core/Button';
import CloseIcon from '@material-ui/icons/Close';
import Dialog from '@material-ui/core/Dialog';
import IconButton from '@material-ui/core/IconButton';
import PropTypes from 'prop-types';
import React, { Component } from 'react';
import Slide from '@material-ui/core/Slide';
import Store from '../stores/store';
import TextField from '@material-ui/core/TextField';
import Toolbar from '@material-ui/core/Toolbar';
import Typography from '@material-ui/core/Typography';
import { withStyles } from '@material-ui/core/styles';

import { styles } from '@postgres.ai/shared/styles/styles';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

import Actions from '../actions/actions';
import ConsoleBreadcrumbs from './ConsoleBreadcrumbs';
import ConsolePageTitle from './ConsolePageTitle';
import FlameGraph from './FlameGraph';
import explainSamples from '../assets/explainSamples';
import visualizeTypes from '../assets/visualizeTypes';


const getStyles = theme => ({
  root: {
    width: '100%',
    [theme.breakpoints.down('sm')]: {
      maxWidth: 'calc(100vw - 40px)'
    },
    [theme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 220px)'
    },
    [theme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 220px)'
    },
    minHeight: '100%',
    zIndex: 1,
    position: 'relative'
  },
  pointerLink: {
    cursor: 'pointer'
  },
  breadcrumbPaper: {
    marginBottom: 15
  },
  planTextField: {
    '& .MuiInputBase-root': {
      width: '100%'
    },
    'display': 'block',
    'width': '100%'
  },
  appBar: {
    position: 'relative'
  },
  title: {
    marginLeft: theme.spacing(2),
    flex: 1,
    fontSize: '16px'
  },
  visFrame: {
    height: '100%'
  },
  nextButton: {
    marginLeft: '10px'
  },
  flameGraphContainer: {
    padding: '20px'
  }
});

const FullScreenDialogTransition = React.forwardRef(function Transition(props, ref) {
  return <Slide direction='up' ref={ref} {...props} />;
});

class ExplainVisualization extends Component {
  componentDidMount() {
    const that = this;

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  handleChange = event => {
    let id = event.target.id;
    let value = event.target.value;

    this.setState({
      [id]: value
    });
  };

  insertSample = () => {
    this.setState({ plan: explainSamples[0].value });
  };

  getExternalVisualization = () => {
    return this.state && this.state.data &&
      this.state.data.externalVisualization ? this.state.data.externalVisualization : null;
  };

  showExternalVisualization = (type) => {
    const { plan } = this.state;

    if (!plan) {
      return;
    }

    Actions.getExternalVisualizationData(type, plan, '');
  };

  closeExternalVisualization = () => {
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
    this.setState({
      showFlameGraph: true
    });
  };

  render() {
    const { classes } = this.props;

    const breadcrumbs = (
      <ConsoleBreadcrumbs
        {...this.props}
        breadcrumbs={[
          { name: 'SQL Optimization' },
          { name: 'Plan visualization' }
        ]}
      />
    );

    const pageTitle = (
      <ConsolePageTitle
        title={ 'Plan visualization' }
        information={
          <React.Fragment>
            <p>
              Visualize explain plans gathered manually. Plans gathered with Joe
              will be automatically saved in Joe history and can be visualized in
              command page without copy-pasting of a plan.
            </p>
            <p>
              Currently only JSON format is supported.
            </p>
            <p>
              For better results, use: explain (analyze, costs, verbose, buffers, format json).
            </p>
          </React.Fragment>
        }
      />
    );

    if (!this.state || !this.state.data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <Spinner size='lg' className={classes.progress} />
        </div>
      );
    }

    const { plan, showFlameGraph } = this.state;

    const externalVisualization = this.getExternalVisualization();

    const disableVizButtons = !plan || showFlameGraph ||
      (externalVisualization && externalVisualization.isProcessing);
    const openVizDialog = showFlameGraph || (externalVisualization &&
      externalVisualization.url && externalVisualization.url.length > 0);

    return (
      <div className={ classes.root }>
        { breadcrumbs }

        { pageTitle }

        <div className={ classes.relativeFieldBlock }>
          <Button
            variant='outlined'
            className={ classes.button }
            onClick={ this.insertSample }>
            Use an example
          </Button>
        </div>

        <TextField
          id='plan'
          label='Plan with execution (JSON)'
          className={ classes.planTextField }
          autoFocus={ true }
          margin='normal'
          multiline
          rows='15'
          variant='outlined'
          value={ this.state.plan }
          onChange={ this.handleChange }
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel
          }}
        />

        <div>
          <Button
            variant='contained'
            color='primary'
            onClick={ this.handleExternalVisualizationClick(visualizeTypes.depesz) }
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
            className={ classes.nextButton }
            variant='contained'
            color='primary'
            onClick={ this.handleExternalVisualizationClick(visualizeTypes.pev2) }
            disabled={ disableVizButtons }
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

        <div>
          <Dialog
            fullScreen
            open={ openVizDialog }
            onClose={ this.handleDialogClose }
            TransitionComponent={ FullScreenDialogTransition }
          >
            <AppBar className={ classes.appBar }>
              <Toolbar>
                <Typography variant='h2' className={classes.title}>
                  Visualization
                </Typography>
                <IconButton
                  edge='end'
                  color='inherit'
                  onClick={ this.closeExternalVisualization }
                  aria-label='close'
                >
                  <CloseIcon />
                </IconButton>
              </Toolbar>
            </AppBar>

            { showFlameGraph &&
              <div className={ classes.flameGraphContainer }>
                <h4>Flame Graph (buffers):</h4>
                <FlameGraph
                  data={ plan }
                  type='buffers'
                  name='chart-b'
                />

                <h4>Flame Graph (timing):</h4>
                <FlameGraph
                  data={ plan }
                  type='time'
                  name='chart-t'
                />
              </div>
            }

            { externalVisualization.url &&
              <iframe
                id='iframeVisualization'
                title='Visualization'
                src={ externalVisualization.url }
                className={ classes.visFrame }
              />
            }
          </Dialog>
        </div>
      </div>
    );
  }
}

ExplainVisualization.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(ExplainVisualization);
