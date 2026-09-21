import { useState, useEffect } from 'react'
import { wsSnackbar } from '@postgres.ai/shared/pages/Logs/wsSnackbar'

export const useWsScroll = (isLoading: boolean, simpleInstall?: boolean) => {
  const [isNewData, setIsNewData] = useState(false)
  const [isAtBottom, setIsAtBottom] = useState(true)

  useEffect(() => {
    if (!isLoading) {
      wsSnackbar(isAtBottom, isNewData)
    }

    const contentElement = document.getElementById('content-container')
    const targetNode = simpleInstall
      ? document.getElementById('logs-container')?.parentElement
      : document.getElementById('logs-container')

    const clientAtBottom = (element: HTMLElement) =>
      element.scrollHeight - element.scrollTop - 50 < element.clientHeight

    const handleScroll = (e: Event) => {
      if (!clientAtBottom(e.target as HTMLElement)) {
        setIsAtBottom(false)
        return
      }

      setIsAtBottom(true)
      setIsNewData(false)
    }

    // Log lines arrive as <p> nodes; a <div> is page chrome. The filter keeps the tagName
    // guard the removed DOMNodeInserted listener used.
    const handleInsert = (mutations: MutationRecord[]) => {
      const hasNewLine = mutations.some((mutation) =>
        Array.from(mutation.addedNodes).some((node) => node.nodeName !== 'DIV'),
      )

      if (!hasNewLine) return

      if (isAtBottom) {
        targetNode?.scrollIntoView(false)
      }

      setIsNewData(true)
    }

    const observer = new MutationObserver(handleInsert)

    contentElement?.addEventListener('scroll', handleScroll, false)

    if (contentElement) {
      observer.observe(contentElement, { childList: true, subtree: true })
    }

    return () => {
      contentElement?.removeEventListener('scroll', handleScroll, false)
      observer.disconnect()
    }
  }, [isAtBottom, isNewData, isLoading, simpleInstall])
}
