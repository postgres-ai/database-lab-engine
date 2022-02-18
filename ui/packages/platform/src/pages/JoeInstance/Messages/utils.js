/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

export const getMaxScrollTop = element => element.scrollHeight - element.clientHeight;

export const getMessageArtifactIds = (message) => {
  let ids = [];

  if (!message) {
    return ids;
  }

  let match = message.match(/\/artifact\/(\d*)/);
  if (!match || (match && match.length < 3)) {
    return ids;
  }

  for (let i = 1; i < match.length; i = i + 2) {
    ids.push(parseInt(match[i], 10));
  }

  return ids;
};

export const getUserMessagesCount = (messages) => {
  if (!messages) {
    return 0;
  }

  const keys = Object.keys(messages);

  return keys.reduce((count, key) => {
    const message = messages[key];
    return message.author_id ? count + 1 : count;
  }, 0);
};
