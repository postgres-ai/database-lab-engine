import { simpleInstallRequest } from 'helpers/simpleInstallRequest'

export const regenerateCode = async (req: {
  taskID: string
  userID?: number
}) => {
  const response = await simpleInstallRequest(
    '/regenerate-code',
    {
      method: 'POST',
      body: JSON.stringify({
        taskID: req.taskID,
      }),
    },
    req?.userID,
  )

  return {
    response: response.ok ? await response.json() : null,
    error: response.ok ? null : await response.json(),
  }
}
