const LOGS_NEW_DATA_MESSAGE =
  'New data arrived below - scroll down to see it 👇🏻'

export const wsSnackbar = (
  contentElement: HTMLElement,
  targetNode: HTMLElement,
  snackbarTag: HTMLElement,
) => {
  if (
    contentElement.scrollHeight - contentElement.scrollTop - 50 >
    contentElement.clientHeight
  ) {
    if (!targetNode.querySelector('.snackbar-tag')) {
      targetNode.appendChild(snackbarTag)
      snackbarTag.classList.add('snackbar-tag')
      if (snackbarTag.childNodes.length === 0) {
        snackbarTag.appendChild(document.createTextNode(LOGS_NEW_DATA_MESSAGE))
      }
      snackbarTag.onclick = () => {
        targetNode.scrollIntoView({
          behavior: 'smooth',
          block: 'end',
          inline: 'end',
        })
      }
    }
  } else {
    snackbarTag.remove()
  }
}
