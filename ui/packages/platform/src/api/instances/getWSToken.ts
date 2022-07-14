/*--------------------------------------------------------------------------
 * Copyright (c) 2019-2022, Postgres.ai, Nikolay Samokhvalov nik@postgres.ai
 * All Rights Reserved. Proprietary and confidential.
 * Unauthorized copying of this file, via any medium is strictly prohibited
 *--------------------------------------------------------------------------
 */

import { request } from 'helpers/request';
import { localStorage } from "helpers/localStorage";

import { GetWSToken } from "@postgres.ai/shared/types/api/endpoints/getWSToken";
import { formatWSTokenDto, WSTokenDTO } from "@postgres.ai/shared/types/api/entities/wsToken";
import { request as requestCore } from "@postgres.ai/shared/helpers/request";
import { formatInstanceDto, InstanceDto } from "@postgres.ai/shared/types/api/entities/instance";

export const getWSToken: GetWSToken = async (req) => {
    // TODO: define instance and get a websocket token.
    const instanceResponse = await request('/dblab_instances', {
        params: {
            id: `eq.${req.instanceId}`,
        },
    })

    if (!instanceResponse.ok) {
        return {
            response: null,
            error: instanceResponse,
        }
    }

    const instance = (await instanceResponse.json() as InstanceDto[]).map(formatInstanceDto)[0]

    const authToken = localStorage.getAuthToken()

    if (instance.useTunnel) {
        return {
            response: null,
            error: new Response(null, {
                status: 400,
                statusText: `Cannot connect to an instance that is using a tunnel`,
            })
        }
    }

    const response = await requestCore(instance.url + '/admin/ws-auth', {
        headers: {
            ...(authToken && {'Verification-Token': authToken}),
        },
    })

    return {
        response: response.ok
            ? formatWSTokenDto((await response.json()) as WSTokenDTO)
            : null,
        error: response.ok ? null : response,
    }
}


