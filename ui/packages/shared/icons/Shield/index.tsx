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

export const ShieldIcon = React.forwardRef<SVGSVGElement, Props>(
  (props, ref) => {
    const { className, ...hiddenProps } = props
    return (
      <svg
        {...hiddenProps}
        ref={ref}
        className={className}
        viewBox="0 0 8 10"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          d="M4.00195 0L0.251953 1.66667V4.16667C0.251953 6.47917 1.85195 8.64167 4.00195 9.16667C6.15195 8.64167 7.75195 6.47917 7.75195 4.16667V1.66667L4.00195 0ZM3.16862 6.66667L1.50195 5L2.08945 4.4125L3.16862 5.4875L5.91445 2.74167L6.50195 3.33333L3.16862 6.66667Z"
          fill="currentColor"
        />
      </svg>
    )
  },
)
