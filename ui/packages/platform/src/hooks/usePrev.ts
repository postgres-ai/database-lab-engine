/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2021, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { useRef, useEffect } from 'react'

export const usePrev = <T>(value: T) => {
  const ref = useRef<T>()
  
  useEffect(() => {
    ref.current = value;
  }, [value])

  return ref.current
}
