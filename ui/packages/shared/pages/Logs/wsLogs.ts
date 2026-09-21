import moment from 'moment'
import { Api } from '../Instance/stores/Main'
import {
  readLogsFilterState,
  stringContainsPattern,
  stringWithoutBrackets,
} from './utils'

const logsEndpoint = '/instance/logs'

const LOGS_TIME_LIMIT = 20
const LOGS_LINE_LIMIT = 1000

// The page keeps at most one log socket open. `generation` invalidates a connection attempt
// that is still awaiting its token when the caller reconnects or unmounts, so a socket opened
// after that point is closed instead of left dangling.
let activeSocket: WebSocket | null = null
let generation = 0

export const closeConnection = () => {
  generation++

  if (!activeSocket) return

  // Detach the handler first: it reports a server-side drop to the user, which an
  // intentional close is not.
  activeSocket.onclose = null
  activeSocket.close()
  activeSocket = null
}

export const establishConnection = async (api: Api, instanceId: string) => {
  if (!api.getWSToken) return

  closeConnection()

  const currentGeneration = generation

  const logElement = document.getElementById('logs-container')

  if (logElement === null) {
    console.log('Not found container element')
    return
  }

  const appendLogElement = (logEntry: string, logType?: string) => {
    const tag = document.createElement('p')
    const logLevel = logEntry.split(' ')[3]
    const logInitiator = logEntry.split(' ')[2]
    const logsFilterState = readLogsFilterState()

    const filterInitiators = Object.keys(logsFilterState).some((state) => {
      if (logsFilterState[state]) {
        if (state === '[other]') {
          return !stringContainsPattern(logInitiator)
        }
        return logInitiator?.includes(stringWithoutBrackets(state))
      }
    })

    if (
      filterInitiators &&
      (logsFilterState[logInitiator] ||
        logsFilterState[logLevel] ||
        logEntry === 'Connection Error')
    ) {
      tag.appendChild(document.createTextNode(logEntry))
      logElement.appendChild(tag)
    }

    // we need to check both second and third element of logEntry,
    // since the pattern of the response returned isn't always consistent
    if (logInitiator === '[ERROR]' || logLevel === '[ERROR]') {
      tag.classList.add('error-log')
    }

    // The container is empty whenever the active filters drop every line received so far,
    // so the oldest entry the trim is measured against may not exist.
    const oldestEntry = logElement.children[0]

    if (logType === 'message' && oldestEntry) {
      const logEntryTime = moment.utc(
        oldestEntry.innerHTML.split(' ').slice(0, 2).join(' '),
      )

      const timeDifference =
        moment(logEntryTime).isValid() &&
        moment.duration(moment.utc(Date.now()).diff(logEntryTime)).asMinutes()

      if (
        logElement.childElementCount > LOGS_LINE_LIMIT &&
        Number(timeDifference) > LOGS_TIME_LIMIT
      ) {
        logElement.removeChild(logElement.children[0])
      }
    }

    if (
      logEntry.split(' ')[2] === '[ERROR]' ||
      logEntry.split(' ')[3] === '[ERROR]'
    ) {
      tag.classList.add('error-log')
    }
  }

  const { response, error } = await api.getWSToken({
    instanceId: instanceId,
  })

  if (error || response == null) {
    console.log('Not authorized:', error)
    appendLogElement('Not authorized')
    return
  }

  if (api.initWS == null) {
    console.log('WebSocket Connection is not configured')
    appendLogElement('WebSocket Connection is not configured')
    return
  }

  const socket = api.initWS(logsEndpoint, response.token)

  if (currentGeneration !== generation) {
    socket.close()
    return
  }

  activeSocket = socket

  socket.onopen = () => {
    console.log('Successfully Connected')
  }

  socket.onclose = (event) => {
    console.log('Socket Closed Connection: ', event)

    if (activeSocket === socket) {
      activeSocket = null
    }

    appendLogElement('DBLab Connection Closed')
  }

  socket.onerror = (error) => {
    console.log('Socket Error: ', error)

    appendLogElement('Connection Error')
  }

  socket.onmessage = function (event) {
    const logEntry = decodeURIComponent(atob(event.data))
    appendLogElement(logEntry, 'message')
  }
}

export const restartConnection = (api: Api, instanceId: string) => {
  const logElement = document.getElementById('logs-container')

  if (logElement && logElement.childElementCount > 1) {
    while (logElement.firstChild) {
      logElement.removeChild(logElement.firstChild)
    }
  }

  establishConnection(api, instanceId)
}
