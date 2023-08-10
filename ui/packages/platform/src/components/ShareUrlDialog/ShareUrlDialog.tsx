/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { Component } from 'react'
import {
  IconButton,
  TextField,
  Dialog,
  Typography,
  Radio,
  RadioGroup,
  FormControlLabel,
  Button,
  makeStyles,
} from '@material-ui/core'
import MuiDialogTitle from '@material-ui/core/DialogTitle'
import MuiDialogContent from '@material-ui/core/DialogContent'
import MuiDialogActions from '@material-ui/core/DialogActions'

import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { ClassesType, RefluxTypes } from '@postgres.ai/platform/src/components/types'

import Store from '../../stores/store'
import Actions from '../../actions/actions'
import Urls from '../../utils/urls'

interface ShareUrlDialogProps {
  classes: ClassesType
}

interface DialogTitleProps {
  id: string
  children: React.ReactNode
  onClose: () => void
}

interface ShareUrlDialogState {
  data: {
    shareUrl: {
      shared: string | null
      uuid: string | null
      remark: string | null
      isProcessed: boolean
      open: boolean
      isRemoving: boolean
      isAdding: boolean
      data: {
        uuid: string | null
      } | null
    }
  } | null
  shared: string | null
  uuid: string | null
}

const DialogTitle = (props: DialogTitleProps) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        margin: 0,
        padding: theme.spacing(1),
      },
      dialogTitle: {
        fontSize: 16,
        lineHeight: '19px',
        fontWeight: 600,
      },
      closeButton: {
        position: 'absolute',
        right: theme.spacing(1),
        top: theme.spacing(1),
        color: theme.palette.grey[500],
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()
  const { children, onClose, ...other } = props
  return (
    <MuiDialogTitle disableTypography className={classes.root} {...other}>
      <Typography className={classes.dialogTitle}>{children}</Typography>
      {onClose ? (
        <IconButton
          aria-label="close"
          className={classes.closeButton}
          onClick={onClose}
        >
          {icons.closeIcon}
        </IconButton>
      ) : null}
    </MuiDialogTitle>
  )
}

const DialogContent = (props: { children: React.ReactNode }) => {
  const useStyles = makeStyles(
    (theme) => ({
      dialogContent: {
        paddingTop: 10,
        padding: theme.spacing(2),
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()
  return (
    <MuiDialogContent dividers className={classes.dialogContent}>
      {props.children}
    </MuiDialogContent>
  )
}

const DialogActions = (props: { children: React.ReactNode }) => {
  const useStyles = makeStyles(
    (theme) => ({
      root: {
        margin: 0,
        padding: theme.spacing(1),
      },
    }),
    { index: 1 },
  )

  const classes = useStyles()
  return (
    <MuiDialogActions className={classes.root}>
      {props.children}
    </MuiDialogActions>
  )
}

class ShareUrlDialog extends Component<
  ShareUrlDialogProps,
  ShareUrlDialogState
> {
  state: ShareUrlDialogState = {
    shared: null,
    uuid: null,
    data: {
      shareUrl: {
        shared: null,
        uuid: null,
        remark: null,
        isProcessed: false,
        isRemoving: false,
        isAdding: false,
        open: false,
        data: null,
      },
    },
  }

  unsubscribe: Function
  componentDidMount() {
    const that = this
    // const { url_type, url_id } = this.props;

     this.unsubscribe = (Store.listen as RefluxTypes["listen"]) (function () {
      let stateData = { data: this.data, shared: '', uuid: '' }

      if (this.data.shareUrl.isAdding) {
        return
      }

      if (this.data.shareUrl.data && this.data.shareUrl.data.uuid) {
        stateData.shared = 'shared'
        stateData.uuid = this.data.shareUrl.data.uuid
      } else {
        stateData.shared = 'default'
        stateData.uuid = this.data.shareUrl.uuid
      }

      that.setState(stateData)
    })

    Actions.refresh()
  }

  componentWillUnmount() {
    this.unsubscribe()
  }

  copyUrl = () => {
    let copyText = document.getElementById('sharedUrl') as HTMLInputElement

    if (copyText) {
      copyText.select()
      copyText.setSelectionRange(0, 99999)
      document.execCommand('copy')
    }
  }

  closeShareDialog = (close: boolean, save: boolean) => {
    Actions.closeShareUrlDialog(close, save, this.state.shared === 'shared')
    if (close) {
      this.setState({ data: null, shared: null })
    }
  }

  render() {
    const { classes } = this.props
    const shareUrl =
      this.state && this.state.data && this.state.data.shareUrl
        ? this.state.data.shareUrl
        : null

    if (
      !shareUrl ||
      (shareUrl && !shareUrl.isProcessed) ||
      (shareUrl && !shareUrl.open && !shareUrl.data) ||
      this.state.shared === null
    ) {
      return null
    }

    const urlField = (
      <div>
        <TextField
          id="token"
          className={classes.textField}
          margin="normal"
          value={Urls.linkShared(this.state.uuid)}
          variant="outlined"
          style={{ width: 500 }}
          InputProps={{
            readOnly: true,
            id: 'sharedUrl',
          }}
          InputLabelProps={{
            shrink: true,
            style: styles.inputFieldLabel,
          }}
          FormHelperTextProps={{
            style: styles.inputFieldHelper,
          }}
        />

        <IconButton
          className={classes.copyButton}
          aria-label="Copy"
          onClick={this.copyUrl}
        >
          {icons.copyIcon}
        </IconButton>
      </div>
    )

    return (
      <Dialog
        onClose={() => this.closeShareDialog(true, false)}
        aria-labelledby="customized-dialog-title"
        open={shareUrl.open}
        className={classes.dialog}
      >
        <DialogTitle
          id="customized-dialog-title"
          onClose={() => this.closeShareDialog(true, false)}
        >
          Share
        </DialogTitle>
        <DialogContent>
          <RadioGroup
            aria-label="shareUrl"
            name="shareUrl"
            value={this.state.shared}
            onChange={(event) => {
              this.setState({ shared: event.target.value })
            }}
            className={classes.radioLabel}
          >
            <FormControlLabel
              value="default"
              control={<Radio />}
              label="Only members of the organization can view"
            />

            <FormControlLabel
              value="shared"
              control={<Radio />}
              label="Anyone with a special link and members of thr organization can view"
            />
          </RadioGroup>
          {shareUrl.remark && (
            <Typography className={classes.remark}>
              <span className={classes.remarkIcon}>{icons.warningIcon}</span>
              {shareUrl.remark}
            </Typography>
          )}
          {this.state.shared === 'shared' && (
            <div className={classes.urlContainer}>{urlField}</div>
          )}
        </DialogContent>
        <DialogActions>
          <Button
            autoFocus
            variant="contained"
            disabled={shareUrl.isRemoving || shareUrl.isAdding}
            onClick={() => this.closeShareDialog(false, true)}
            color="primary"
          >
            Save changes
            {(shareUrl.isRemoving || shareUrl.isAdding) && (
              <span>
                &nbsp;
                <Spinner size="sm" />
              </span>
            )}
          </Button>
          <Button
            onClick={() => this.closeShareDialog(true, false)}
            variant="outlined"
            color="secondary"
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    )
  }
}

export default ShareUrlDialog
