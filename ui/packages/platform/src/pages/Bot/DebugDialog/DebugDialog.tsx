import React, { useEffect, useRef, useState } from 'react'
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from '@material-ui/core/DialogTitle';
import { DialogContent, IconButton, makeStyles } from "@material-ui/core";
import { icons } from "@postgres.ai/shared/styles/icons";
import { DebugMessage } from "../../../types/api/entities/bot";
import { getDebugMessages } from "../../../api/bot/getDebugMessages";
import { DebugLogs } from "../DebugLogs/DebugLogs";
import { createMessageFragment } from "../utils";

type DebugDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  messageId: string
}

const useStyles = makeStyles(
  (theme) => ({
    dialogTitle: {
      fontSize: '1rem',
      paddingBottom: '8px',
      '& > *': {
        fontSize: 'inherit'
      }
    },
    closeButton: {
      position: 'absolute',
      right: theme.spacing(1),
      top: theme.spacing(1),
      color: theme.palette.grey[500],
    },
    dialogContent: {
      backgroundColor: 'rgb(250, 250, 250)',
      padding: '0.5rem 0',
      '& pre': {
        whiteSpace: 'pre-wrap!important'
      }
    }
  }))

export const DebugDialog = (props: DebugDialogProps) => {
  const {isOpen, onClose, messageId} = props;
  const classes = useStyles()

  const [debugLoading, setDebugLoading] = useState(false);
  const debugMessages = useRef<DebugMessage[] | null>(null)

  const generateMessages = (messages: DebugMessage[]) => {
    const container = document.getElementById(`logs-container-${messageId}`);
    if (container) {
      let code = container.getElementsByTagName('code')?.[0];
      if (!code) {
        code = document.createElement('code');
        container.appendChild(code);
      }

      const fragment = createMessageFragment(messages);
      code.appendChild(fragment);
    }
  };

  const getDebugMessagesForMessage = async () => {
    setDebugLoading(true)
    if (messageId) {
      const { response} = await getDebugMessages({ message_id: messageId })
      if (response) {
        debugMessages.current = response;
        generateMessages(response)
      }
    }
  }

  useEffect(() => {
    if (isOpen && !debugMessages.current) {
      getDebugMessagesForMessage()
        .then(() => setDebugLoading(false))
    } else if (isOpen && debugMessages.current && debugMessages.current?.length > 0) {
      setTimeout(() => generateMessages(debugMessages.current!), 0)
    }
  }, [isOpen]);


  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      fullWidth
      maxWidth="lg"
    >
      <DialogTitle classes={{root: classes.dialogTitle}}>
        Debug
        <IconButton
          aria-label="close"
          className={classes.closeButton}
          onClick={onClose}
        >
          {icons.closeIcon}
        </IconButton>
      </DialogTitle>
      <DialogContent
        classes={{root: classes.dialogContent}}
      >
        <DebugLogs
          id={messageId}
          isEmpty={!debugMessages || debugMessages.current?.length === 0}
          isLoading={debugLoading}
        />
      </DialogContent>
    </Dialog>
  )
}