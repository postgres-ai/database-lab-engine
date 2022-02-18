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

const linkColor = colors.secondary2.main;

const styles = () => ({
  link: {
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
  }
});

// TODO(Anton): legacy, remove later, change to Link2.
// Such type of link can be used only for external targets. ReactRouter does not control such links.
class Link extends Component {
  render() {
    const { classes } = this.props;

    const { link, target, children } = this.props;

    return (
      <a className={classes.link} href={link} target={target}>{children}</a>
    );
  }
}

Link.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(Link);
