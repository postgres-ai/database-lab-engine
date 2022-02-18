/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';

import { colors } from '@postgres.ai/shared/styles/colors';
import { icons } from '@postgres.ai/shared/styles/icons';

const linkColor = colors.secondary2.main;

const styles = () => ({
  root: {
    backgroundColor: colors.secondary1.lightLight,
    color: colors.secondary1.darkDark,
    fontSize: '14px',
    paddingTop: '5px',
    paddingBottom: '5px',
    paddingLeft: '10px',
    paddingRight: '10px',
    display: 'flex',
    alignItems: 'center',
    borderRadius: '3px'
  },
  block: {
    marginBottom: '20px'
  },
  icon: {
    padding: '5px'
  },
  actions: {
    '& a': {
      'color': linkColor,
      '&:visited': {
        color: linkColor
      },
      '&:hover': {
        color: linkColor
      },
      '&:active': {
        color: linkColor
      }
    },
    'marginRight': '15px'
  },
  container: {
    marginLeft: '5px',
    marginRight: '5px',
    lineHeight: '24px'
  }
});

class Warning extends Component {
  render() {
    const { classes, children, actions, inline } = this.props;

    return (
      <div className={(!inline ? `${classes.block} ` : '') + classes.root}>
        { icons.warningIcon }

        <div className={ classes.container }>
          { children }
        </div>

        {actions ? (
          <span className={classes.actions}>
            {actions.map(a => {
              return (
                <span key={a.key} className={classes.pageTitleActionContainer}>
                  {a}
                </span>
              );
            })}
          </span>
        ) : null}
      </div>
    );
  }
}

Warning.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(Warning);
