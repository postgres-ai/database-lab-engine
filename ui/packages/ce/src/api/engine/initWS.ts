import { InitWS } from "@postgres.ai/shared/types/api/endpoints/initWS";
import { WS_URL_PREFIX } from 'config/env'

export const initWS: InitWS = (path: string, token: string): WebSocket => {
    let url = new URL(WS_URL_PREFIX + path, window.location.href);
    url.protocol = url.protocol.replace('http', 'ws');
    const wsAddr = url.href + '?token=' + token;

    return new WebSocket(wsAddr)
}
