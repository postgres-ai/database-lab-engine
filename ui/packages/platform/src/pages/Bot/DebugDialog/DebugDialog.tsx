import React, { useEffect, useRef, useState } from 'react'
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from '@material-ui/core/DialogTitle';
import { DialogContent, IconButton, makeStyles, Typography } from "@material-ui/core";
import ReactMarkdown from "react-markdown";
import Format from "../../../utils/format";
import { icons } from "@postgres.ai/shared/styles/icons";
import { disallowedHtmlTagsForMarkdown } from "../utils";
import { BotMessage, DebugMessage } from "../../../types/api/entities/bot";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import { getDebugMessages } from "../../../api/bot/getDebugMessages";
import { PageSpinner } from "@postgres.ai/shared/components/PageSpinner";
import { SyntaxHighlight } from "@postgres.ai/shared/components/SyntaxHighlight";
import { DebugLogs } from "../DebugLogs/DebugLogs";

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
    const code = document.createElement('code');
    messages.forEach((item) => {
      code.appendChild(document.createTextNode(`[${item.created_at}]: ${item.content}\n`))
    })
    const container = document.getElementById(`logs-container-${messageId}`);
    if (container) {
      container.appendChild(code)
    }
  }

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