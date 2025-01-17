/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { useEffect, useMemo, useState } from 'react';
import cn from "classnames";
import Box from '@mui/material/Box/Box';
import { makeStyles, Typography, useMediaQuery } from "@material-ui/core";
import { useHistory, useRouteMatch } from "react-router-dom";
import { ConsoleBreadcrumbsWrapper } from 'components/ConsoleBreadcrumbs/ConsoleBreadcrumbsWrapper';
import { ErrorWrapper } from "../../components/Error/ErrorWrapper";
import { Messages } from './Messages/Messages';
import { Command } from './Command/Command';
import { ChatsList } from "./ChatsList/ChatsList";
import { BotWrapperProps } from "./BotWrapper";
import { useAiBot } from "./hooks";
import { usePrev } from "../../hooks/usePrev";
import {HeaderButtons} from "./HeaderButtons/HeaderButtons";
import settings from "../../utils/settings";
import { SettingsDialog } from "./SettingsDialog/SettingsDialog";
import { theme } from "@postgres.ai/shared/styles/theme";
import { colors } from "@postgres.ai/shared/styles/colors";
import { SettingsPanel } from "./SettingsPanel/SettingsPanel";
import { DebugConsole } from "./DebugConsole/DebugConsole";
import Store from "../../stores/store";
import Actions from "../../actions/actions";
import { Link } from "@postgres.ai/shared/components/Link2";

type BotPageProps = BotWrapperProps;

type DbLabInstance = {
  id: number;
  plan: string | null;
}

type RefluxState = {
  dbLabInstances: {
    data: Record<number, DbLabInstance>;
  }
} | null

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
        width: 'min(100%, 320px)',
        marginLeft: 'auto'
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
    },
    publicChatNote: {
      marginTop: 4,
      marginBottom: -14,
      alignSelf: 'center',
      fontSize: 12
    }
  }),
  { index: 1 },
)

export const BotPage = (props: BotPageProps) => {
  const { match, project, orgData, orgId } = props;

  const {
    messages,
    error,
    clearChat,
    unsubscribe,
    getDebugMessagesForWholeThread,
    getChatsList,
    chatVisibility
  } = useAiBot();

  const matches = useMediaQuery(theme.breakpoints.down('sm'));

  const [isChatsListVisible, setChatsListVisible] = useState(window?.innerWidth > 640);
  const [isSettingsDialogVisible, setSettingsDialogVisible] = useState(false);
  const [isDebugConsoleVisible, setDebugConsoleVisible] = useState(false);

  const history = useHistory();

  const prevThreadId = usePrev(match.params.threadId);

  const isDemoOrg = useRouteMatch(`/${settings.demoOrgAlias}`);

  const classes = useStyles();

  const breadcrumbs = (
    <ConsoleBreadcrumbsWrapper
      org={match.params.org}
      project={project}
      breadcrumbs={[
        { name: 'Assistant', url: 'assistant' },
      ]}
    />
  );

  const toggleChatsList = () => {
    setChatsListVisible((prevState) => !prevState)
  }

  const toggleSettingsDialog = () => {
    setSettingsDialogVisible((prevState) => !prevState)
  }

  const handleOpenDebugConsole = () => {
    setDebugConsoleVisible(true);
    getDebugMessagesForWholeThread()
  }

  const toggleDebugConsole = () => {
    setDebugConsoleVisible((prevState) => !prevState)
  }

  const handleCreateNewChat = () => {
    clearChat();
    history.push(`/${match.params.org}/assistant`);
  }

  const handleChatListLinkClick = (targetThreadId: string) => {
    if (match.params.threadId && match.params.threadId !== targetThreadId) {
      unsubscribe(match.params.threadId)
    }
  }

  const isSubscriber = useMemo(() => orgData?.chats_private_allowed, [orgData?.chats_private_allowed]);

  const publicChatMessage = useMemo(() => {
    if (isDemoOrg) {
      return <>AI can make mistakes❗️ All chats here are currently public. To enable private conversations, <Link to="/addorg">create your organization</Link></>;
    }

    if (!isSubscriber) {
      return <>AI can make mistakes❗️ All chats here are currently public. <Link to={`/${match.params.org}/instances`}>Connect DBLab SE</Link> or <Link external to="https://postgres.ai/consulting">become a consulting client</Link> to enable private conversations</>;
    }

    if (isSubscriber && chatVisibility === 'public') {
      return <>AI can make mistakes❗️ This chat is public, you can change it in Settings.</>
    }

    return null;
  }, [isDemoOrg, match.params.org, chatVisibility, isSubscriber]);

  useEffect(() => {
    if (!match.params.threadId && !prevThreadId && messages && messages.length > 0 && messages[0].id) {
      // hack that skip additional loading chats_ancestors_and_descendants
      history.replace(`/${match.params.org}/assistant/${messages[0].id}`, { skipReloading: true })
      getChatsList();
    } else if (prevThreadId && !match.params.threadId) {
      clearChat()
    }
  }, [match.params.threadId, match.params.org, messages, prevThreadId]);

  useEffect(() => {
    // fixes hack with skipping additional loading chats_ancestors_and_descendants
    history.replace({ state: {} })
  }, []);

  useEffect(() => {
    if (isDebugConsoleVisible) setDebugConsoleVisible(false)
  }, [match.params.threadId]);


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
      <DebugConsole
        onClose={toggleDebugConsole}
        isOpen={isDebugConsoleVisible}
        threadId={match.params.threadId}
      />
      <SettingsDialog
        isOpen={isSettingsDialogVisible}
        onClose={toggleSettingsDialog}
        threadId={match.params.threadId || null}
        orgAlias={match.params.org || ''}
        isSubscriber={isSubscriber}
      />
      <ChatsList
        isOpen={isChatsListVisible}
        onCreateNewChat={handleCreateNewChat}
        onClose={toggleChatsList}
        isDemoOrg={Boolean(isDemoOrg)}
        withChatVisibilityButton={matches && Boolean(match.params.threadId)}
        onSettingsClick={toggleSettingsDialog}
        onLinkClick={handleChatListLinkClick}
        onConsoleClick={handleOpenDebugConsole}
      />
      <Box className={classes.actions}>
        {!matches &&
          <SettingsPanel
            onSettingsClick={toggleSettingsDialog}
            onConsoleClick={handleOpenDebugConsole}
          />}
        <Box className={classes.hiddenButtons}>
          <HeaderButtons
            isOpen={isChatsListVisible}
            onClose={toggleChatsList}
            onCreateNewChat={handleCreateNewChat}
            withChatVisibilityButton={matches && Boolean(match.params.threadId)}
            onSettingsClick={toggleSettingsDialog}
            onConsoleClick={handleOpenDebugConsole}
          />
        </Box>
      </Box>
      <Box className={cn(classes.contentContainer, {[classes.isChatsListVisible]: isChatsListVisible})}>
        <Messages orgId={orgData.id} threadId={match.params.threadId} />
        <Command
          threadId={match.params.threadId}
          orgId={orgData.id}
        />
        <Typography className={classes.publicChatNote}>{publicChatMessage}</Typography>
      </Box>
    </>
  )
}