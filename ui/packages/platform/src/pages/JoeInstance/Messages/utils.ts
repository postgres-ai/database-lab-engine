/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const getMaxScrollTop = (element: HTMLElement) =>
  element.scrollHeight - element.clientHeight

export const getMessageArtifactIds = (message: string) => {
  const ids: number[] = []

  if (!message) {
    return ids
  }

  const match = message.match(/\/artifact\/(\d*)/)
  if (!match || (match && match.length < 3)) {
    return ids
  }

  for (let i = 1; i < match.length; i = i + 2) {
    ids.push(parseInt(match[i], 10))
  }

  return ids
}

export const getUserMessagesCount = (messages: {
  [x: string]: { author_id: string }
}) => {
  if (!messages) {
    return 0
  }

  const keys = Object.keys(messages)

  return keys.reduce((count, key) => {
    return messages[key].author_id ? count + 1 : count
  }, 0)
}
