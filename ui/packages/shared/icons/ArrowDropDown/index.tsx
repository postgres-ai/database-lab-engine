/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import React from 'react'

type Props = {
  className?: string
  onClick?: () => void
}

export const ArrowDropDownIcon = (props: Props) => {
  return (
    <svg
      className={props.className}
      viewBox="0 0 8 6"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      onClick={props.onClick}
    >
      <path
        // eslint-disable-next-line max-len
        d="M7.8515 0.898419C7.75261 0.799446 7.63538 0.75 7.49994 0.75H0.500038C0.364534 0.75 0.247392 0.799446 0.148419 0.898419C0.0494455 0.997501 0 1.11464 0 1.25006C0 1.38546 0.0494455 1.5026 0.148419 1.6016L3.64838 5.10156C3.74746 5.20054 3.86461 5.25009 4 5.25009C4.13539 5.25009 4.25265 5.20054 4.35154 5.10156L7.8515 1.60157C7.95036 1.5026 8 1.38546 8 1.25004C8 1.11464 7.95036 0.997501 7.8515 0.898419Z"
        fill="currentColor"
      />
    </svg>
  )
}
