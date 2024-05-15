import {request} from "../../helpers/request";
import {BotMessage} from "../../types/api/entities/bot";

type Req = {
  query?: string
}

export const getChats = async (req: Req): Promise<{ response: BotMessage[] | null; error: Response | null }> => {
  const { query } = req;

  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';

  try {
    const response = await request(`${apiServer}/chats${query ? query : ''}`, {
      method: 'GET',
    });

    if (!response.ok) {
      return { response: null, error: response };
    }

    const responseData: BotMessage[] = await response.json();

    return { response: responseData, error: null };

  } catch (error) {
    return { response: null, error: error as Response };
  }
}