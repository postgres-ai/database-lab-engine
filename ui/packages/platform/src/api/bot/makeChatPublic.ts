import {request} from "../../helpers/request";
import {BotMessage} from "../../types/api/entities/bot";

type Req = {
  thread_id: string,
  is_public: boolean
}

export const makeChatPublic = async (req: Req): Promise<{ response: BotMessage | null; error: Response | null }> => {
  const { thread_id, is_public } = req;

  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';

  try {
    const response = await request(`${apiServer}/chats_internal?thread_id=eq.${thread_id}`, {
      method: 'PATCH',
      headers: {
        Prefer: 'return=representation'
      },
      body: JSON.stringify({
        is_public
      })
    });

    if (!response.ok) {
      return { response: null, error: response };
    }

    const responseData: BotMessage = await response.json();

    return { response: responseData, error: null };

  } catch (error) {
    return { response: null, error: error as Response };
  }
}
