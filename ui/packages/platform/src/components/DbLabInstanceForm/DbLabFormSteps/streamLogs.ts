import { initStreamLogs } from '@postgres.ai/platform/src/api/configs/initStreamLogs'
import { getTaskState } from 'api/configs/getTaskState'
import { regenerateCode } from 'api/configs/regenerateCode'

export const establishConnection = async ({
  taskId,
  otCode,
  userID,
  isConnected,
  setIsConnected,
}: {
  taskId: string
  otCode: string
  userID?: number
  isConnected: boolean
  setIsConnected: (isConnected: boolean) => void
}) => {
  const logElement = document.getElementById('logs-container')

  if (logElement === null) {
    return
  }

  const appendLogElement = (logEntry: string) => {
    const codeTag = logElement.querySelector('code')
    if (codeTag) {
      codeTag.appendChild(document.createTextNode(logEntry + '\n'))
      logElement.appendChild(codeTag)
    }
  }

  const socket = initStreamLogs(taskId, otCode)

  socket.onclose = () => {
    setIsConnected(false)
  }

  socket.onerror = () => {
    if (!isConnected) {
      return
    }

    setTimeout(() => {
      getTaskState({ taskID: taskId, userID }).then((res) => {
        if (
          res.response?.state &&
          res.response.state !== 'finished' &&
          res.response.state !== 'error'
        ) {
          while (logElement.firstChild) {
            logElement.removeChild(logElement.firstChild)
          }
          regenerateCode({ taskID: taskId, userID }).then((res) => {
            if (res.response) {
              establishConnection({
                taskId,
                otCode: res.response?.otCode,
                userID,
                isConnected,
                setIsConnected,
              })
            }
          })
        }
      })
    }, 5000)
  }

  socket.onmessage = function (event) {
    const logEntry = decodeURIComponent(atob(event.data))
    appendLogElement(logEntry)
    setIsConnected(true)
  }
}
