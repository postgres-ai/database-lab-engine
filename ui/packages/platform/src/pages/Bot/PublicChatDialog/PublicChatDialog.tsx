/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useState } from 'react'
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
import { colors } from "@postgres.ai/shared/styles/colors";

type DialogTitleProps = {
  id: string
  children: React.ReactNode
  onClose: () => void
}

type PublicChatDialogProps = {
  defaultValue: 'public' | 'private'
  isOpen: boolean
  onClose: () => void
  onSaveChanges: (value: boolean) => void
  isLoading: boolean
  threadId: string
}

const useDialogTitleStyles = makeStyles(
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
      top: 4,
      color: theme.palette.grey[500],
    },
  }),
  { index: 1 },
)

const DialogTitle = (props: DialogTitleProps) => {
  const classes = useDialogTitleStyles()
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

const useDialogContentStyles = makeStyles(
  (theme) => ({
    dialogContent: {
      paddingTop: 10,
      padding: theme.spacing(2),
    },
  }),
  { index: 1 },
)

const DialogContent = (props: { children: React.ReactNode }) => {
  const classes = useDialogContentStyles()
  return (
    <MuiDialogContent dividers className={classes.dialogContent}>
      {props.children}
    </MuiDialogContent>
  )
}

const useDialogActionsStyles = makeStyles(
  (theme) => ({
    root: {
      margin: 0,
      padding: theme.spacing(1),
    },
  }),
  { index: 1 },
)

const DialogActions = (props: { children: React.ReactNode }) => {
  const classes = useDialogActionsStyles()
  return (
    <MuiDialogActions className={classes.root}>
      {props.children}
    </MuiDialogActions>
  )
}

const useDialogStyles = makeStyles(
  () => ({
    textField: {
      ...styles.inputField,
      marginTop: '0px',
      width: 480,
    },
    copyButton: {
      marginTop: '-3px',
      fontSize: '20px',
    },
    dialog: {},
    remark: {
      fontSize: 12,
      lineHeight: '12px',
      color: colors.state.warning,
      paddingLeft: 20,
    },
    remarkIcon: {
      display: 'block',
      height: '20px',
      width: '22px',
      float: 'left',
      paddingTop: '5px',
    },
    urlContainer: {
      marginTop: 10,
      paddingLeft: 22,
    },
    radioLabel: {
      fontSize: 12,
    },
    dialogContent: {
      paddingTop: 10,
    },
  }),
  { index: 1 },
)

export const PublicChatDialog = (props: PublicChatDialogProps) => {
  const { onSaveChanges, defaultValue, onClose, isOpen, isLoading, threadId } = props;

  const [visibility, setVisibility] = useState(defaultValue ? "public" : "private");

  const classes = useDialogStyles();

  const publicUrl = `https://postgres.ai/chats/${threadId}`;

  const handleCopyUrl = () => {
    if ('clipboard' in navigator) {
      navigator.clipboard.writeText(publicUrl);
    }
  }

  const handleSaveChanges = () => {
    if (defaultValue !== visibility) {
      onSaveChanges(visibility === 'public');
    }
  }

  const urlField = (
    <div>
      <TextField
        id="token"
        className={classes.textField}
        margin="normal"
        value={publicUrl}
        variant="outlined"
        style={{ width: 500 }}
        onFocus={(event) => event.target.select()}
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
        onClick={handleCopyUrl}
      >
        {icons.copyIcon}
      </IconButton>
    </div>
  )

  return (
    <Dialog
      onClose={onClose}
      aria-labelledby="customized-dialog-title"
      open={isOpen}
      className={classes.dialog}
    >
      <DialogTitle
        id="customized-dialog-title"
        onClose={onClose}
      >
        Public Chat
      </DialogTitle>
      <DialogContent>
        <RadioGroup
          aria-label="shareUrl"
          name="shareUrl"
          value={visibility}
          onChange={(event) => {
            setVisibility(event.target.value)
          }}
          className={classes.radioLabel}
        >
          <FormControlLabel
            value="private"
            control={<Radio />}
            label="Only members of the organization can view"
          />

          <FormControlLabel
            value="public"
            control={<Radio />}
            label="Anyone with a special link and members of the organization can view"
          />
        </RadioGroup>
        {/*{shareUrl.remark && (
            <Typography className={classes.remark}>
              <span className={classes.remarkIcon}>{icons.warningIcon}</span>
              {shareUrl.remark}
            </Typography>
          )}*/}
        {visibility && (
            <div className={classes.urlContainer}>{urlField}</div>
          )}
      </DialogContent>
      <DialogActions>
        <Button
          autoFocus
          variant="contained"
          disabled={isLoading}
          onClick={handleSaveChanges}
          color="primary"
        >
          Save changes
          {isLoading && (
              <span>
                &nbsp;
                <Spinner size="sm" />
              </span>
            )}
        </Button>
        <Button
          onClick={() => onClose()}
          variant="outlined"
          color="secondary"
        >
          Close
        </Button>
      </DialogActions>
    </Dialog>
  )
}