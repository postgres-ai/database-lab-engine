import { useEffect } from 'react'
import { wsSnackbar } from '@postgres.ai/shared/pages/Logs/wsSnackbar'

export const useWsScroll = () => {
  const snackbarTag = document.createElement('div')
  const contentElement = document.getElementById('content-container')

  useEffect(() => {
    const targetNode = document.getElementById('logs-container')
    contentElement.addEventListener(
      'scroll',
      () => {
        wsSnackbar(contentElement, targetNode, snackbarTag)
      },
      false,
    )

    return () =>
      contentElement.removeEventListener(
        'scroll',
        () => {
          wsSnackbar(contentElement, targetNode, snackbarTag)
        },
        false,
      )
  }, [])
}
