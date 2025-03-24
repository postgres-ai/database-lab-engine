/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect, useState } from 'react'
import { useRouteMatch } from "react-router-dom";
import {
  Button,
  Dialog,
  FormControlLabel,
  IconButton,
  makeStyles,
  Radio,
  RadioGroup,
  TextField, Theme,
  Typography,
} from '@material-ui/core'
import MuiDialogTitle from '@material-ui/core/DialogTitle'
import MuiDialogContent from '@material-ui/core/DialogContent'
import MuiDialogActions from '@material-ui/core/DialogActions'
import FormLabel from '@mui/material/FormLabel'
import { styles } from '@postgres.ai/shared/styles/styles'
import { icons } from '@postgres.ai/shared/styles/icons'
import { Spinner } from '@postgres.ai/shared/components/Spinner'
import { useAiBot, Visibility } from "../hooks";
import { AiModel } from "../../../types/api/entities/bot";
import settings from "../../../utils/settings";
import { Link } from "@postgres.ai/shared/components/Link2";
import { ExternalIcon } from "@postgres.ai/shared/icons/External";
import Divider from "@material-ui/core/Divider";
import cn from "classnames";

type DialogTitleProps = {
  id: string
  children: React.ReactNode
  onClose: () => void
}

type PublicChatDialogProps = {
  isOpen: boolean
  onClose: () => void
  threadId: string | null
  orgAlias: string
  isSubscriber: boolean
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

const useDialogStyles = makeStyles<Theme>(
  (theme) => ({
    textField: {
      ...styles.inputField,
      marginTop: '0px',
      width: 480,
      [theme.breakpoints.down('sm')]: {

      }
    },
    copyButton: {
      marginTop: '-3px',
      fontSize: '20px',
    },
    urlContainer: {
      marginTop: 8,
      paddingLeft: 20,
      [theme.breakpoints.down('sm')]: {
        padding: 0,
        width: '100%',
        '& .MuiTextField-root': {
          maxWidth: 'calc(100% - 36px)'
        }
      },
    },
    radioGroup: {
      fontSize: 12,
      '&:not(:last-child)': {
        marginBottom: 12
      }
    },
    dialogContent: {
      paddingTop: 10,
    },
    unlockNote: {
      marginTop: 2,
      '& ol': {
        paddingLeft: 18,
        marginTop: 4,
        marginBottom: 0
      }
    },
    unlockNoteDemo: {
      paddingLeft: 20
    },
    formControlLabel: {
      '& .Mui-disabled > *, & .Mui-disabled': {
        color: 'rgba(0, 0, 0, 0.6)'
      },
      [theme.breakpoints.down('sm')]: {
        marginRight: 0,
        alignItems: 'flex-start',
        '&:first-child': {
          marginTop: 6
        }
      },
    },
    formControlLabelRadio: {
      [theme.breakpoints.down('sm')]: {
        padding: '4px 9px'
      }
    },
    externalIcon: {
      width: 14,
      height: 14,
      marginLeft: 4,
      transform: 'translateY(2px)',
    },
    divider: {
      margin: '12px 0'
    }
  }),
  { index: 1 },
)

export const SettingsDialog = (props: PublicChatDialogProps) => {
  const {
    onClose,
    isOpen,
    threadId,
    orgAlias,
    isSubscriber
  } = props;

  const {
    chatVisibility,
    changeChatVisibility,
    isChangeVisibilityLoading,
    getChatsList,
    aiModels,
    aiModel: activeModel,
    setAiModel: setActiveModel,
    setChatVisibility
  } = useAiBot();

  const [model, setModel] = useState<AiModel | null>(activeModel)
  const [visibility, setVisibility] = useState<Visibility>(chatVisibility);

  const classes = useDialogStyles();

  const publicUrl = `https://postgres.ai/chats/${threadId}`;

  const isDemoOrg = useRouteMatch(`/${settings.demoOrgAlias}`);

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
      changeChatVisibility(threadId, visibility === Visibility.PUBLIC)
      getChatsList();
    } else if (visibility !== chatVisibility) {
      setChatVisibility(visibility)
    }
    onClose()
  }

  useEffect(() => {
    if (isOpen) {
      if (visibility !== chatVisibility) {
        setVisibility(chatVisibility)
      }
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
        Chat settings
      </DialogTitle>
      <DialogContent>
        <>
          <FormLabel component="legend">Visibility</FormLabel>
          <RadioGroup
            aria-label="Thread visibility"
            name="threadVisibility"
            value={visibility}
            onChange={(event) => {
              setVisibility(event.target.value as Visibility)
            }}
            className={classes.radioGroup}
          >
            <FormControlLabel
              value={Visibility.PUBLIC}
              className={classes.formControlLabel}
              control={<Radio className={classes.formControlLabelRadio} />}
              label={<><b>Public:</b> anyone can view chats, but only team members can respond</>}
              aria-label="Public: anyone can view chats, but only team members can respond"
            />
            {visibility === Visibility.PUBLIC && threadId && (
              <div className={classes.urlContainer}>{urlField}</div>
            )}
            <FormControlLabel
              value={Visibility.PRIVATE}
              className={classes.formControlLabel}
              control={<Radio className={classes.formControlLabelRadio} />}
              label={<><b>Private:</b> chats are visible only to members of your organization</>}
              aria-label="Private: chats are visible only to members of your organization"
              disabled={Boolean(isDemoOrg) || !isSubscriber}
            />
            {Boolean(isDemoOrg) && <Typography className={cn(classes.unlockNote, classes.unlockNoteDemo)}>Private chats are not allowed in "Demo"</Typography>}
            {!Boolean(isDemoOrg) && !isSubscriber && <Typography variant="body2" className={classes.unlockNote}>
              Unlock private conversations by either:
              <ol>
                <li>
                  <Link to={`/${orgAlias}/instances`} target="_blank">
                    Installing a DBLab SE instance
                    <ExternalIcon className={classes.externalIcon}/>
                  </Link>
                </li>
                <li>
                  <Link external to="https://postgres.ai/consulting" target="_blank">
                    Becoming a Postgres.AI consulting customer
                    <ExternalIcon className={classes.externalIcon}/>
                  </Link>
                </li>
              </ol>
            </Typography>}
          </RadioGroup>
        </>
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