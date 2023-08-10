/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {
  AppBar,
  Dialog,
  Button,
  IconButton,
  TextField,
  Toolbar,
  Typography,
} from '@material-ui/core'
import CloseIcon from '@material-ui/icons/Close'
import React, { Component } from 'react'
import Store from '../../stores/store'

import { styles } from '@postgres.ai/shared/styles/styles'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Actions from '../../actions/actions'
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper'

import ConsolePageTitle from '../ConsolePageTitle'
import FlameGraph from '../FlameGraph'
import explainSamples from '../../assets/explainSamples'
import { visualizeTypes } from '../../assets/visualizeTypes'

interface ExplainVisualizationProps {
  classes: ClassesType
}

interface ExplainVisualizationState {
  plan: string | null
  showFlameGraph: boolean
  data: {
    externalVisualization: {
      url: string
      type: string
      isProcessing: boolean
    }
  } | null
}

class ExplainVisualization extends Component<
  ExplainVisualizationProps,
  ExplainVisualizationState
> {
  unsubscribe: Function
  componentDidMount() {
    const that = this

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      that.setState({ data: this.data })
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  handleChange = (event: React.ChangeEvent<HTMLTextAreaElement>) => {
    this.setState({
      plan: event.target.value,
    })
  }

  insertSample = () => {
    this.setState({ plan: explainSamples[0].value })
  }

  getExternalVisualization = () => {
    return this.state &&
      this.state.data &&
      this.state.data.externalVisualization
      ? this.state.data.externalVisualization
      : null
  }

  showExternalVisualization = (type: string) => {
    const { plan } = this.state

    if (!plan) {
      return
    }

    Actions.getExternalVisualizationData(type, plan, '')
  }

  closeExternalVisualization = () => {
    Actions.closeExternalVisualization()
    this.setState({
      showFlameGraph: false,
    })
  }

  handleExternalVisualizationClick = (type: string) => {
    return () => {
      this.showExternalVisualization(type)
    }
  }

  showFlameGraphVisualization = () => {
    this.setState({
      showFlameGraph: true,
    })
  }

  render() {
    const { classes } = this.props

    const breadcrumbs = (
      <ConsoleBreadcrumbsWrapper
        {...this.props}
        breadcrumbs={[
          { name: 'SQL Optimization' },
          { name: 'Plan visualization' },
        ]}
      />
    )

    const pageTitle = (
      <ConsolePageTitle
        title={'Plan visualization'}
        information={
          <React.Fragment>
            <p>
              Visualize explain plans gathered manually. Plans gathered with Joe
              will be automatically saved in Joe history and can be visualized
              in command page without copy-pasting of a plan.
            </p>
            <p>Currently only JSON format is supported.</p>
            <p>
              For better results, use: explain (analyze, costs, verbose,
              buffers, format json).
            </p>
          </React.Fragment>
        }
      />
    )

    if (!this.state || !this.state.data) {
      return (
        <div className={classes.root}>
          {breadcrumbs}

          {pageTitle}

          <Spinner size="lg" className={classes.progress} />
        </div>
      )
    }

    const { plan, showFlameGraph } = this.state

    const externalVisualization = this.getExternalVisualization()

    const disableVizButtons =
      !plan || showFlameGraph || externalVisualization?.isProcessing
    const openVizDialog =
      showFlameGraph ||
      (externalVisualization?.url
        ? externalVisualization.url?.length > 0
        : false)

    return (
      <div className={classes.root}>
        {breadcrumbs}

        {pageTitle}

        <div className={classes.relativeFieldBlock}>
          <Button
            variant="outlined"
            className={classes.button}
            onClick={this.insertSample}
          >
            Use an example
          </Button>
        </div>

        <TextField
          id="plan"
          label="Plan with execution (JSON)"
          className={classes.planTextField}
          autoFocus={true}
          margin="normal"
          multiline
          rows="15"
          variant="outlined"
          value={this.state.plan}
          onChange={this.handleChange}
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel,
          }}
        />

        <div>
          <Button
            variant="contained"
            color="primary"
            onClick={this.handleExternalVisualizationClick(
              visualizeTypes.depesz,
            )}
            disabled={disableVizButtons}
          >
            Explain Depesz
            {disableVizButtons &&
              externalVisualization?.type === visualizeTypes.depesz && (
                <span>
                  &nbsp;
                  <Spinner size="sm" />
                </span>
              )}
          </Button>

          <Button
            className={classes.nextButton}
            variant="contained"
            color="primary"
            onClick={this.handleExternalVisualizationClick(visualizeTypes.pev2)}
            disabled={disableVizButtons}
          >
            Explain PEV2
            {disableVizButtons &&
              externalVisualization?.type === visualizeTypes.pev2 && (
                <span>
                  &nbsp;
                  <Spinner size="sm" />
                </span>
              )}
          </Button>

          <Button
            className={classes.nextButton}
            variant="contained"
            color="primary"
            onClick={this.showFlameGraphVisualization}
            disabled={disableVizButtons}
          >
            Explain FlameGraph
          </Button>
        </div>

        <div>
          <Dialog fullScreen open={openVizDialog}>
            <AppBar className={classes.appBar}>
              <Toolbar>
                <Typography variant="h2" className={classes.title}>
                  Visualization
                </Typography>
                <IconButton
                  edge="end"
                  color="inherit"
                  onClick={this.closeExternalVisualization}
                  aria-label="close"
                >
                  <CloseIcon />
                </IconButton>
              </Toolbar>
            </AppBar>

            {showFlameGraph && (
              <div className={classes.flameGraphContainer}>
                <h4>Flame Graph (buffers):</h4>
                <FlameGraph data={plan} type="buffers" name="chart-b" />

                <h4>Flame Graph (timing):</h4>
                <FlameGraph data={plan} type="time" name="chart-t" />
              </div>
            )}

            {externalVisualization?.url && (
              <iframe
                id="iframeVisualization"
                title="Visualization"
                src={externalVisualization?.url}
                className={classes.visFrame}
              />
            )}
          </Dialog>
        </div>
      </div>
    )
  }
}

export default ExplainVisualization
