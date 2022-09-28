/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import * as d3Lib from 'd3';
import * as d3Flamegraph from 'd3-flame-graph';
import React, { Component } from 'react';
import { withStyles } from '@material-ui/core/styles';


const d3 = Object.assign({}, d3Lib, d3Flamegraph);

const styles = theme => ({
  root: {
    width: '100%',
    minHeight: '100%',
    zIndex: 1,
    position: 'relative',
    [theme.breakpoints.down('sm')]: {
      maxWidth: '100vw'
    },
    [theme.breakpoints.up('md')]: {
      maxWidth: 'calc(100vw - 200px)'
    },
    [theme.breakpoints.up('lg')]: {
      maxWidth: 'calc(100vw - 200px)'
    }
  },
  pointerLink: {
    cursor: 'pointer'
  },
  breadcrumbPaper: {
    marginBottom: 15
  }
});

const fgStyles = (
  <div>
    <style>{`
        .flamegraph-hidden {
            display: none;
        }

        .flamegraph{
            margin-bottom: 40px;
        }

        .d3-flame-graph rect{
            stroke: #EEE;
            fill-opacity: .8;
        }

        .d3-flame-graph rect:hover{
            stroke: #474747;
            stroke-width: .5;
            cursor: pointer;
        }

        .d3-flame-graph .label{
            pointer-events: none;
            white-space: nowrap;
            text-overflow: ellipsis;
            overflow: hidden;
            font-size: 12px;
            font-family: Verdana;
            margin-left: 4px;
            margin-right: 4px;
            line-height: 1.5;
            padding: 0;
            font-weight: 400;
            color: #000;
            text-align: left;
        }

        .d3-flame-graph .fade{
            opacity: .6!important;
        }

        .d3-flame-graph .title{
            font-size: 20px;
            font-family: Verdana;
        }

        .d3-flame-graph-tip{
            z-index:  1000;
            line-height: 1;
            font-family: Verdana;
            font-size: 12px;
            padding: 12px;
            background: rgba(0,0,0,.8);
            color: #fff;
            border-radius: 2px;
            pointer-events: none;
        }

        .d3-flame-graph-tip: after{
            box-sizing: border-box;
            display: inline;
            font-size: 10px;
            width: 100%;
            line-height: 1;
            color: rgba(0,0,0,.8);
            position: absolute;
            pointer-events: none;
        }

        .d3-flame-graph-tip.n: after{
            content: "\\25BC";
            margin: -1px 0 0;
            top: 100%;
            left: 0;
            text-align: center;
        }

        .d3-flame-graph-tip.e: after{
            content: "\\25C0";
            margin: -4px 0 0;
            top: 50%;
            left: -8px;
        }

        .d3-flame-graph-tip.s: after{
            content: "\\25B2";
            margin: 0 0 1px;
            top: -8px;
            left: 0;
            text-align: center;
        }

        .d3-flame-graph-tip.w: after{
            content: "\\25B6";
            margin: -4px 0 0 -1px;
            top: 50%;
            left: 100%;
        }
    `}</style>
  </div>
);

const TYPE_TIME = 'time';
const TYPE_BUFFERS = 'buffers';

class FlameGraph extends Component {
  constructor(props) {
    super(props);
    this.state = { noData: true };
  }

  componentDidMount() {
    this.update();
  }

  componentDidUpdate(prevProps, prevState) {
    this.update(prevProps, prevState);
  }

  render() {
    return (
      <div>
        {fgStyles}

        {this.state.noData && <h5>No data</h5>}

        <div
          id={this.props.name}
          className={'flamegraph' +
            (this.state.noData ? ' flamegraph-hidden' : '')}
        />
      </div>
    );
  }

  update = (prevProps, prevState) => {
    let explainJson;

    try {
      explainJson = JSON.parse(this.props.data);
    } catch (err) {
      this.setNoData(prevState, true);
      return;
    }

    let type = TYPE_TIME;
    if (this.props.type === TYPE_BUFFERS) {
      type = TYPE_BUFFERS;
    }

    const data = this.convert(explainJson, type);
    if (!data) {
      this.setNoData(prevState, true);
      return;
    }

    this.setNoData(prevState, false);

    this.draw(data);
  };

  setNoData = (prevState, noData) => {
    if (!prevState) {
      this.setState({ noData });
      return;
    }

    if (prevState.noData !== noData) {
      this.setState({ noData });
    }
  };

  convert = (explainJson, type) => {
    if (!explainJson || !Array.isArray(explainJson) ||
      explainJson.length < 1 || !explainJson[0].Plan) {
      return null;
    }

    const data = this.convertRec(explainJson[0].Plan, type);
    return data;
  };

  convertRec = (plan, type) => {
    let children = [];

    if (!!plan['Plans']) {
      for (let i = 0; i < plan['Plans'].length; i++) {
        children.push(this.convertRec(plan['Plans'][i], type));
      }
    }

    return this.newNode(plan, type, children);
  };

  newNode = (plan, type, children) => {
    const buffersHit = plan['Shared Hit Blocks'];
    const buffersRead = plan['Shared Read Blocks'];
    const timing = plan['Actual Total Time'];

    let name = this.buildNodeName(plan);
    let tooltip = name;
    let value;

    if (type === TYPE_BUFFERS) {
      value = buffersHit + buffersRead;
      name = name + ', ' + value;
      tooltip = tooltip + ', Buffers hit: ' + buffersHit + ', Buffers read: ' +
        buffersRead;
    } else {
      value = timing;
      name = name + ', ' + value + ' ms';
      tooltip = tooltip + ', Execution time: ' + value + ' ms';
    }

    return {
      name,
      value,
      tooltip,
      buffersHit,
      buffersRead,
      timing,
      children
    };
  };

  buildNodeName = (plan) => {
    const nodeType = plan['Node Type'];
    let name = nodeType;

    if (nodeType === 'Modify Table') {
      // E.g. for Insert.
      name = plan['Operation'];
    }

    if (nodeType === 'Hash Join' && plan['Join Type'] === 'Left') {
      name = 'Hash Left Join';
    }

    if (nodeType === 'Aggregate' && plan['Strategy'] === 'Hashed') {
      name = 'Hash Aggregate';
    }

    if (nodeType === 'Nested Loop' && plan['Join Type'] === 'Left') {
      name = 'Nested Loop Left Join';
    }

    if (plan['Parallel Aware']) {
      name = 'Parallel ' + name;
    }

    return name;
  };

  draw = (data) => {
    d3.select('#' + this.props.name).selectAll('*').remove();

    let fg = d3.flamegraph()
      .width(600)
      .cellHeight(18)
      .transitionDuration(750)
      .transitionEase(d3.easeCubic)
      .label((d) => {
        return d.data && d.data.tooltip;
      });

    d3.select('#' + this.props.name)
      .datum(data)
      .call(fg);
  };
}

export default withStyles(styles, { withTheme: true })(FlameGraph);
