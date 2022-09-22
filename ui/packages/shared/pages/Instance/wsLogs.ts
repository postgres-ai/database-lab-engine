import moment from 'moment';
import { Api } from "./stores/Main";

const logsEndpoint = '/instance/logs';

const LOGS_TIME_LIMIT = 20
const LOGS_LINE_LIMIT = 1000

export const establishConnection = async (api: Api) => {
    const logElement = document.getElementById("logs-container");

    if (logElement === null) {
        console.log("Not found container element");
        return;
    }

    const appendLogElement = (logEntry: string, logType?: string) => {
      const tag = document.createElement('p')
      tag.appendChild(document.createTextNode(logEntry))
      logElement.appendChild(tag)
      logElement.scrollIntoView(false)
  
      if (logType === 'message') {
        const logEntryTime = moment.utc(
          logElement.children[0].innerHTML.split(' ').slice(0, 2).join(' '),
        )
  
        const timeDifference =
          moment(logEntryTime).isValid() &&
          moment.duration(moment.utc(Date.now()).diff(logEntryTime)).asMinutes()
  
        if (
          logElement.childElementCount > LOGS_LINE_LIMIT &&
          timeDifference > LOGS_TIME_LIMIT
        ) {
          logElement.removeChild(logElement.children[0])
        }
      }
  
      if (logEntry.split(' ')[2] === '[ERROR]' || logEntry.split(' ')[3] === "[ERROR]") {
        tag.classList.add('error-log')
      }
    }

    const { response, error } = await api.getWSToken({
        instanceId: "",
    })

    if (error || response == null) {
        console.log("Not authorized:", error);
        appendLogElement("Not authorized")
        return;
    }

    if (api.initWS == null) {
        console.log("WebSocket Connection is not configured");
        appendLogElement("WebSocket Connection is not configured")
        return;
    }

    const socket = api.initWS(logsEndpoint, response.token);

    socket.onopen = () => {
        console.log("Successfully Connected");
    };

    socket.onclose = event => {
        console.log("Socket Closed Connection: ", event);
        socket.send("Client Closed")
        appendLogElement("DLE Connection Closed")
    };

    socket.onerror = error => {
        console.log("Socket Error: ", error);

        appendLogElement("Connection Error")
    };

    socket.onmessage = function (event) {
        const logEntry = decodeURIComponent(atob(event.data))
        appendLogElement(logEntry, "message")
    }
};
