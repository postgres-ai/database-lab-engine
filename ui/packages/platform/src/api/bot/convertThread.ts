import {request} from "../../helpers/request";

export const convertThread = (thread_id: string): Promise<{ response: { final_thread_id: string, msg: string } | null; error: Response | null }> => {
  const apiServer = process.env.REACT_APP_BOT_API_URL || '';

  return request(
    `/convert_thread`,
    {
      method: 'POST',
      body: JSON.stringify({ thread_id }),
    },
    apiServer
  )
    .then(async (response) => {
      if (!response.ok) {
        return { response: null, error: response };
      }
      const responseData = await response.json();
      return { response: responseData, error: null };
    })
    .catch((error: Response) => {
      return { response: null, error };
    });
};