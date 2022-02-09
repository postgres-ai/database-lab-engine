/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'

type Props = {
  className?: string
}

export const CircleIcon = (props: Props) => {
  const { className } = props

  return (
    <svg
      className={className}
      viewBox="0 0 10 10"
      version="1.1"
      xmlns="http://www.w3.org/2000/svg"
    >
      <circle cx="5" cy="5" r="5" fill="currentColor"></circle>
    </svg>
  )
}
