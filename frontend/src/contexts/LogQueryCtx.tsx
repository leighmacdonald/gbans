import { createContext, useContext } from 'react';
import { Nullable } from '../util/types';
import { noop } from 'lodash-es';

export type ServerLogQuery = {
    rate: number;
    setRate: (rate: number) => void;
    limit: number;
    setLimit: (limit: number) => void;
    eventTypes: number[];
    setEventTypes: (types: number[]) => void;
    afterDate: Nullable<Date>;
    setAfterDate: (rate: Nullable<Date>) => void;
    beforeDate: Nullable<Date>;
    setBeforeDate: (rate: Nullable<Date>) => void;
    selectedServerIDs: number[];
    setSelectedServerIDs: (ids: number[]) => void;
    cidr: string;
    setCidr: (cidr: string) => void;
    steamID: string;
    setSteamID: (cidr: string) => void;
};

export const ServerLogQueryCtx = createContext<ServerLogQuery>({
    rate: 5,
    setRate: () => noop,
    limit: 50,
    setLimit: () => noop,
    eventTypes: [],
    setEventTypes: () => noop,
    afterDate: null,
    setAfterDate: () => noop,
    beforeDate: null,
    setBeforeDate: () => noop,
    selectedServerIDs: [],
    setSelectedServerIDs: () => noop,
    cidr: '',
    setCidr: () => noop,
    steamID: '',
    setSteamID: () => noop
});

export const useServerLogQueryCtx = () => useContext(ServerLogQueryCtx);
