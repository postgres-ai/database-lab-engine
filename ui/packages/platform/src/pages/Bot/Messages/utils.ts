/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import {BotMessage} from "../../../types/api/entities/bot";

export const getMaxScrollTop = (element: HTMLElement) =>
  element.scrollHeight - element.clientHeight


export const getUserMessagesCount = (messages: BotMessage[]) => {
  if (!messages) {
    return 0
  }

  const keys = Object.keys(messages)

  return keys.reduce((count, key) => {
    const idx = Number(key)
    return !messages[idx].is_ai ? count + 1 : count
  }, 0)
}
