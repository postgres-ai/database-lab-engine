/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import clsx from 'clsx';

import { colors } from '@postgres.ai/shared/styles/colors';
import { theme } from '@postgres.ai/shared/styles/theme';

const styles = muiTheme => ({
  /* eslint no-dupe-keys: 0 */
  root: {
    '& h1': {
      fontSize: '16px',
      margin: '0'
    },
    [muiTheme.breakpoints.down('xs')]: {
      height: '350px'
    },
    'fontFamily': theme.typography.fontFamily,
    'fontSize': '14px',
    'border': '1px solid ' + colors.consoleStroke,
    'maxWidth': '450px',
    'height': '260px',
    'paddingTop': '15px',
    'paddingBottom': '15px',
    'paddingLeft': '20px',
    'paddingRight': '20px',
    'alignItems': 'center',
    'borderRadius': '3px',
    // Flexbox.
    'display': '-ms-flexbox',
    'display': '-webkit-flex',
    'display': 'flex',
    '-webkit-flex-direction': 'column',
    '-ms-flex-direction': 'column',
    'flex-direction': 'column',
    '-webkit-flex-wrap': 'nowrap',
    '-ms-flex-wrap': 'nowrap',
    'flex-wrap': 'nowrap',
    '-webkit-justify-content': 'space-between',
    '-ms-flex-pack': 'justify',
    'justify-content': 'space-between',
    '-webkit-align-content': 'flex-start',
    '-ms-flex-line-pack': 'start',
    'align-content': 'flex-start',
    '-webkit-align-items': 'flex-start',
    '-ms-flex-align': 'start',
    'align-items': 'flex-start'
  },
  block: {
    marginBottom: '20px'
  },
  icon: {
    padding: '5px'
  },
  actionsContainer: {
    marginTop: '15px',

    [muiTheme.breakpoints.down('xs')]: {
      marginTop: '5px'
    }
  },
  contentContainer: {
    marginTop: '15px'
  },
  bottomContainer: {
    'width': '100%',
    'display': '-ms-flexbox',
    'display': '-webkit-flex',
    'display': 'flex',
    '-webkit-flex-direction': 'row',
    '-ms-flex-direction': 'row',
    'flex-direction': 'row',
    '-webkit-flex-wrap': 'wrap',
    '-ms-flex-wrap': 'wrap',
    'flex-wrap': 'wrap',
    '-webkit-justify-content': 'space-between',
    '-ms-flex-pack': 'justify',
    'justify-content': 'space-between',
    '-webkit-align-content': 'stretch',
    '-ms-flex-line-pack': 'stretch',
    'align-content': 'stretch',
    '-webkit-align-items': 'flex-end',
    '-ms-flex-align': 'end',
    'align-items': 'flex-end',

    [muiTheme.breakpoints.down('xs')]: {
      flexDirection: 'column',
      alignItems: 'center'
    }
  },
  buttonSpan: {
    '&:not(:first-child)': {
      marginLeft: '15px'
    },

    [muiTheme.breakpoints.down('xs')]: {
      '&:not(:first-child)': {
        marginLeft: 0
      },

      '& button': {
        marginTop: '15px',
        width: '100%'
      }
    }
  }
});

class ProductCard extends Component {
  render() {
    const { classes, children, actions, inline, title, icon, style, className } = this.props;

    return (
      <div
        className={clsx(!inline && classes.block, classes.root, className)}
        style={ style }
      >
        <div>
          <h1>{ title }</h1>
          <div className={ classes.contentContainer }>
            { children }
          </div>
        </div>

        <div className={ classes.bottomContainer }>
          { icon }
          <div className={ classes.actionsContainer }>
            { actions?.map(a => {
              return (
                <span key={a.id} className={ classes.buttonSpan }>
                  {a.content}
                </span>
              );
            }) }
          </div>
        </div>
      </div>
    );
  }
}

ProductCard.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired,
  className: PropTypes.string,
  title: PropTypes.string,
  icon: PropTypes.icon,
  actions: PropTypes.arrayOf(
    PropTypes.shape({
      id: PropTypes.string.isRequired,
      content: PropTypes.any.isRequired
    }).isRequired
  )
};

export default withStyles(styles, { withTheme: true })(ProductCard);
