/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { Component } from 'react'
import { InputAdornment } from '@material-ui/core'
import { IconButton, TextField } from '@material-ui/core'

import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'
import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'

import Store from '../../stores/store'
import Actions from '../../actions/actions'

interface DisplayTokenProps {
  classes: ClassesType
}

interface DisplayTokenState {
  data: {
    tokenRequest: {
      isProcessed: boolean
      error: boolean
      data: {
        name: string
        expires_at: string
        token: string
      }
    } | null
  } | null
}

class DisplayToken extends Component<DisplayTokenProps, DisplayTokenState> {
  unsubscribe: Function
  componentDidMount() {
    const that = this

    document.getElementsByTagName('html')[0].style.overflow = 'hidden'

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      that.setState({ data: this.data })
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  copyToken = () => {
    const copyText = document.getElementById(
      'generatedToken',
    ) as HTMLInputElement

    if (copyText) {
      copyText.select()
      copyText.setSelectionRange(0, 99999)
      document.execCommand('copy')
    }
  }

  render() {
    const { classes } = this.props
    const tokenRequest =
      this.state && this.state.data && this.state.data.tokenRequest
        ? this.state.data.tokenRequest
        : null
    let tokenDisplay = null

    if (
      tokenRequest &&
      tokenRequest.isProcessed &&
      !tokenRequest.error &&
      tokenRequest.data &&
      tokenRequest.data.name &&
      tokenRequest.data.expires_at &&
      tokenRequest.data.token
    ) {
      tokenDisplay = (
        <TextField
          id="token"
          className={classes.textField}
          margin="normal"
          value={tokenRequest.data.token}
          variant="outlined"
          style={{ width: '100%', maxWidth: '500px' }}
          InputProps={{
            className: classes.input,
            classes: {
              input: classes.inputElement,
            },
            readOnly: true,
            id: 'generatedToken',
            endAdornment: (
              <InputAdornment position="end" className={classes.inputAdornment}>
                <IconButton
                  className={classes.inputButton}
                  aria-label="Copy"
                  onClick={this.copyToken}
                >
                  {icons.copyIcon}
                </IconButton>
              </InputAdornment>
            ),
          }}
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel,
          }}
          FormHelperTextProps={{
            style: styles.inputFieldHelper,
          }}
          helperText="Make sure you have saved token - you will not be able to access it again"
        />
      )
    }

    return <div>{tokenDisplay}</div>
  }
}

export default DisplayToken
