/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useRef, useEffect, useState } from 'react';
import { makeStyles, Typography } from "@material-ui/core";
import cn from "classnames";
import { ResizeObserver } from '@juggle/resize-observer';
import {colors} from "@postgres.ai/shared/styles/colors";
import {PageSpinner} from "@postgres.ai/shared/components/PageSpinner";
import { usePrev } from 'hooks/usePrev';
import { getMaxScrollTop, getUserMessagesCount } from './utils';
import Format from "../../../utils/format";
import {BotMessage} from "../../../types/api/entities/bot";
import { Message } from "./Message/Message";

const useStyles = makeStyles(
  (theme) => ({
    root: {
      borderRadius: 4,
      overflow: 'hidden',
      flex: '1 0 160px',
      display: 'flex',
      flexDirection: 'column',
    },
    emptyChat: {
      justifyContent: 'center',
      alignItems: 'center',
      textAlign: 'center'
    },
    emptyChatMessage: {
      maxWidth: '80%',
      fontSize: '0.875rem'
    },
    messages: {
      overflowY: 'auto',
      flex: '1 1 100%'
    },
    message: {
      padding: 10,
      paddingLeft: 60,
      position: 'relative',
      whiteSpace: 'normal',
      [theme.breakpoints.down('xs')]: {
        paddingLeft: 30
      },
      '& .markdown pre': {
        [theme.breakpoints.down('sm')]: {
          display: 'inline-block',
          minWidth: '100%',
          width: 'auto',
        },
        [theme.breakpoints.up('md')]: {
          display: 'block',
          maxWidth: 'auto',
          width: 'auto',
        },
        [theme.breakpoints.up('lg')]: {
          display: 'block',
          maxWidth: 'auto',
          width: 'auto',
        },
      },
    },
    messageAvatar: {
      top: '10px',
      left: '15px',
      position: 'absolute',
      width: 30,
      height: 30,
      [theme.breakpoints.down('xs')]: {
        width: 24,
        height: 24,
        left: 0,
        '& svg': {
          width: 24,
          height: 24,
        }
      }
    },
    messageAvatarImage: {
      width: '100%',
      borderRadius: '50%'
    },
    messageAuthor: {
      fontSize: 14,
      fontWeight: 'bold',
    },
    messageInfo: {
      display: 'inline-block',
      marginLeft: 10,
      fontSize: '0.75rem',
      color: colors.pgaiDarkGray,
      transition: '.2s ease'
    },
    messageInfoActive: {
      '&:hover': {
        color: '#404040'
      }
    },
    messageHeader: {
      height: '1.125rem',
    },
    messagesSpinner: {
      display: 'flex',
      justifyContent: 'center',
      padding: 10
    }
  }),

)

type MessagesProps = {
  messages: BotMessage[] | null
  isLoading: boolean
  isWaitingForAnswer: boolean
}

type Time = string

type FormattedTime = {
  [id: string]: Time
}

export const Messages = (props: MessagesProps) => {
  const {
    messages,
    isLoading,
    isWaitingForAnswer
  } = props;

  const rootRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const atBottomRef = useRef(true);
  const shouldSkipScrollCalcRef = useRef(false);
  const classes = useStyles();
  const [formattedTimes, setFormattedTimes] = useState<FormattedTime>({});

  // Scroll handlers.
  const scrollBottom = () => {
    shouldSkipScrollCalcRef.current = true;
    if (rootRef.current) {
      rootRef.current.scrollTop = getMaxScrollTop(rootRef.current);
    }
    atBottomRef.current = true;
  };

  const scrollBottomIfNeed = () => {
    if (!atBottomRef.current) {
      return;
    }

    scrollBottom();
  };

  // Listening resizing of wrapper.
  useEffect(() => {
    const observedElement = wrapperRef.current;
    if (!observedElement) return;

    const resizeObserver = new ResizeObserver(scrollBottomIfNeed);
    resizeObserver.observe(observedElement);

    return () => resizeObserver.unobserve(observedElement);
  }, [wrapperRef.current]);

  // Scroll to bottom if user sent new message.
  const userMessagesCount = getUserMessagesCount(messages || [] as BotMessage[]);
  const prevUserMessagesCount = usePrev(userMessagesCount);

  useEffect(() => {
    if ((userMessagesCount > (prevUserMessagesCount || 0)) && rootRef.current) {
      scrollBottom();
    }
  }, [prevUserMessagesCount, userMessagesCount]);

  useEffect(() => {
    if (!isLoading) {
      scrollBottomIfNeed();
    }
  }, [isLoading, scrollBottomIfNeed]);

  useEffect(() => {
    const updateTimes = () => {
      if (messages && messages.length > 0) {
        const newFormattedTimes: FormattedTime = {};
        messages.forEach(message => {
          newFormattedTimes[message.id] = Format.timeAgo(message.created_at) || '';
        });
        setFormattedTimes(newFormattedTimes);
      }
    };

    updateTimes();

    const intervalId = setInterval(updateTimes, 60000);

    return () => clearInterval(intervalId);
  }, [messages]);

  // Check auto-scroll condition.
  const calcIsAtBottom = () => {
    if (shouldSkipScrollCalcRef.current) {
      shouldSkipScrollCalcRef.current = false;
      return;
    }
    if (rootRef.current) {
      atBottomRef.current = rootRef.current.scrollTop >= getMaxScrollTop(rootRef.current);
    }
  };

  if (isLoading) {
    return (
      <div className={classes.root}>
        <PageSpinner/>
      </div>
    )
  }

  if (!messages || messages.length === 0) {
    return (
      <div className={cn(classes.root, classes.emptyChat)}>
        <Typography className={classes.emptyChatMessage}>
          Postgres.AI Bot can make mistakes. <br />
          Consider checking important information. <br />
          Depending on settings, LLM service provider such as GCP or OpenAI is used.
        </Typography>
      </div>
    )
  }

  return (
    <div className={classes.root}>
      <div className={classes.messages} ref={rootRef} onScroll={calcIsAtBottom}>
        <div ref={wrapperRef}>
          {messages &&
            Object.keys(messages).map((m) => {
              const idx = Number(m)

              const {
                id,
                is_ai,
                last_name,
                first_name,
                display_name,
                slack_profile,
                created_at,
                content
              } = messages[idx];
              let name = 'You';

              if (first_name || last_name) {
                name = `${first_name || ''} ${last_name || ''}`.trim();
              } else if (display_name) {
                name = display_name;
              } else if (slack_profile) {
                name = slack_profile;
              }

              let formattedTime = '';

              if (formattedTimes) {
                formattedTime = formattedTimes[id]
              }

              return (
                <Message
                  key={id || content}
                  id={id}
                  isAi={is_ai}
                  name={name}
                  created_at={created_at}
                  content={content}
                  formattedTime={formattedTime}
                />
              )
            })}
          {isWaitingForAnswer &&
            <Message id={null} isLoading isAi={true}/>
          }
        </div>
      </div>
    </div>
  );
};
