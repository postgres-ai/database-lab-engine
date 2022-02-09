/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import Tooltip from '@material-ui/core/Tooltip';
import Button from '@material-ui/core/Button';


const styles = () => ({
  tooltip: {
    fontSize: '10px!important'
  }
});

class ConsoleButton extends Component {
  render() {
    const { classes, title, children, ...other } = this.props;

    // We have to use external tooltip component as disable button cannot show tooltip.
    // Details: https://material-ui.com/components/tooltips/#disabled-elements.
    return (
      <Tooltip classes={{ tooltip: classes.tooltip }} title={ title }>
        <span>
          <Button
            {...other}
            title={ null }
          >
            { children }
          </Button>
        </span>
      </Tooltip>
    );
  }
}

ConsoleButton.propTypes = {
  title: PropTypes.string,
  variant: PropTypes.string,
  color: PropTypes.string,
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(ConsoleButton);
