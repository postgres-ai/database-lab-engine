import {request} from "../../helpers/request";
import { LLMModel } from "../../types/api/entities/bot";

export const getLLMModels = async (): Promise<{ response: LLMModel[] | null; error: Response | null }> => {
  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';

  try {
    const response = await request(`${apiServer}/llm_models`, {
      method: 'GET',
    });

    if (!response.ok) {
      return { response: null, error: response };
    }

    const responseData: LLMModel[] = await response.json();

    return { response: responseData, error: null };

  } catch (error) {
    return { response: null, error: error as Response };
  }
}