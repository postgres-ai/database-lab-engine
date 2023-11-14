import { simpleInstallRequest } from 'helpers/simpleInstallRequest'

export const getTaskState = async (req: { taskID: string; userID?: number }) => {
  const response = await simpleInstallRequest(
    `/state/${req.taskID}`,
    {
      method: 'GET',
    },
    req?.userID,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
