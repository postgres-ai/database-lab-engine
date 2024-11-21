/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React, { createContext, Dispatch, SetStateAction, useCallback, useContext, useEffect, useState } from "react";
import useWebSocket, {ReadyState} from "react-use-websocket";
import { useLocation } from "react-router-dom";
import {
  BotMessage,
  DebugMessage,
  AiModel,
  StateMessage,
  StreamMessage,
  ErrorMessage
} from "../../types/api/entities/bot";
import {getChatsWithWholeThreads} from "../../api/bot/getChatsWithWholeThreads";
import {getChats} from "api/bot/getChats";
import {useAlertSnackbar} from "@postgres.ai/shared/components/AlertSnackbar/useAlertSnackbar";
import {localStorage} from "../../helpers/localStorage";
import { updateChatVisibility } from "../../api/bot/updateChatVisibility";
import { getAiModels } from "../../api/bot/getAiModels";
import { getDebugMessages } from "../../api/bot/getDebugMessages";


const WS_URL = process.env.REACT_APP_WS_URL || '';

const DEFAULT_MODEL_NAME = 'gpt-4o-mini'

export enum Visibility {
  PUBLIC = 'public',
  PRIVATE = 'private'
}

type ErrorType = {
  code?: number;
  message: string;
  type?: 'connection' | 'chatNotFound';
}

type SendMessageType = {
  content: string;
  thread_id?: string | null;
  org_id?: number | null;
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
  unsubscribe: (threadId: string) => void;
  chatVisibility: Visibility;
  setChatVisibility: Dispatch<SetStateAction<Visibility>>;
  debugMessages: DebugMessage[] | null;
  getDebugMessagesForWholeThread: () => void;
  chatsList: UseBotChatsListHook['chatsList'];
  chatsListLoading: UseBotChatsListHook['loading'];
  getChatsList: UseBotChatsListHook['getChatsList'];
  aiModel: UseAiModelsList['aiModel'];
  setAiModel: UseAiModelsList['setAiModel'];
  aiModels: UseAiModelsList['aiModels'];
  aiModelsLoading: UseAiModelsList['loading'];
  debugMessagesLoading: boolean;
  stateMessage: StateMessage | null;
  isStreamingInProcess: boolean;
  currentStreamMessage: StreamMessage | null;
  errorMessage: ErrorMessage | null;
}

type UseAiBotArgs = {
  threadId?: string;
  orgId?: number
  isPublicByDefault?: boolean
}

export const useAiBotProviderValue = (args: UseAiBotArgs): UseAiBotReturnType => {
  const { threadId, orgId, isPublicByDefault } = args;
  const { showMessage, closeSnackbar } = useAlertSnackbar();
  const {
    aiModels,
    aiModel,
    setAiModel,
    loading: aiModelsLoading
  } = useAiModelsList();
  let location = useLocation<{skipReloading?: boolean}>();

  const {
    chatsList,
    loading: chatsListLoading,
    getChatsList,
  } = useBotChatsList(orgId);

  const [messages, setMessages] = useState<BotMessage[] | null>(null);
  const [errorMessage, setErrorMessage] = useState<ErrorMessage | null>(null)
  const [debugMessages, setDebugMessages] = useState<DebugMessage[] | null>(null);
  const [debugMessagesLoading, setDebugMessagesLoading] = useState<boolean>(false);
  const [isLoading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<ErrorType | null>(null);
  const [wsLoading, setWsLoading] = useState<boolean>(false);
  const [chatVisibility, setChatVisibility] = useState<UseAiBotReturnType['chatVisibility']>(Visibility.PUBLIC);
  const [stateMessage, setStateMessage] = useState<StateMessage | null>(null)
  const [currentStreamMessage, setCurrentStreamMessage] = useState<StreamMessage | null>(null)
  const [isStreamingInProcess, setStreamingInProcess] = useState<boolean>(false)

  const [isChangeVisibilityLoading, setIsChangeVisibilityLoading] = useState<boolean>(false);
  
  const token = localStorage.getAuthToken()

  const onWebSocketError = (error: WebSocketEventMap['error']) => {
    console.error('WebSocket error:', error);
    showMessage('WebSocket connection error: attempting to reconnect');
  }

  const onWebSocketMessage = (event: WebSocketEventMap['message']) => {
    if (event.data) {
      const messageData: BotMessage | DebugMessage | StateMessage | StreamMessage | ErrorMessage = JSON.parse(event.data);
      if (messageData) {
        const isThreadMatching = threadId && threadId === messageData.thread_id;
        const isParentMatching = !threadId && 'parent_id' in messageData && messageData.parent_id && messages;
        const isDebugMessage = messageData.type === 'debug';
        const isStateMessage = messageData.type === 'state';
        const isStreamMessage = messageData.type === 'stream';
        const isErrorMessage = messageData.type === 'error';

        if (isThreadMatching || isParentMatching || isDebugMessage || isStateMessage || isStreamMessage || isErrorMessage) {
          switch (messageData.type) {
            case 'debug':
              handleDebugMessage(messageData)
              break;
            case 'state':
              handleStateMessage(messageData, Boolean(isThreadMatching))
              break;
            case 'stream':
              handleStreamMessage(messageData, Boolean(isThreadMatching))
              break;
            case 'message':
              handleBotMessage(messageData)
              break;
            case 'error':
              handleErrorMessage(messageData)
              break;
          }
        } else if (threadId !== messageData.thread_id) {
          const threadInList = chatsList?.find((item) => item.thread_id === messageData.thread_id)
          if (!threadInList) getChatsList()
          if (currentStreamMessage) setCurrentStreamMessage(null)
          if (wsLoading) setWsLoading(false);
          if (isStreamingInProcess) setStreamingInProcess(false)
        }
      } else {
        showMessage('An error occurred. Please try again')
      }
    } else {
      showMessage('An error occurred. Please try again')
    }

    setLoading(false);
  }

  const handleDebugMessage = (message: DebugMessage) => {
    let currentDebugMessages = [...(debugMessages || [])];
    currentDebugMessages.push(message)
    setDebugMessages(currentDebugMessages)
  }

  const handleStateMessage = (message: StateMessage, isThreadMatching?: boolean) => {
    if (isThreadMatching || !threadId) {
      if (message.state) {
        setStateMessage(message)
      } else {
        setStateMessage(null)
      }
    }
  }

  const handleStreamMessage = (message: StreamMessage, isThreadMatching?: boolean) => {
    if (isThreadMatching || !threadId) {
      if (!isStreamingInProcess) setStreamingInProcess(true)
      setCurrentStreamMessage(message)
      setWsLoading(false);
    }
  }

  const handleBotMessage = (message: BotMessage) => {
    if (messages && messages.length > 0) {
      let currentMessages = [...messages];
      const lastMessage = currentMessages[currentMessages.length - 1];
      if (lastMessage && !lastMessage.id && message.parent_id) {
        lastMessage.id = message.parent_id;
        lastMessage.created_at = message.created_at;
        lastMessage.is_public = message.is_public;
      }

      currentMessages.push(message);
      if (currentStreamMessage) setCurrentStreamMessage(null)
      setMessages(currentMessages);
      setWsLoading(false);
      setStreamingInProcess(false);
      if (document.visibilityState === "hidden") {
        if (Notification.permission === "granted") {
          new Notification("New message", {
            body: 'New message from Postgres.AI Assistant',
            icon: '/images/bot_avatar.png'
          });
        }
      }
    }
  }

  const handleErrorMessage = (message: ErrorMessage) => {
    if (message && message.message) {
      let error = {
        hint: null,
        details: null
      };
      const jsonMatch = message.message.match(/{.*}/);
      const json = jsonMatch ? JSON.parse(jsonMatch[0]) : null;

      if (json) {
        const { hint, details } = json;
        if (hint) error["hint"] = hint
        if (details) error["details"] = details
      }
      const errorMessage: ErrorMessage = {
        type: "error",
        message: `${error.details}\n\n${error.hint}`,
        thread_id: message.thread_id
      }
      setLoading(false)
      setWsLoading(false)
      setErrorMessage(errorMessage)
    }
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
    setDebugMessages(null);
    setErrorMessage(null);
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

  const sendMessage = async ({content, thread_id, org_id}: SendMessageType) => {
    setWsLoading(true)
    setErrorMessage(null)
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
          ai_model: `${aiModel?.vendor}/${aiModel?.name}`,
          is_public: chatVisibility === 'public'
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
      const { error } = await updateChatVisibility({
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
    if (messages && messages.length > 0) {
      const newVisibility = messages[0].is_public ? Visibility.PUBLIC : Visibility.PRIVATE;
      if (newVisibility !== chatVisibility) {
        setChatVisibility(newVisibility)
      }
    }
  }, [messages]);

  useEffect(() => {
    const newVisibility = isPublicByDefault ? Visibility.PUBLIC : Visibility.PRIVATE;
    if (newVisibility !== chatVisibility) {
      setChatVisibility(newVisibility)
    }
  }, [isPublicByDefault, threadId]);

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
    aiModel,
    setAiModel,
    aiModels,
    aiModelsLoading,
    chatVisibility,
    setChatVisibility,
    debugMessages,
    debugMessagesLoading,
    stateMessage,
    isStreamingInProcess,
    currentStreamMessage,
    errorMessage
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

type UseAiModelsList = {
  aiModels: AiModel[] | null
  error: Response | null
  aiModel: AiModel | null
  loading: boolean
  setAiModel: (model: AiModel) => void
}

export const useAiModelsList = (): UseAiModelsList => {
  const [llmModels, setLLMModels] = useState<UseAiModelsList['aiModels']>(null);
  const [error, setError] = useState<Response | null>(null);
  const [userModel, setUserModel] = useState<AiModel | null>(null);
  const [loading, setLoading] = useState(false)

  const getModels = useCallback(async () => {
    let models = null;
    setLoading(true);
    try {
      const { response } = await getAiModels();
      setLLMModels(response);
      const currentModel = window.localStorage.getItem('bot.ai_model');
      const parsedModel: AiModel = currentModel ? JSON.parse(currentModel) : null;

      if (currentModel && parsedModel.name !== userModel?.name && response) {
        // Check if the parsedModel exists in the response models
        const modelInResponse = response.find(
          (model) =>
            model.name.includes(parsedModel.name)
        );

        if (modelInResponse) {
          setUserModel(modelInResponse);
          window.localStorage.setItem('bot.ai_model', JSON.stringify(modelInResponse));
        } else {
          // Model from localStorage does not exist in response
          // Find a default model
          const defaultModel = response.find((model) =>
            model.name.includes(DEFAULT_MODEL_NAME)
          );

          if (defaultModel) {
            setUserModel(defaultModel);
            window.localStorage.setItem('bot.ai_model', JSON.stringify(defaultModel));
          }
        }
      } else if (response) {
        // Find a model where the model name includes the DEFAULT_MODEL_NAME
        const matchingModel = response.find((model) =>
          model.name.includes(DEFAULT_MODEL_NAME)
        );
        if (matchingModel) {
          setModel(matchingModel);
          window.localStorage.setItem('bot.ai_model', JSON.stringify(matchingModel));
        }
      }
    } catch (e) {
      setError(e as unknown as Response);
    }
    setLoading(false);
    return models;
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

  const setModel = (model: AiModel) => {
    if (model !== userModel) {
      setUserModel(model);
      window.localStorage.setItem('bot.ai_model', JSON.stringify(model))
    }
  }

  return {
    aiModels: llmModels,
    error,
    setAiModel: setModel,
    loading,
    aiModel: userModel,
  }
}