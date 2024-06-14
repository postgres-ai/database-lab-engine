/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { createContext, useCallback, useContext, useEffect, useState } from "react";
import useWebSocket, {ReadyState} from "react-use-websocket";
import { useLocation } from "react-router-dom";
import { BotMessage, DebugMessage, LLMModel } from "../../types/api/entities/bot";
import {getChatsWithWholeThreads} from "../../api/bot/getChatsWithWholeThreads";
import {getChats} from "api/bot/getChats";
import {useAlertSnackbar} from "@postgres.ai/shared/components/AlertSnackbar/useAlertSnackbar";
import {localStorage} from "../../helpers/localStorage";
import { makeChatPublic } from "../../api/bot/makeChatPublic";
import { getLLMModels } from "../../api/bot/getLLMModels";
import { getDebugMessages } from "../../api/bot/getDebugMessages";


const WS_URL = process.env.REACT_APP_WS_URL || '';

export type Model = `${LLMModel['vendor']}/${LLMModel['name']}`

const DEFAULT_MODEL_NAME = 'gemini-1.5-pro'
const DEFAULT_VENDOR = 'gcp';

export const DEFAULT_MODEL: Model = `${DEFAULT_VENDOR}/${DEFAULT_MODEL_NAME}`

type ErrorType = {
  code?: number;
  message: string;
  type?: 'connection' | 'chatNotFound';
}

type SendMessageType = {
  content: string;
  thread_id?: string | null;
  org_id?: number | null;
  is_public?: boolean;
}

type UseAiBotReturnType = {
  messages: BotMessage[] | null;
  error: ErrorType | null;
  loading: boolean;
  sendMessage: (args: SendMessageType) => Promise<void>;
  clearChat: () => void;
  wsLoading: boolean;
  wsReadyState: ReadyState;
  changeChatVisibility: (threadId: string, isPublic: boolean) => void;
  isChangeVisibilityLoading: boolean;
  unsubscribe: (threadId: string) => void
  chatVisibility: 'public' | 'private'
  debugMessages: DebugMessage[] | null
  getDebugMessagesForWholeThread: () => void
  chatsList: UseBotChatsListHook['chatsList'];
  chatsListLoading: UseBotChatsListHook['loading'];
  getChatsList: UseBotChatsListHook['getChatsList']
  model: UseLLMModelsList['model'],
  setModel: UseLLMModelsList['setModel']
  llmModels: UseLLMModelsList['llmModels']
  debugMessagesLoading: boolean;
}

type UseAiBotArgs = {
  threadId?: string;
  orgId?: number
}

export const useAiBotProviderValue = (args: UseAiBotArgs): UseAiBotReturnType => {
  const { threadId, orgId } = args;
  const { showMessage, closeSnackbar } = useAlertSnackbar();
  const { llmModels, model, setModel } = useLLMModelsList();
  let location = useLocation<{skipReloading?: boolean}>();

  const {
    chatsList,
    loading: chatsListLoading,
    getChatsList,
  } = useBotChatsList(orgId);

  const [messages, setMessages] = useState<BotMessage[] | null>(null);
  const [debugMessages, setDebugMessages] = useState<DebugMessage[] | null>(null);
  const [debugMessagesLoading, setDebugMessagesLoading] = useState<boolean>(false);
  const [isLoading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<ErrorType | null>(null);
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [chatVisibility, setChatVisibility] = useState<UseAiBotReturnType['chatVisibility']>('public');

  const [isChangeVisibilityLoading, setIsChangeVisibilityLoading] = useState<boolean>(false);
  
  const token = localStorage.getAuthToken()

  const onWebSocketError = (error: WebSocketEventMap['error']) => {
    console.error('WebSocket error:', error);
    showMessage('WebSocket connection error: attempting to reconnect');
  }

  const onWebSocketMessage = (event: WebSocketEventMap['message']) => {
    if (event.data) {
      const messageData: BotMessage | DebugMessage = JSON.parse(event.data);
      if (messageData) {
        const isThreadMatching = threadId && threadId === messageData.thread_id;
        const isParentMatching = !threadId && 'parent_id' in messageData && messageData.parent_id && messages;
        const isDebugMessage = messageData.type === 'debug';
        if (isThreadMatching || isParentMatching || isDebugMessage) {
          if (isDebugMessage) {
            let currentDebugMessages = [...(debugMessages || [])];
            currentDebugMessages.push(messageData)
            setDebugMessages(currentDebugMessages)
          } else {
            // Check if the last message needs its data updated
            let currentMessages = [...(messages || [])];
            const lastMessage = currentMessages[currentMessages.length - 1];
            if (lastMessage && !lastMessage.id && messageData.parent_id) {
              lastMessage.id = messageData.parent_id;
              lastMessage.created_at = messageData.created_at;
              lastMessage.is_public = messageData.is_public;
            }

            currentMessages.push(messageData);
            setMessages(currentMessages);
            setWsLoading(false);
            if (document.visibilityState === "hidden") {
              if (Notification.permission === "granted") {
                new Notification("New message", {
                  body: 'New message from Postgres.AI Bot',
                  icon: '/images/bot_avatar.png'
                });
              }
            }
          }
        } else if (threadId !== messageData.thread_id) {
          const threadInList = chatsList?.find((item) => item.thread_id === messageData.thread_id)
          if (!threadInList) getChatsList()
          setWsLoading(false);
        }
      } else {
        showMessage('An error occurred. Please try again')
      }
    } else {
      showMessage('An error occurred. Please try again')
    }

    setLoading(false);
  }

  const onWebSocketOpen = () => {
    console.log('WebSocket connection established');
    if (threadId) {
      subscribe(threadId)
    }
    setWsLoading(false);
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
    setDebugMessages(null)
    if (threadId) {
      setLoading(true);
      try {
        const { response } = await getChatsWithWholeThreads({id: threadId});
        subscribe(threadId)
        if (response && response.length > 0) {
          setMessages(response);
        } else {
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

  const sendMessage = async ({content, thread_id, org_id, is_public}: SendMessageType) => {
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
          is_public,
          ai_model: model
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
    setDebugMessages(null);
    setWsLoading(false);
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

  const getDebugMessagesForWholeThread = async () => {
    setDebugMessagesLoading(true)
    if (threadId) {
      const { response } = await getDebugMessages({thread_id: threadId})
      if (response) {
        setDebugMessages(response)
      }
    }
    setDebugMessagesLoading(false)
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

  useEffect(() => {
    if (messages && messages.length > 0 && threadId) {
      setChatVisibility(messages[0].is_public ? 'public' : 'private')
    }
  }, [messages]);

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
    getDebugMessagesForWholeThread,
    unsubscribe,
    chatsList,
    chatsListLoading,
    getChatsList,
    model,
    setModel,
    llmModels,
    chatVisibility,
    debugMessages,
    debugMessagesLoading
  }
}

type AiBotContextType = UseAiBotReturnType;

const AiBotContext = createContext<AiBotContextType | undefined>(undefined);

type AiBotProviderProps = {
  children: React.ReactNode;
  args: UseAiBotArgs;
};

export const AiBotProvider = ({ children, args }: AiBotProviderProps) => {
  const aiBot = useAiBotProviderValue(args);
  return (
    <AiBotContext.Provider value={aiBot}>
      {children}
      </AiBotContext.Provider>
  );
};

export const useAiBot = (): AiBotContextType => {
  const context = useContext(AiBotContext);
  if (context === undefined) {
    throw new Error('useAiBotContext must be used within an AiBotProvider');
  }
  return context;
};

type UseBotChatsListHook = {
  chatsList: BotMessage[] | null;
  error: Response | null;
  loading: boolean;
  getChatsList: () => void;
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

type UseLLMModelsList = {
  llmModels: LLMModel[] | null
  error: Response | null
  model: Model
  setModel: (model: Model) => void
}

const useLLMModelsList = (): UseLLMModelsList => {
  const [llmModels, setLLMModels] = useState<UseLLMModelsList['llmModels']>(null);
  const [error, setError] = useState<Response | null>(null);
  const [userModel, setUserModel] = useState<Model | null>(null)

  const getModels = useCallback(async () => {
    let models = null;
    try {
      const { response } = await getLLMModels();
      setLLMModels(response)
      const currentModel = window.localStorage.getItem('bot.llm_model')

      if (currentModel && currentModel !== userModel) {
        setUserModel(currentModel as Model)
      } else if (!currentModel) {
        setModel(DEFAULT_MODEL)
      }
    } catch (e) {
      setError(e as unknown as Response)
    }
    return models
  }, []);

  useEffect(() => {
    let isCancelled = false;

    getModels()
      .catch((e) => {
        if (!isCancelled) {
          setError(e);
        }
      });
    return () => {
      isCancelled = true;
    };
  }, [getModels]);

  const setModel = (model: Model) => {
    if (model !== userModel) {
      setUserModel(model);
      window.localStorage.setItem('bot.llm_model', model)
    }
  }

  return {
    llmModels,
    error,
    setModel,
    model: userModel || DEFAULT_MODEL,
  }
}