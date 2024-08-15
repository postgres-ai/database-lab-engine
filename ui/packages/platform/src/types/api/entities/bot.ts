export type DebugMessage = {
  type: 'debug'
  message_id: string | null
  org_id: string
  thread_id: string
  content: string
  created_at: string
}

export type BotMessage = {
  id: string
  created_at: string
  parent_id: string | null
  content: string
  is_ai: boolean
  is_public: boolean
  first_name: string | null
  last_name: string | null
  display_name: string | null
  slack_profile: string | null
  user_id: string
  org_id: string
  thread_id: string
  type: 'message' | undefined
  ai_model: string
}

export type BotMessageWithDebugInfo = BotMessage & {
  debugMessages?: DebugMessage[]
}

export type AiModel = {
  name: string;
  vendor: string;
  isThirdParty: boolean;
  freeUseAvailable: boolean;
};

export type StateMessage = {
  type: 'state'
  state: string | null
  thread_id: string
}

export type StreamMessage = {
  type: 'stream'
  content: string
  ai_model: string
  thread_id: string
}