/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { emoji } from 'config/emoji'

export const preformatJoeMessage = (text: string) => {
  let result = text

  // Preprocess emoji.
  const emojiEntries = Object.entries(emoji)

  for (const [emojiKey, emojiValue] of emojiEntries) {
    result = result.split(`:${emojiKey}:`).join(emojiValue)
  }

  // Remove artifacts links.
  result = result.replace('_(The text in the preview above has been cut)_', '')
  result = result.replace(
    'Other artifacts are provided in the thread',
    'Other artifacts are provided below',
  )

  // Prepare links.
  result = result.replace(/<(.*)\|(.*)>/gm, '[$2]($1)')

  // Preprocess  bold.
  // Matched example: 'Hello *username*!' -> 'Hello **username**!'.
  // Unmatched example: 'explain select * from user where username ~*' -> 'explain select * from user where username ~*'.
  // TODO(Anton): Tests like Jest.
  result = result.replace(/\*(\S+)\*/g, '**$1**')

  // Preprocess code blocks.
  result = result.replace(/```(.*)```/g, '`$1`')
  result = result.split(/^```/m).join('```\n')
  result = result.split(/```$/m).join('\n```')
  result = result.split('\n').join('  \n')

  return result
}
