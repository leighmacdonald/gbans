import { JsonObject } from 'react-use-websocket/dist/lib/types';
import { SlimServer } from './server';
import { wsValue } from './ws';
import { wsMsgTypePugUserMessageResponse } from '../pug/pug';

export enum qpMsgType {
    qpMsgTypeJoinLobbyRequest = 0,
    qpMsgTypeLeaveLobbyRequest,
    qpMsgTypeJoinLobbySuccess,
    qpMsgTypeSendMsgRequest
}

interface qpClient extends JsonObject {
    leader: boolean;
    user: undefined;
}

export interface qpLobby extends JsonObject {
    lobby_id: string;
    clients: qpClient[];
}

export interface qpMsgJoinedLobbySuccessI extends JsonObject {
    lobby: qpLobby;
}

export interface qpMsgJoinLobbyRequestI extends JsonObject {
    lobby_id: string;
}

export type qpUserMessage = wsValue<wsMsgTypePugUserMessageResponse>;
export type qpMsgJoinLobbyRequest = wsValue<qpMsgJoinLobbyRequestI>;
export type qpMsgLeaveLobbyRequest = wsValue<qpMsgJoinLobbyRequestI>;
export type qpMsgJoinedLobbySuccess = wsValue<qpMsgJoinedLobbySuccessI>;

export type qpRequestTypes =
    | qpMsgJoinLobbyRequest
    | qpUserMessage
    | qpMsgJoinedLobbySuccess
    | qpMsgLeaveLobbyRequest;

export type qpAutoQueueMode = 'eager' | 'full';

export interface qpGameType {
    name: string;
    map_filters: RegExp[];
}

export const qpKnownGameTypes: qpGameType[] = [
    {
        map_filters: [/^pl_.+?/],
        name: 'Payload'
    },
    {
        map_filters: [/^cp_.+?/],
        name: 'Control Points'
    },
    {
        map_filters: [/^koth_.+?/],
        name: 'King of the Hill'
    },
    {
        map_filters: [/^mvm_.+?/],
        name: 'Mann-vs-Machine'
    },
    {
        map_filters: [/^plr_.+?/],
        name: 'Payload Race'
    },
    {
        map_filters: [/^pd_.+?/],
        name: 'Player Destruction'
    },
    {
        map_filters: [/^rd_.+?/],
        name: 'Robot Destruction'
    },
    {
        map_filters: [/^arena_.+?/],
        name: 'Arena'
    },
    {
        map_filters: [/^ctf_.+?/],
        name: 'CTF'
    },
    {
        map_filters: [/^pass_.+?/],
        name: 'Passtime'
    },
    {
        map_filters: [/^mann_.+?/],
        name: 'Mannpower'
    },
    {
        map_filters: [/^sd_.+?/],
        name: 'Special Delivery'
    },
    {
        map_filters: [/^tc_.+?/],
        name: 'Territory Control'
    },
    {
        map_filters: [/^cp_degrootkeep$/],
        name: 'Medieval Mode'
    }
];

export const filterServerGameTypes = (
    allowedTypes: qpGameType[],
    servers: SlimServer[]
): SlimServer[] => {
    if (allowedTypes.length == 0) {
        return servers;
    }
    const matched = [];
    // eslint-disable-next-line no-loops/no-loops
    for (let si = 0; si < servers.length; si++) {
        let found = false;
        const server = servers[si];
        // eslint-disable-next-line no-loops/no-loops
        for (let gti = 0; gti < allowedTypes.length; gti++) {
            if (found) break;
            const allowedType = allowedTypes[gti];
            // eslint-disable-next-line no-loops/no-loops
            for (let mfi = 0; mfi < allowedType.map_filters.length; mfi++) {
                if (found) break;
                const at = allowedType.map_filters[mfi];
                if (at.test(server.map)) {
                    matched.push(server);
                    found = true;
                    break;
                }
            }
        }
    }
    return matched;
};
