import moment from 'moment'

import {
  LOGS_ENDPOINT,
  LOGS_LINE_LIMIT,
  LOGS_TIME_LIMIT,
} from '@postgres.ai/shared/pages/Logs/constants'
import { Api } from '@postgres.ai/shared/pages/Instance/stores/Main'

export const establishConnection = async (api: Api) => {
  const logElement = document.getElementById('logs-container')

  if (logElement === null) {
    console.log('Not found container element');
    return
  }

  const appendLogElement = (logEntry: string, logType?: string) => {
    const tag = document.createElement('p')
    const logsFilterState = JSON.parse(localStorage.getItem('logsState') || '')

    // check if logEntry message type is truthy in the logsFilterState
    if (logsFilterState[logEntry.split(' ')[2]] || logsFilterState[logEntry.split(' ')[3]]) {
      tag.appendChild(document.createTextNode(logEntry))
      logElement.appendChild(tag)
    }

    // we need to check both second and third element of logEntry,
    // since the pattern of the response returned isn't always consistent
    if (
      logEntry.split(' ')[2] === '[ERROR]' ||
      logEntry.split(' ')[3] === '[ERROR]'
    ) {
      tag.classList.add('error-log')
    }

    if (logType === 'message') {
      const logEntryTime = logElement.children[1]?.innerHTML
        .split(' ')
        .slice(0, 2)
        .join(' ')

      const timeDifference =
        moment(logEntryTime).isValid() &&
        moment.duration(moment.utc(Date.now()).diff(logEntryTime)).asMinutes()

      if (
        logElement.childElementCount > LOGS_LINE_LIMIT &&
        timeDifference > LOGS_TIME_LIMIT
      ) {
        logElement.removeChild(logElement.children[1])
      }
    }
  }

  const { response, error } = await api.getWSToken({
    instanceId: '',
  })

  if (error || response == null) {
    console.log('Not authorized:', error);
    appendLogElement('Not authorized')
    return;
  }

  if (api.initWS == null) {
    console.log('WebSocket Connection is not configured')
    appendLogElement('WebSocket Connection is not configured')
    return
  }

  const socket = api.initWS(LOGS_ENDPOINT, response.token)

  socket.onopen = () => {
    console.log('Successfully Connected');
  }

  socket.onclose = (event) => {
    console.log('Socket Closed Connection: ', event);
    socket.send('Client Closed')
    appendLogElement('DLE Connection Closed')
  }

  socket.onerror = (error) => {
    console.log('Socket Error: ', error);

    appendLogElement('Connection Error')
  }

  socket.onmessage = function (event) {
    const logEntry = decodeURIComponent(atob(event.data))
    appendLogElement(logEntry, 'message')
  }
}
