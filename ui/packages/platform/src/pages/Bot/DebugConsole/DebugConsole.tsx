import React, { useEffect, useRef, useState } from "react";
import Dialog from "@material-ui/core/Dialog";
import { DialogContent, DialogTitle, makeStyles } from "@material-ui/core";
import { useAiBot } from "../hooks";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";
import ReactMarkdown from "react-markdown";
import IconButton from "@material-ui/core/IconButton";
import CloseIcon from "@material-ui/icons/Close";
import { disallowedHtmlTagsForMarkdown } from "../utils";
import { DebugLogs } from "../DebugLogs/DebugLogs";
import { DebugMessage } from "../../../types/api/entities/bot";

const useStyles = makeStyles(
  (theme) => ({
    dialogStyle: {
      top: '5%!important',
      right: '10%!important',
      left: 'unset!important',
      height: 'fit-content',
      width: 'fit-content',
    },
    paper: {
      width: '80vw',
      height: '70vh',
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
      let code: HTMLElement = containerRef.current.getElementsByTagName('code')?.[0];
      if (code.hasChildNodes()) {
        code.appendChild(document.createTextNode(`[${debugMessages[debugMessages.length - 1].created_at}]: ${debugMessages[debugMessages.length - 1].content}\n`))
      } else {
        debugMessages.forEach((item) => {
          code.appendChild(document.createTextNode(`[${item.created_at}]: ${item.content}\n`))
        })
        const container = document.getElementById(`logs-container-${threadId}`);
        if (container) {
          container.appendChild(code)
        }
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