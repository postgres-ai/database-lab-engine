/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useRef, useEffect } from 'react';
import {
  Button,
  TextField,
  ExpansionPanel,
  ExpansionPanelSummary,
  ExpansionPanelDetails,
  Typography
} from '@material-ui/core';
import { ExpandMore as ExpandMoreIcon } from '@material-ui/icons';
import ReactMarkdown from 'react-markdown';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { ResizeObserver } from '@juggle/resize-observer';

import { icons } from '@postgres.ai/shared/styles/icons';
import { Spinner } from '@postgres.ai/shared/components/Spinner';

import { usePrev } from 'hooks/usePrev';

import { getMaxScrollTop, getMessageArtifactIds, getUserMessagesCount } from './utils';
import { Banner } from './Banner';

import styles from './styles.module.scss';

export const Messages = (props) => {
  const {
    classes,
    messages,
    markdownLink,
    sendCommand,
    loadMessageArtifacts,
    preformatJoeMessageStatus,
    systemMessages
  } = props;

  const rootRef = useRef();
  const wrapperRef = useRef();
  const atBottomRef = useRef(true);
  const shouldSkipScrollCalcRef = useRef(false);

  // Scroll handlers.
  const scrollBottom = () => {
    shouldSkipScrollCalcRef.current = true;
    rootRef.current.scrollTop = getMaxScrollTop(rootRef.current);
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
  const userMessagesCount = getUserMessagesCount(messages);
  const prevUserMessagesCount = usePrev(userMessagesCount);

  if (userMessagesCount > prevUserMessagesCount) {
    scrollBottom();
  }

  // Check auto-scroll condition.
  const calcIsAtBottom = () => {
    if (shouldSkipScrollCalcRef.current) {
      shouldSkipScrollCalcRef.current = false;
      return;
    }

    atBottomRef.current = rootRef.current.scrollTop >= getMaxScrollTop(rootRef.current);
  };

  return (
    <div className={styles.root}>
      <div className={styles.messages} ref={rootRef} onScroll={calcIsAtBottom}>
        <div ref={wrapperRef}>
          {messages &&
            // TODO(Anton): Objects doesn't guarantee keys order.
            Object.keys(messages).map((m) => {
              const msgArtifacts = getMessageArtifactIds(messages[m].message);
              const otherArtifactsExists =
                messages[m].message.indexOf('Other artifacts are provided in the thread') !== -1;

              return (
                <div key={m} className={classes.message} messageid={messages[m].id}>
                  <div className={classes.messageAvatar}>
                    {messages[m].author_id ? icons.userChatIcon : icons.joeChatIcon}
                  </div>
                  <div className={classes.messageHeader}>
                    <span className={classes.messageAuthor}>
                      {messages[m].author_id ? 'You' : 'Joe Bot'}
                    </span>
                    <span className={classes.messageTime}>{messages[m].formattedTime}</span>
                    {(!messages[m].parent_id ||
                      (messages[m].parent_id && messages[messages[m].parent_id])) && (
                      <Button
                        variant='outlined'
                        className={classes.repeatCmdButton}
                        key={m}
                        onClick={() =>
                          sendCommand(
                            messages[m].parent_id
                              ? messages[messages[m].parent_id].message
                              : messages[m].message
                          )
                        }
                      >
                        Repeat
                      </Button>
                    )}
                  </div>
                  {messages[m].parent_id ? (
                    <div>
                      {
                        <ReactMarkdown
                          className='markdown'
                          children={messages[m].formattedMessage}
                          rehypePlugins={[rehypeRaw]}
                          remarkPlugins={[remarkGfm]}
                          linkTarget='_blank'
                          components={{
                            a: (properties) => {
                              return markdownLink(properties, m);
                            },
                            p: 'div'
                          }}
                        />
                      }
                    </div>
                  ) : (
                    <div>
                      <TextField
                        variant='outlined'
                        id={messages[m].id + 'cmd'}
                        multiline
                        fullWidth
                        value={messages[m].message}
                        className={classes.sqlCode}
                        margin='normal'
                        variant='outlined'
                        InputProps={{
                          readOnly: true
                        }}
                      />
                    </div>
                  )}
                  {messages[m].delivery_status === 'artifact_attached' && otherArtifactsExists ? (
                    <div>
                      <ExpansionPanel
                        className={classes.messageArtifacts}
                        TransitionProps={{
                          enter: false,
                          exit: false,
                          appear: false,
                          mountOnEnter: false,
                          timeout: 0
                        }}
                      >
                        <ExpansionPanelSummary
                          expandIcon={<ExpandMoreIcon />}
                          onClick={(event) => {
                            loadMessageArtifacts(event, messages[m].id);
                          }}
                          aria-controls={'message-' + messages[m].id + '-artifacts-content'}
                          id={'message-' + messages[m].id + '-artifacts-header'}
                          className={classes.advancedExpansionPanelSummary}
                        >
                          <Typography className={classes.heading}>
                            <nobr>Other artifacts</nobr>
                          </Typography>
                        </ExpansionPanelSummary>
                        <ExpansionPanelDetails
                          className={classes.messageArtifactExpansionPanelDetails}
                        >
                          {!messages[m].artifacts ||
                          (messages[m].artifacts && messages[m].artifacts.isProcessing) ? (
                              <Spinner />
                            ) : null}

                          {messages[m].artifacts && messages[m].artifacts.files ? (
                            <div className={classes.messageArtifactsContainer}>
                              {Object.keys(messages[m].artifacts.files).map((f) => {
                                let artifactAlreadyExists =
                                  msgArtifacts.indexOf(messages[m].artifacts.files[f].id) !== -1;
                                return !artifactAlreadyExists ? (
                                  <ExpansionPanel
                                    key={messages[m].artifacts.files[f].id}
                                    className={classes.messageArtifact}
                                    TransitionProps={{
                                      enter: false,
                                      exit: false,
                                      appear: false,
                                      mountOnEnter: false,
                                      timeout: 0
                                    }}
                                  >
                                    <ExpansionPanelSummary
                                      expandIcon={<ExpandMoreIcon />}
                                      className={
                                        classes.messageArtifactExpansionPanelSummary
                                      }
                                      aria-controls={
                                        'artifact-' +
                                        messages[m].artifacts.files[f].id +
                                        '-content'
                                      }
                                      id={
                                        'artifact-' +
                                        messages[m].artifacts.files[f].id +
                                        '-header'
                                      }
                                    >
                                      <Typography
                                        className={classes.messageArtifactTitle}
                                      >
                                        {messages[m].artifacts.files[f].title}
                                      </Typography>
                                    </ExpansionPanelSummary>
                                    <ExpansionPanelDetails
                                      className={
                                        classes.advancedExpansionPanelDetails
                                      }
                                    >
                                      <TextField
                                        variant='outlined'
                                        id={
                                          'artifact' +
                                          messages[m].artifacts.files[f].id
                                        }
                                        multiline
                                        fullWidth
                                        value={
                                          messages[m].artifacts.files[f].content
                                        }
                                        className={classes.code}
                                        margin='normal'
                                        variant='outlined'
                                        InputProps={{
                                          readOnly: true
                                        }}
                                      />
                                    </ExpansionPanelDetails>
                                  </ExpansionPanel>
                                ) : null;
                              })}
                            </div>
                          ) : null}
                        </ExpansionPanelDetails>
                      </ExpansionPanel>
                    </div>
                  ) : null}
                  {!messages[m].author_id ? (
                    <div className={classes.messageStatusContainer}>
                      {preformatJoeMessageStatus(messages[m].status)}
                    </div>
                  ) : null}
                </div>
              );
            })}
        </div>
      </div>

      { Boolean(systemMessages.length) && <Banner messages={systemMessages} /> }
    </div>
  );
};
