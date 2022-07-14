import { Api } from "./stores/Main";

const logsEndpoint = '/instance/logs';

export const establishConnection = async (api: Api) => {
    const logElement = document.getElementById("logs-container");

    if (logElement === null) {
        console.log("Not found container element");
        return;
    }

    const appendLogElement = (logEntry: string) => {
        const tag = document.createElement("p");

        tag.appendChild(document.createTextNode(logEntry));
        logElement.appendChild(tag);
        logElement.scrollIntoView(false);
    };

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
        const logEntry = decodeURIComponent(atob(event.data));

        appendLogElement(logEntry)
    };
};
