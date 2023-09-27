import { SI_API_SERVER } from 'helpers/simpleInstallRequest'

export const initStreamLogs = (taskId: string, otCode: string): WebSocket => {
  let url = new URL(
    `${SI_API_SERVER.replace(
      'https',
      'wss',
    )}/stream-logs/${taskId}?otCode=${otCode}`,
  )
  return new WebSocket(url)
}
