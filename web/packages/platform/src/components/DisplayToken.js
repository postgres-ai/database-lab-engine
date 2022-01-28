/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react';
import PropTypes from 'prop-types';
import { InputAdornment } from '@material-ui/core';
import { withStyles } from '@material-ui/core/styles';
import IconButton from '@material-ui/core/IconButton';
import TextField from '@material-ui/core/TextField';

import { styles } from '@postgres.ai/shared/styles/styles';
import { icons } from '@postgres.ai/shared/styles/icons';

import Store from '../stores/store';
import Actions from '../actions/actions';

const getStyles = () => ({
  textField: {
    ...styles.inputField,
    marginTop: 0
  },
  input: {
    '&.MuiOutlinedInput-adornedEnd': {
      padding: 0
    }
  },
  // TODO (Anton): Rewrite styling of TextField component and remove !important everywhere.
  inputElement: {
    marginRight: '-8px'
  },
  inputAdornment: {
    margin: 0
  },
  inputButton: {
    padding: '9px 10px'
  }
});

class DisplayToken extends Component {
  componentDidMount() {
    const that = this;

    document.getElementsByTagName('html')[0].style.overflow = 'hidden';

    this.unsubscribe = Store.listen(function () {
      that.setState({ data: this.data });
    });

    Actions.refresh();
  }

  componentWillUnmount() {
    this.unsubscribe();
  }

  copyToken = () => {
    let copyText = document.getElementById('generatedToken');

    copyText.select();
    copyText.setSelectionRange(0, 99999);
    document.execCommand('copy');
  }

  render() {
    const { classes } = this.props;
    const tokenRequest = this.state && this.state.data &&
      this.state.data.tokenRequest ? this.state.data.tokenRequest : null;
    let tokenDisplay = null;

    if (tokenRequest && tokenRequest.isProcessed && !tokenRequest.error &&
      tokenRequest.data && tokenRequest.data.name &&
      tokenRequest.data.expires_at && tokenRequest.data.token) {
      tokenDisplay = (
        <TextField
          id='token'
          className={classes.textField}
          margin='normal'
          value={tokenRequest.data.token}
          variant='outlined'
          style={{ width: '100%', maxWidth: '500px' }}
          InputProps={{
            className: classes.input,
            classes: {
              input: classes.inputElement
            },
            readOnly: true,
            id: 'generatedToken',
            endAdornment: (
              <InputAdornment position='end' className={classes.inputAdornment}>
                <IconButton
                  className={classes.inputButton}
                  aria-label='Copy'
                  onClick={this.copyToken}>
                  {icons.copyIcon}
                </IconButton>
              </InputAdornment>
            )
          }}
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel
          }}
          FormHelperTextProps={{
            style: styles.inputFieldHelper
          }}
          helperText='Make sure you have saved token - you will not be able to access it again'
        />
      );
    }

    return (
      <div>
        {tokenDisplay}
      </div>
    );
  }
}

DisplayToken.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(getStyles, { withTheme: true })(DisplayToken);
