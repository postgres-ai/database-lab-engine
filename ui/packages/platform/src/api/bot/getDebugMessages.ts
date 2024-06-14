import {request} from "../../helpers/request";
import { DebugMessage } from "../../types/api/entities/bot";

type Req =
  | { thread_id: string; message_id?: string }
  | { thread_id?: string; message_id: string };

export const getDebugMessages = async (req: Req): Promise<{ response: DebugMessage[] | null; error: Response | null }> => {
  const { thread_id, message_id } = req;

  const params: { [key: string]: string } = {};

  if (thread_id) {
    params['chat_thread_id'] = `eq.${thread_id}`;
  }

  if (message_id) {
    params['chat_msg_id'] = `eq.${message_id}`;
  }

  const queryString = new URLSearchParams(params).toString();

  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';

  try {
    const response = await request(`${apiServer}/chat_debug_messages?${queryString}`);

    if (!response.ok) {
      return { response: null, error: response };
    }

    const responseData: DebugMessage[] = await response.json();

    return { response: responseData, error: null };

  } catch (error) {
    return { response: null, error: error as Response };
  }
}