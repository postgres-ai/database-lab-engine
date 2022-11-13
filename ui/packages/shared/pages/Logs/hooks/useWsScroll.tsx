import { useState, useEffect } from 'react'
import { wsSnackbar } from '@postgres.ai/shared/pages/Logs/wsSnackbar'

export const useWsScroll = (isLoading: boolean) => {
  const [isNewData, setIsNewData] = useState(false)
  const [isAtBottom, setIsAtBottom] = useState(true)

  useEffect(() => {
    !isLoading && wsSnackbar(isAtBottom, isNewData)

    const contentElement = document.getElementById('content-container')
    const targetNode = document.getElementById('logs-container')

    const clientAtBottom = (element: HTMLElement) =>
      element.scrollHeight - element.scrollTop - 50 < element.clientHeight

    const handleScroll = (e: Event) => {
      if (clientAtBottom(e.target as HTMLElement)) {
        setIsAtBottom(true)
        setIsNewData(false)
      } else {
        setIsAtBottom(false)
      }
    }

    const handleInsert = (e: Event | any) => {
      if (e.srcElement?.tagName !== 'DIV') {
        isAtBottom && targetNode?.scrollIntoView(false)
        setIsNewData(true)
      }
    }

    contentElement?.addEventListener('scroll', handleScroll, false)
    contentElement?.addEventListener('DOMNodeInserted', handleInsert, false)

    return () => {
      contentElement?.removeEventListener('scroll', handleScroll, false)
      contentElement?.removeEventListener(
        'DOMNodeInserted',
        handleInsert,
        false,
      )
    }
  }, [isAtBottom, isNewData, isLoading])
}
