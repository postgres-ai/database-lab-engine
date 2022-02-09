/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

type Instance = {
  channels?: {
    [key in string]?: {
      commandErrorMessage?: string
      wsErrorMessage?: string
    }
  }
}

export const getSystemMessages = (instance: Instance, channelId: string) => {
  const channels = instance?.channels
  if (!channels) return []

  const channel = channels[channelId]
  if (!channel) return []

  const { commandErrorMessage, wsErrorMessage } = channel

  const systemMessages = []

  if (commandErrorMessage) {
    systemMessages.push({
      type: 'error' as const,
      value: commandErrorMessage,
    })
  }

  if (wsErrorMessage) {
    systemMessages.push({
      type: 'warning' as const,
      value: wsErrorMessage,
    })
  }

  return systemMessages
}
