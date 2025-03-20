import React from "react";
import cn from "classnames";
import { permalinkLinkBuilder } from "../../../utils";
import { makeStyles } from "@material-ui/core";
import { colors } from "@postgres.ai/shared/styles/colors";
import { BaseMessageProps } from "../Message";


const useStyles = makeStyles(
  () => ({
    messageAuthor: {
      fontSize: 14,
      fontWeight: 'bold',
    },
    messageInfo: {
      display: 'inline-block',
      marginLeft: 10,
      padding: 0,
      fontSize: '0.75rem',
      color: colors.pgaiDarkGray,
      transition: '.2s ease',
      background: "none",
      border: "none",
      textDecoration: "none",
      '@media (max-width: 450px)': {
        '&:nth-child(1)': {
          display: 'none'
        }
      }
    },
    messageInfoActive: {
      borderBottom: '1px solid currentcolor',
      cursor: 'pointer',
      '&:hover': {
        color: '#404040'
      }
    },
    messageHeader: {
      height: '1.125rem',
      display: 'flex',
      flexWrap: 'wrap',
      alignItems: 'baseline',
      '@media (max-width: 450px)': {
        height: 'auto',
      }
    },
    additionalInfo: {
      '@media (max-width: 450px)': {
        width: '100%',
        marginTop: 4,
        marginLeft: -10,

      }
    },
  }),
)

type MessageHeaderProps = Pick<
  BaseMessageProps,
  'name' | 'id' | 'formattedTime' | 'isPublic' | 'isLoading' | 'aiModel'
> & {
  isAi: boolean;
  toggleDebugDialog: () => void;
  createdAt: BaseMessageProps["created_at"];
};

export const MessageHeader = (props: MessageHeaderProps) => {
  const {isAi, formattedTime, id, name, createdAt, isLoading, aiModel, toggleDebugDialog, isPublic} = props;
  const classes = useStyles();
  return (
    <div className={classes.messageHeader}>
      <span className={classes.messageAuthor}>
        {isAi ? 'Postgres.AI' : name}
      </span>
      {createdAt && formattedTime &&
        <span
          className={cn(classes.messageInfo)}
          title={createdAt}
        >
          {formattedTime}
        </span>
      }
      <div className={classes.additionalInfo}>
        {id && isPublic && <>
          <span className={classes.messageInfo}>|</span>
          <a
            className={cn(classes.messageInfo, classes.messageInfoActive)}
            href={permalinkLinkBuilder(id)}
            target="_blank"
            rel="noreferrer"
          >
            permalink
          </a>
        </>}
        {!isLoading && isAi && id && <>
          <span className={classes.messageInfo}>|</span>
          <button
            className={cn(classes.messageInfo, classes.messageInfoActive)}
            onClick={toggleDebugDialog}
          >
            debug info
          </button>
        </>}
        {
          aiModel && isAi && <>
            <span className={classes.messageInfo}>|</span>
            <span
              className={cn(classes.messageInfo)}
              title={aiModel}
            >
              {aiModel}
            </span>
          </>
        }
      </div>
    </div>
  )
}