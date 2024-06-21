import React, { useEffect, useRef } from "react";
import Dialog from "@material-ui/core/Dialog";
import { DialogContent, DialogTitle, makeStyles } from "@material-ui/core";
import { useAiBot } from "../hooks";
import IconButton from "@material-ui/core/IconButton";
import CloseIcon from "@material-ui/icons/Close";
import { DebugLogs } from "../DebugLogs/DebugLogs";
import { createMessageFragment } from "../utils";

const useStyles = makeStyles(
  (theme) => ({
    dialogStyle: {
      top: '5%!important',
      right: '10%!important',
      left: 'unset!important',
      height: '80vh',
      width: '80vw',
    },
    paper: {
      width: '80vw',
      height: '80vh',
      opacity: '.5',
      transition: '.2s ease',
      '&:hover': {
        opacity: 1
      }
    },
    dialogTitle: {
      padding: '0.5rem',
      '& h2': {
        fontSize: '1rem',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center'
      }
    },
    dialogContent: {
      padding: 0,
      margin: 0,
    }
  }))

type DebugConsoleProps = {
  isOpen: boolean
  onClose: () => void
  threadId?: string
}

export const DebugConsole = (props: DebugConsoleProps) => {
  const { isOpen, onClose, threadId } = props;
  const { debugMessages, debugMessagesLoading } = useAiBot();
  const classes = useStyles();
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (containerRef.current && debugMessages?.length && threadId && !debugMessagesLoading && isOpen) {
      let code = containerRef.current.getElementsByTagName('code')?.[0];
      if (!code) {
        code = document.createElement('code');
        containerRef.current.appendChild(code);
      }

      if (code.hasChildNodes()) {
        const lastMessage = debugMessages[debugMessages.length - 1];
        const fragment = createMessageFragment([lastMessage]);
        code.appendChild(fragment);
      } else {
        const fragment = createMessageFragment(debugMessages);
        code.appendChild(fragment);
      }
    }
  }, [debugMessages, isOpen, threadId, debugMessagesLoading]);

  return (
    <Dialog
      open={isOpen}
      disableEnforceFocus
      hideBackdrop
      className={classes.dialogStyle}
      classes={{paper: classes.paper}}
      scroll="paper"
      fullWidth
      maxWidth="xl"
    >
      <DialogTitle className={classes.dialogTitle}>
        Debug console
        <IconButton
          onClick={onClose}
        >
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent
        classes={{root: classes.dialogContent}}
        ref={containerRef}
      >
        <DebugLogs
          isLoading={debugMessagesLoading}
          isEmpty={debugMessages?.length === 0}
          id={threadId!}
        />
      </DialogContent>
    </Dialog>
  )
}