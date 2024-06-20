import {request} from "../../helpers/request";
import { AiModel } from "../../types/api/entities/bot";

export const getAiModels = async (): Promise<{ response: AiModel[] | null; error: Response | null }> => {
  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';

  try {
    const response = await request(`${apiServer}/llm_models`, {
      method: 'GET',
    });

    if (!response.ok) {
      return { response: null, error: response };
    }

    const responseData: AiModel[] = await response.json();

    return { response: responseData, error: null };

  } catch (error) {
    return { response: null, error: error as Response };
  }
}