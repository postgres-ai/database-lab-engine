import {request} from "../../helpers/request";
import { AiModel } from "../../types/api/entities/bot";

export const getAiModels = async (orgId?: number): Promise<{ response: AiModel[] | null; error: Response | null }> => {
  const apiServer = process.env.REACT_APP_API_URL_PREFIX || '';
  const body = {
    org_id: orgId
  }
  try {
    const response = await request(`${apiServer}/rpc/bot_llm_models`, {
      method: 'POST',
      body: JSON.stringify(body),
      headers: {
        'Accept': 'application/vnd.pgrst.object+json',
        'Prefer': 'return=representation',
      }
    });

    if (!response.ok) {
      return { response: null, error: response };
    }

    const responseData: { bot_llm_models: AiModel[] | null } = await response.json();

    return { response: responseData?.bot_llm_models, error: null };

  } catch (error) {
    return { response: null, error: error as Response };
  }
}