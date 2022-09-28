/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Store from '../stores/store';
import Actions from '../actions/actions';
import { Component } from 'react';
import React from 'react';
import List from '@material-ui/core/List';
import { NavLink } from 'react-router-dom';
import { ListItem } from '@material-ui/core';


const styles = () => ({
  reportMenu: {
    fontSize: 13,
    padding: 0
  },
  subheader: {
    paddingLeft: 50,
    lineHeight: '25px'
  },
  listItem: {
    paddingTop: 5,
    paddingBottom: 5,
    paddingLeft: 40,
    fontSize: 13
  },
  titleListItem: {
    paddingLeft: 20,
    fontSize: 13
  },
  navLink: {
    textDecoration: 'none!important',
    color: '#000000',
    width: '100%',
    marginLeft: 10,
    paddingTop: 0,
    paddingBottom: 0,
    paddingLeft: 24,
    paddingRight: 24,
    fontSize: 13
  },
  activeNavLink: {
    color: '#3F51B5',
    fontWeight: 'bold',
    width: '100%',
    fontSize: 13
  }
});

class ReportMenu extends Component {
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

  handleFileClick = (event, reportId, id, type) => {
    const auth = this.state && this.state.data && this.state.data.auth ?
      this.state.data.auth : null;

    Actions.getCheckupReportFile(auth.token, id, type);
    this.props.history.push('/reportfile/' + reportId + '/' + id + '/' + type);
  };

  handleReportClick = (event, reportId) => {
    this.props.history.push('/report/' + reportId);
  };

  render() {
    const { classes } = this.props;
    const data = this.state.data && this.state.data.report ?
      this.state.data.report : null;

    if (!true && data.reportId && data.data && data.data.length > 0) {
      return (
        <div>
          <ListItem
            button
            className={classes.titleListItem}
            onClick={event => this.handleReportClick(event, data.reportId)}
          >
            <NavLink
              className={classes.navLink}
              activeClassName={classes.activeNavLink}
              to={'/report/' + data.reportId}
            >
              Checkup Report #{data.reportId}
            </NavLink>
          </ListItem>

          {this.props.showFiles ? (
            <List component='nav' className={classes.reportMenu}>
              {data.data.map(f => {
                return (
                  <div>
                    { f.filename.startsWith('0_') ? '' : (
                      <ListItem
                        button
                        className={classes.listItem}
                        onClick={event => this.handleFileClick(event, data.reportId, f.id, f.type)}
                      >
                        <NavLink
                          className={classes.navLink}
                          activeClassName={classes.activeNavLink}
                          to={'/reportfile/' + data.reportId + '/' + f.id + '/' + f.type}
                        >
                          {f.filename.split('.')[0]}
                        </NavLink>
                      </ListItem>)
                    }
                  </div>
                );
              })}
            </List>) : ''}
        </div>
      );
    }

    return (<div/>);
  }
}

ReportMenu.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  history: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(ReportMenu);
