/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect, useState } from 'react'
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
import FormLabel from '@mui/material/FormLabel'
import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { colors } from "@postgres.ai/shared/styles/colors";
import { useAiBot } from "../hooks";
import { AiModel } from "../../../types/api/entities/bot";

export type Visibility = 'public' | 'private';

type DialogTitleProps = {
  id: string
  children: React.ReactNode
  onClose: () => void
}

type PublicChatDialogProps = {
  isOpen: boolean
  onClose: () => void
  threadId: string | null
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

export const SettingsDialog = (props: PublicChatDialogProps) => {
  const {
    onClose,
    isOpen,
    threadId,
  } = props;


  const {
    chatVisibility,
    changeChatVisibility,
    isChangeVisibilityLoading,
    getChatsList,
    aiModels,
    aiModel: activeModel,
    setAiModel: setActiveModel,
  } = useAiBot();

  const [model, setModel] = useState<AiModel | null>(activeModel)
  const [visibility, setVisibility] = useState<string>(chatVisibility);

  const classes = useDialogStyles();

  const publicUrl = `https://postgres.ai/chats/${threadId}`;

  const handleCopyUrl = () => {
    if ('clipboard' in navigator) {
      navigator.clipboard.writeText(publicUrl);
    }
  }

  const handleSaveChanges = () => {
    if (model && model !== activeModel) {
      setActiveModel(model)
    }
    if (visibility !== chatVisibility && threadId) {
      changeChatVisibility(threadId, visibility === 'public')
      getChatsList();
    }

    onClose()
  }

  useEffect(() => {
    if (isOpen) {
      if (visibility !== chatVisibility) {
        setVisibility(chatVisibility)
      }
      console.log('model', model)
      console.log('active', activeModel)
      console.log('eq', model?.name !== activeModel?.name)
      if (model?.name !== activeModel?.name) {
        setModel(activeModel)
      }
    }
  }, [isOpen]);

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
      fullWidth
    >
      <DialogTitle
        id="customized-dialog-title"
        onClose={onClose}
      >
        Chat Settings
      </DialogTitle>
      <DialogContent>
        {threadId && <>
          <FormLabel component="legend">Visibility</FormLabel>
          <RadioGroup
            aria-label="shareUrl"
            name="shareUrl"
            value={visibility}
            onChange={(event) => {
              setVisibility(event.target.value as Visibility)
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
          {visibility && (
              <div className={classes.urlContainer}>{urlField}</div>
            )}
        </>}
        {aiModels && <>
          <FormLabel component="legend">Model</FormLabel>
          <RadioGroup
            aria-label="model"
            name="model"
            value={`${model?.vendor}/${model?.name}`}
            onChange={(event) => {
              const selectedModel = aiModels?.find((model) => `${model.vendor}/${model.name}` === event.target.value)
              setModel(selectedModel!)
            }}
            className={classes.radioLabel}
          >
            {aiModels.map((model) =>
                <FormControlLabel
                  key={`${model.vendor}/${model.name}`}
                  value={`${model.vendor}/${model.name}`}
                  control={<Radio />}
                  label={model.name}
                />
              )
            }
          </RadioGroup>
        </>}
      </DialogContent>

      <DialogActions>
        <Button
          autoFocus
          variant="contained"
          disabled={isChangeVisibilityLoading}
          onClick={handleSaveChanges}
          color="primary"
        >
          Save changes
          {isChangeVisibilityLoading && (
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