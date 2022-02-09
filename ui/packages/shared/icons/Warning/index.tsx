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

export const WarningIcon = (props: Props) => {
  return (
    <svg
      className={props.className}
      viewBox="0 0 10 10"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        // eslint-disable-next-line max-len
        d="M9.79222 7.49885L6.2597 1.00518C5.69214 0.04969 4.30861 0.0484205 3.74029 1.00518L0.207947 7.49885C-0.372248 8.4752 0.330193 9.71157 1.46736 9.71157H8.53251C9.66873 9.71157 10.3724 8.47619 9.79222 7.49885ZM5 8.53969C4.67699 8.53969 4.41406 8.27676 4.41406 7.95375C4.41406 7.63074 4.67699 7.36782 5 7.36782C5.323 7.36782 5.58593 7.63074 5.58593 7.95375C5.58593 8.27676 5.323 8.53969 5 8.53969ZM5.58593 6.19594C5.58593 6.51895 5.323 6.78188 5 6.78188C4.67699 6.78188 4.41406 6.51895 4.41406 6.19594V3.26625C4.41406 2.94324 4.67699 2.68032 5 2.68032C5.323 2.68032 5.58593 2.94324 5.58593 3.26625V6.19594Z"
        fill="currentColor"
      />
    </svg>
  )
}
