import React from 'react'
import Dialog from "@material-ui/core/Dialog";
import DialogTitle from '@material-ui/core/DialogTitle';
import { DialogContent, IconButton, makeStyles, Typography } from "@material-ui/core";
import ReactMarkdown from "react-markdown";
import Format from "../../../utils/format";
import { icons } from "@postgres.ai/shared/styles/icons";
import { disallowedHtmlTagsForMarkdown } from "../utils";

type DebugDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  debugMessages: { created_at: string, message: string }[]
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
    message: {
      marginBottom: 12,
      '& > *': {
        fontFamily: "'Menlo', 'DejaVu Sans Mono', 'Liberation Mono', 'Consolas', 'Ubuntu Mono', 'Courier New'," +
          " 'andale mono', 'lucida console', monospace",
        fontSize: '0.813rem',
        lineHeight: '120%'
      },
    },
    time: {
      marginBottom: 4,
      display: 'inline-block'
    }
  }))

export const DebugDialog = (props: DebugDialogProps) => {
  const {isOpen, onClose, debugMessages} = props;
  const classes = useStyles()
  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      fullWidth
      maxWidth="sm"
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
      <DialogContent>
        {
          debugMessages.map((debugMessage) => {
              const formattedTime = debugMessage.created_at ? Format.timeAgo(debugMessage.created_at) : null
              return (
                <div className={classes.message}>
                  <Typography
                    component="span"
                    title={debugMessage.created_at}
                    className={classes.time}
                  >
                    {formattedTime}
                  </Typography>
                  <ReactMarkdown
                    linkTarget='_blank'
                    components={{
                      p: 'div',
                    }}
                    disallowedElements={disallowedHtmlTagsForMarkdown}
                    unwrapDisallowed
                  >
                    {debugMessage.message}
                  </ReactMarkdown>
                </div>
              )
            }
          )
        }
        {
          (!debugMessages || debugMessages.length === 0) && (
            <div className={classes.message}>
              <Typography>No debug information available for the selected message.</Typography>
            </div>
          )
        }
      </DialogContent>
    </Dialog>
  )
}