import { useState, useEffect } from 'react'
import { wsSnackbar } from '@postgres.ai/shared/pages/Logs/wsSnackbar'

export const useWsScroll = (isLoading: boolean, simpleInstall?: boolean) => {
  const [isNewData, setIsNewData] = useState(false)
  const [isAtBottom, setIsAtBottom] = useState(true)

  useEffect(() => {
    !isLoading && wsSnackbar(isAtBottom, isNewData)
    const targetNode = simpleInstall
      ? document.getElementById('logs-container')?.parentElement
      : document.getElementById('logs-container')

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
        isAtBottom &&
          targetNode?.scroll({
            top: targetNode.scrollHeight,
          })
        setIsNewData(true)
      }
    }

    targetNode?.addEventListener('scroll', handleScroll, false)
    targetNode?.addEventListener('DOMNodeInserted', handleInsert, false)

    return () => {
      targetNode?.removeEventListener('scroll', handleScroll, false)
      targetNode?.removeEventListener('DOMNodeInserted', handleInsert, false)
    }
  }, [isAtBottom, isNewData, isLoading])
}
