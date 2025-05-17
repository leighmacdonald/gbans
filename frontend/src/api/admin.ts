import { z } from 'zod';
import {
    schemaAnticheat,
    schemaDebug,
    schemaDemos,
    schemaDiscord,
    schemaExports,
    schemaFilters,
    schemaGeneral,
    schemaGeo,
    schemaLocalStore,
    schemaLogging,
    schemaNetwork,
    schemaPatreon,
    schemaSentry,
    schemaSSH
} from '../schema/config.ts';
import { apiCall } from './common.ts';

export const apiSaveSettings = async (settings: Config) => {
    return await apiCall(`/api/config`, 'PUT', settings);
};

export const apiGetSettings = async () => {
    return await apiCall<Config>('/api/config', 'GET');
};

export type Config = {
    general: z.infer<typeof schemaGeneral>;
    filters: z.infer<typeof schemaFilters>;
    demo: z.infer<typeof schemaDemos>;
    patreon: z.infer<typeof schemaPatreon>;
    discord: z.infer<typeof schemaDiscord>;
    network: z.infer<typeof schemaNetwork>;
    log: z.infer<typeof schemaLogging>;
    sentry: z.infer<typeof schemaSentry>;
    geo_location: z.infer<typeof schemaGeo>;
    debug: z.infer<typeof schemaDebug>;
    local_store: z.infer<typeof schemaLocalStore>;
    ssh: z.infer<typeof schemaSSH>;
    exports: z.infer<typeof schemaExports>;
    anticheat: z.infer<typeof schemaAnticheat>;
};

export enum Action {
    Ban = 'ban',
    Kick = 'kick',
    Gag = 'gag'
}

export const ActionColl = [Action.Ban, Action.Kick, Action.Gag];
