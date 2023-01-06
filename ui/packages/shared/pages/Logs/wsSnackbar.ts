import { LOGS_NEW_DATA_MESSAGE } from '@postgres.ai/shared/pages/Logs/constants'

export const wsSnackbar = (clientAtBottom: boolean, isNewData: boolean) => {
  const targetNode = document.getElementById('logs-container')
  const snackbarTag = document.createElement('div')

  if (!clientAtBottom && isNewData) {
    if (!targetNode?.querySelector('.snackbar-tag')) {
      targetNode?.appendChild(snackbarTag)
      snackbarTag.classList.add('snackbar-tag')
      if (snackbarTag.childNodes.length === 0) {
        snackbarTag.appendChild(document.createTextNode(LOGS_NEW_DATA_MESSAGE))
      }
      snackbarTag.onclick = () => {
        targetNode?.scroll({
          top: targetNode.scrollHeight,
          behavior: 'smooth',
        })
      }
    }
  } else {
    targetNode?.querySelector('.snackbar-tag')?.remove()
  }
}
