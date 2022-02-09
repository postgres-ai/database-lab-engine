import { useEffect, useRef } from 'react'

import { config } from '@postgres.ai/shared/config'

type Props = {
  title: string
}

export const Head = (props: Props) => {
  const { title } = props

  const titleRef = useRef(window.document.title)

  useEffect(
    () => () => {
      window.document.title = titleRef.current
    },
    [],
  )

  window.document.title = title

  return null
}

export const createTitle = (parts: string[]) => {
  parts.reverse()
  parts.push(config.appName)
  return parts.join(' · ');
}
