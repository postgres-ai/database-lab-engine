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

export const RenewableIcon = React.forwardRef<SVGSVGElement, Props>(
  (props, ref) => {
    const { className, ...hiddenProps } = props
    return (
      <svg
        {...hiddenProps}
        ref={ref}
        className={className}
        viewBox="0 0 10 10"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
      >
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          // eslint-disable-next-line max-len
          d="M6.14337 0L8.19809 2.9755L4.59387 3.26718L5.16139 2.07055C4.22847 2.10806 3.3161 2.48386 2.66614 3.18795L2.20594 2.76312C3.05679 1.84141 4.26977 1.40592 5.45883 1.44339L6.14337 0Z"
          fill="currentColor"
        />
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          // eslint-disable-next-line max-len
          d="M9.7343 9.13905L6.12728 9.39371L7.71025 6.14261L8.45158 7.24009C8.89401 6.4179 9.03479 5.44126 8.75942 4.52346L9.35931 4.34347C9.71979 5.54495 9.47743 6.81074 8.8401 7.81527L9.7343 9.13905Z"
          fill="currentColor"
        />
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          // eslint-disable-next-line max-len
          d="M0 7.57495L1.61898 4.34163L3.60962 7.36037L2.28757 7.43896C2.76953 8.23862 3.53811 8.85742 4.46813 9.08819L4.3173 9.69607C3.09983 9.39397 2.13422 8.54041 1.59468 7.48015L0 7.57495Z"
          fill="currentColor"
        />
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          // eslint-disable-next-line max-len
          d="M6.26612 2.51078C5.07868 2.17637 3.73747 2.48899 2.89616 3.40036L1.97575 2.55072C3.17071 1.25624 5.01514 0.857134 6.60567 1.30506L6.26612 2.51078Z"
          fill="currentColor"
        />
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          // eslint-disable-next-line max-len
          d="M7.33938 8.11969C8.34588 7.28131 8.83847 5.87691 8.45939 4.61345L9.65917 4.25348C10.1969 6.04581 9.49312 7.95595 8.14107 9.08215L7.33938 8.11969Z"
          fill="currentColor"
        />
        <path
          fill-rule="evenodd"
          clip-rule="evenodd"
          // eslint-disable-next-line max-len
          d="M2.07072 5.82736C2.20416 7.20026 3.19328 8.44922 4.54348 8.78425L4.24181 10C2.32911 9.5254 1.00387 7.79934 0.823976 5.94853L2.07072 5.82736Z"
          fill="currentColor"
        />
      </svg>
    )
  },
)
