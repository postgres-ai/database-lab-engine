import React from "react";
import Alert from '@mui/material/Alert';
import ReactMarkdown from "react-markdown";
import { makeStyles } from "@material-ui/core";

const useStyles = makeStyles(() => ({
  message: {
    '& p': {
      padding: 0,
      margin: 0
    }
  }
}))

type ErrorMessageProps = {
  content: string
}

export const ErrorMessage = (props: ErrorMessageProps) => {
  const { content } = props;
  const classes = useStyles()
  return (
    <Alert severity="error">
      <ReactMarkdown className={classes.message}>{content}</ReactMarkdown>
    </Alert>
  )
}