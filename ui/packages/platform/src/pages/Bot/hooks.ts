/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {useCallback, useEffect, useRef, useState} from "react";
import useWebSocket, {ReadyState} from "react-use-websocket";
import { useLocation } from "react-router-dom";
import {BotMessage} from "../../types/api/entities/bot";
import {getChatsWithWholeThreads} from "../../api/bot/getChatsWithWholeThreads";
import {getChats} from "api/bot/getChats";
import {useAlertSnackbar} from "@postgres.ai/shared/components/AlertSnackbar/useAlertSnackbar";
import {localStorage} from "../../helpers/localStorage";
import { makeChatPublic } from "../../api/bot/makeChatPublic";


const WS_URL = process.env.REACT_APP_WS_URL || '';

type ErrorType = {
  code?: number;
  message: string;
  type?: 'connection' | 'chatNotFound';
}

type sendMessageType = {
  content: string;
  thread_id?: string | null;
  org_id?: number | null;
  is_public?: boolean;
}

type UseAiBotReturnType = {
  messages: BotMessage[] | null;
  error: ErrorType | null;
  loading: boolean;
  sendMessage: (args: sendMessageType) => Promise<void>;
  clearChat: () => void;
  wsLoading: boolean;
  wsReadyState: ReadyState;
  changeChatVisibility: (threadId: string, isPublic: boolean) => void;
  isChangeVisibilityLoading: boolean;
  unsubscribe: (threadId: string) => void
}

type UseAiBotArgs = {
  threadId?: string;
  prevThreadId?: string;
  onChatLoadingError?: () => void;
}

export const useAiBot = (args: UseAiBotArgs): UseAiBotReturnType => {
  const { threadId, onChatLoadingError } = args;
  const { showMessage, closeSnackbar } = useAlertSnackbar();
  let location = useLocation<{skipReloading?: boolean}>();

  const [messages, setMessages] = useState<BotMessage[] | null>(null);
  const [isLoading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<ErrorType | null>(null);
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [isChangeVisibilityLoading, setIsChangeVisibilityLoading] = useState<boolean>(false)
  
  const token = localStorage.getAuthToken()

  const onWebSocketError = (error: WebSocketEventMap['error']) => {
    console.error('WebSocket error:', error);
    showMessage('WebSocket connection error: attempting to reconnect');
  }

  const onWebSocketMessage = (event: WebSocketEventMap['message']) => {
    if (event.data) {
      const messageData: BotMessage = JSON.parse(event.data);
      if (messageData) {
        if ((threadId && threadId === messageData.thread_id) || (!threadId && messageData.parent_id && messages)) {
          let currentMessages = [...(messages || [])];
          // Check if the last message needs its data updated
          const lastMessage = currentMessages[currentMessages.length - 1];
          if (lastMessage && !lastMessage.id && messageData.parent_id) {
            lastMessage.id = messageData.parent_id;
            lastMessage.created_at = messageData.created_at;
            lastMessage.is_public = messageData.is_public;
          }

          currentMessages.push(messageData);
          setMessages(currentMessages);
          if (document.visibilityState === "hidden") {
            if (Notification.permission === "granted") {
              new Notification("New message", {
                body: 'New message from Postgres.AI Bot',
                icon: '/images/bot_avatar.png'
              });
            }
          }
        }
      } else {
        showMessage('An error occurred. Please try again')
      }
    } else {
      showMessage('An error occurred. Please try again')
    }
    setWsLoading(false);
    setLoading(false);
  }

  const onWebSocketOpen = () => {
    console.log('WebSocket connection established');
    if (threadId) {
      subscribe(threadId)
    }
    setWsLoading(false);
    setLoading(false);
    closeSnackbar();
  }
  const onWebSocketClose = (event: WebSocketEventMap['close']) => {
    console.log('WebSocket connection closed', event);
    showMessage('WebSocket connection error: attempting to reconnect');
  }

  const { sendMessage: wsSendMessage, readyState, } = useWebSocket(WS_URL, {
    protocols: ['Authorization', token || ''],
    shouldReconnect: () => true,
    reconnectAttempts: 50,
    reconnectInterval: 5000, // ms
    onError: onWebSocketError,
    onMessage: onWebSocketMessage,
    onClose: onWebSocketClose,
    onOpen: onWebSocketOpen
  })

  const getChatMessages = useCallback(async (threadId: string) => {
    setError(null);
    if (threadId) {
      setLoading(true);
      try {
        const { response} = await getChatsWithWholeThreads({id: threadId});
        subscribe(threadId)
        if (response && response.length > 0) {
          setMessages(response);
        } else {
          if (onChatLoadingError) onChatLoadingError();
          setError({
            code: 404,
            message: 'Specified chat not found or you have no access.',
            type: 'chatNotFound'
          })
        }
      } catch (e) {
        setError(e as unknown as ErrorType)
        showMessage('Connection error')
      } finally {
        setLoading(false);
      }
    }
  }, []);

  useEffect(() => {
    let isCancelled = false;
    setError(null);
    setWsLoading(false);

    if (threadId && !location.state?.skipReloading) {
      getChatMessages(threadId)
        .catch((e) => {
          if (!isCancelled) {
            setError(e);
          }
        });
    } else if (threadId) {
      subscribe(threadId)
    }

    return () => {
      isCancelled = true;
    };
  }, [getChatMessages, location.state?.skipReloading, threadId]);

  useEffect(() => {
    const fetchData = async () => {
      if (threadId) {
        const { response } = await getChatsWithWholeThreads({id: threadId});
        if (response && response.length > 0) {
          setMessages(response);
        }
      }
    };

    let intervalId: NodeJS.Timeout | null = null;

    if (readyState === ReadyState.CLOSED) {
      intervalId = setInterval(fetchData, 20000);
    }

    return () => {
      if (intervalId) {
        clearInterval(intervalId);
      }
    };
  }, [readyState, threadId]);

  const sendMessage = async ({content, thread_id, org_id, is_public}: sendMessageType) => {
    setWsLoading(true)
    if (!thread_id) {
      setLoading(true)
    }
    try {
      //TODO: fix it
      if (messages && messages.length > 0) {
        setMessages((prevState) => [...prevState as BotMessage[], { content, is_ai: false, created_at: new Date().toISOString() } as BotMessage])
      } else {
        setMessages([{ content, is_ai: false, created_at: new Date().toISOString() } as BotMessage])
      }
      wsSendMessage(JSON.stringify({
        action: 'send',
        payload: {
          content,
          thread_id,
          org_id,
          is_public
        }
      }))
      setError(error)

    } catch (e) {
      setError(e as unknown as ErrorType)
    } finally {
      setLoading(false)
    }
  }

  const clearChat = () => {
    setMessages(null);
  }

  const changeChatVisibility = async (threadId: string, isPublic: boolean) => {
    setIsChangeVisibilityLoading(true)
    try {
      const { error } = await makeChatPublic({
        thread_id: threadId,
        is_public: isPublic
      })
      if (error) {
        showMessage('Failed to change chat visibility. Please try again later.')
      } else if (messages?.length) {
        const newMessages: BotMessage[] = messages.map((message) => ({
          ...message,
          is_public: isPublic
        }))
        setMessages(newMessages)
      }
    } catch (e) {
      showMessage('Failed to change chat visibility. Please try again later.')
    } finally {
      setIsChangeVisibilityLoading(false)
    }
  }

  const subscribe = (threadId: string) => {
    wsSendMessage(JSON.stringify({
      action: 'subscribe',
      payload: {
        thread_id: threadId,
      }
    }))
  }

  const unsubscribe = (threadId: string) => {
    wsSendMessage(JSON.stringify({
      action: 'unsubscribe',
      payload: {
        thread_id: threadId,
      }
    }))
  }

  useEffect(() => {
    if ('Notification' in window) {
      Notification.requestPermission().then(permission => {
        if (permission === "granted") {
          console.log("Permission for notifications granted");
        } else {
          console.log("Permission for notifications denied");
        }
      });
    }
  }, [])

  return {
    error: error,
    wsLoading: wsLoading,
    wsReadyState: readyState,
    loading: isLoading,
    changeChatVisibility,
    isChangeVisibilityLoading,
    sendMessage,
    clearChat,
    messages,
    unsubscribe
  }
}

type UseBotChatsListHook = {
  chatsList: BotMessage[] | null;
  error: Response | null;
  loading: boolean;
  getChatsList: () => void
};

export const useBotChatsList = (orgId?: number): UseBotChatsListHook => {
  const [chatsList, setChatsList] = useState<BotMessage[] | null>(null);
  const [isLoading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<Response | null>(null)

  const getChatsList = useCallback(async () => {
    setLoading(true);
    try {
      const queryString = `?parent_id=is.null&org_id=eq.${orgId}`;
      const { response, error } = await getChats({ query: queryString });

      setChatsList(response);
      setError(error)

    } catch (e) {
      setError(e as unknown as Response)
    } finally {
      setLoading(false)
    }
  }, []);

  useEffect(() => {
    let isCancelled = false;

    getChatsList()
      .catch((e) => {
        if (!isCancelled) {
          setError(e);
        }
      });

    return () => {
      isCancelled = true;
    };
  }, [getChatsList]);

  return {
    chatsList,
    error,
    getChatsList,
    loading: isLoading
  }
}