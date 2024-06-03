/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect, useState } from 'react';
import cn from "classnames";
import {ReadyState} from "react-use-websocket";
import Box from '@mui/material/Box/Box';
import { makeStyles, useMediaQuery } from "@material-ui/core";
import { useHistory, useRouteMatch } from "react-router-dom";
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper';
import { ErrorWrapper } from "../../components/Error/ErrorWrapper";
import { Messages } from './Messages/Messages';
import { Command } from './Command/Command';
import { ChatsList } from "./ChatsList/ChatsList";
import { BotWrapperProps } from "./BotWrapper";
import { useBotChatsList, useAiBot, Model } from "./hooks";
import { usePrev } from "../../hooks/usePrev";
import {HeaderButtons} from "./HeaderButtons/HeaderButtons";
import settings from "../../utils/settings";
import { SaveChangesFunction, SettingsDialog, Visibility } from "./SettingsDialog/SettingsDialog";
import { theme } from "@postgres.ai/shared/styles/theme";
import { colors } from "@postgres.ai/shared/styles/colors";
import { SettingsWithLabel } from "./SettingsWithLabel/SettingsWithLabel";

type BotPageProps = BotWrapperProps;

const useStyles = makeStyles(
  (theme) => ({
    actions: {
      display: 'flex',
      alignItems: 'center',
      alignSelf: 'flex-end',
      marginTop: -20,
      [theme.breakpoints.down('sm')]: {
        marginTop: -22
      }
    },
    hiddenButtons: {
      width: 192,
      marginLeft: 52,
      [theme.breakpoints.down('sm')]: {
        width: 226
      }
    },
    toggleListButton: {
      flex: '0 0 auto',
    },
    contentContainer: {
      height: '100%',
      display: 'flex',
      flexDirection: 'column',
      flexGrow: 1,
      transition: theme.transitions.create('margin', {
        easing: theme.transitions.easing.sharp,
        duration: theme.transitions.duration.leavingScreen,
      }),
      marginRight: 4,
    },
    isChatsListVisible: {
      transition: theme.transitions.create('margin', {
        easing: theme.transitions.easing.easeOut,
        duration: theme.transitions.duration.enteringScreen,
      }),
      marginRight: 244,
      [theme.breakpoints.down('sm')]: {
        marginRight: 0,
      }
    },
    label: {
      backgroundColor: colors.primary.main,
      color: colors.primary.contrastText,
      display: 'inline-block',
      borderRadius: 3,
      fontSize: 10,
      lineHeight: '12px',
      padding: 2,
      paddingLeft: 3,
      paddingRight: 3,
      verticalAlign: 'text-top',
      marginRight: 8
    },
    labelPrivate: {
      backgroundColor: colors.pgaiDarkGray,
    }
  }),
  { index: 1 },
)

export const BotPage = (props: BotPageProps) => {
  const { match, project, orgData } = props;

  const {
    messages,
    loading,
    error,
    sendMessage,
    clearChat,
    wsLoading,
    wsReadyState,
    isChangeVisibilityLoading,
    changeChatVisibility,
    unsubscribe,
    model,
    setModel
  } = useAiBot({
    threadId: match.params.threadId,
  });
  const {chatsList, loading: chatsListLoading, getChatsList} = useBotChatsList(orgData.id);

  const matches = useMediaQuery(theme.breakpoints.down('sm'));

  const [isChatsListVisible, setChatsListVisible] = useState(window?.innerWidth > 640);
  const [isSettingsDialogVisible, setSettingsDialogVisible] = useState(false);
  const [chatVisibility, setChatVisibility] = useState<'public' | 'private'>('public');

  const history = useHistory();

  const prevThreadId = usePrev(match.params.threadId);

  const isDemoOrg = useRouteMatch(`/${settings.demoOrgAlias}`);

  const classes = useStyles();

  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      org={match.params.org}
      project={project}
      breadcrumbs={[
        { name: 'Bot', url: 'bot' },
      ]}
    />
  );

  const toggleChatsList = () => {
    setChatsListVisible((prevState) => !prevState)
  }

  const toggleSettingsDialog = () => {
    setSettingsDialogVisible((prevState) => !prevState)
  }

  const handleSendMessage = async (message: string) => {
    const { threadId } = match.params;
    const orgId = orgData.id || null;

    await sendMessage({
      content: message,
      thread_id: threadId || null,
      org_id: orgId,
      is_public: chatVisibility === 'public'
    })
  }

  const handleCreateNewChat = () => {
    clearChat();
    history.push(`/${match.params.org}/bot`);
  }

  const handleSaveChatVisibility = (value: boolean) => {
    if (match.params.threadId) {
      changeChatVisibility(match.params.threadId, value)
      getChatsList();
    }
  }

  const handleSaveSettings: SaveChangesFunction = ( _model, _visibility) => {
    if (_model !== model) {
      setModel(_model)
    }
    if (_visibility !== chatVisibility) {
      handleSaveChatVisibility( _visibility === 'public')
    }
    toggleSettingsDialog();
  }

  const handleChatListLinkClick = (targetThreadId: string) => {
    if (match.params.threadId && match.params.threadId !== targetThreadId) {
      unsubscribe(match.params.threadId)
    }
  }

  useEffect(() => {
    if (!match.params.threadId && !prevThreadId && messages && messages.length > 1 && messages[1].parent_id) {
      // hack that skip additional loading chats_ancestors_and_descendants
      history.replace(`/${match.params.org}/bot/${messages[1].parent_id}`, { skipReloading: true })
      getChatsList();
    } else if (prevThreadId && !match.params.threadId) {
      clearChat()
    }
  }, [match.params.threadId, match.params.org, messages, prevThreadId]);

  useEffect(() => {
    if (messages && messages.length > 0 && match.params.threadId) {
      setChatVisibility(messages[0].is_public ? 'public' : 'private')
    }
  }, [messages]);

  useEffect(() => {
    // fixes hack with skipping additional loading chats_ancestors_and_descendants
    history.replace({ state: {} })
  }, []);

  if (error && error.code === 404) {
    return (
      <>
        {breadcrumbs}
        <ErrorWrapper
          message={error.message}
          code={error.code}
        />
      </>
    )
  }

  return (
    <>
      <SettingsDialog
        defaultVisibility={chatVisibility}
        defaultModel={model}
        isOpen={isSettingsDialogVisible}
        isLoading={isChangeVisibilityLoading}
        onClose={toggleSettingsDialog}
        onSaveChanges={handleSaveSettings}
        threadId={match.params.threadId || null}
      />
      <ChatsList
        isOpen={isChatsListVisible}
        onCreateNewChat={handleCreateNewChat}
        onClose={toggleChatsList}
        isDemoOrg={Boolean(isDemoOrg)}
        chatsList={chatsList}
        loading={chatsListLoading}
        withChatVisibilityButton={matches && Boolean(match.params.threadId)}
        onChatVisibilityClick={toggleSettingsDialog}
        currentVisibility={chatVisibility}
        onLinkClick={handleChatListLinkClick}
        permalinkId={messages?.[0]?.id}
      />
      <Box className={classes.actions}>
        {!matches &&
          <SettingsWithLabel
            chatVisibility={chatVisibility}
            onSettingsClick={toggleSettingsDialog}
            permalinkId={messages?.[0]?.id}
          />}
        <Box className={classes.hiddenButtons}>
          <HeaderButtons
            isOpen={isChatsListVisible}
            onClose={toggleChatsList}
            onCreateNewChat={handleCreateNewChat}
            withChatVisibilityButton={matches && Boolean(match.params.threadId)}
            onChatVisibilityClick={toggleSettingsDialog}
            currentVisibility={chatVisibility}
            permalinkId={messages?.[0]?.id}
          />
        </Box>
      </Box>
      <Box className={cn(classes.contentContainer, {[classes.isChatsListVisible]: isChatsListVisible})}>
        <Messages
          messages={messages}
          isLoading={loading}
          isWaitingForAnswer={wsLoading}
        />

        <Command
          sendDisabled={error !== null || loading || wsLoading || wsReadyState !== ReadyState.OPEN}
          onSend={handleSendMessage}
          threadId={match.params.threadId}
        />
      </Box>
    </>
  )
}