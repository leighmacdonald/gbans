import { z } from 'zod/v4';

export const schemaServer = z.object({
    server_id: z.number(),
    short_name: z.string(),
    name: z.string(),
    address: z.string(),
    port: z.number(),
    password: z.string(),
    rcon: z.string(),
    region: z.string(),
    cc: z.string(),
    latitude: z.number(),
    longitude: z.number(),
    default_map: z.string(),
    reserved_slots: z.number(),
    players_max: z.number(),
    is_enabled: z.boolean(),
    colour: z.string(),
    enable_stats: z.boolean(),
    log_secret: z.number(),
    token_created_on: z.date(),
    address_internal: z.string(),
    sdr_enabled: z.boolean(),
    created_on: z.date(),
    updated_on: z.date(),
    discord_seed_role_ids: z.array(z.string())
});

export type Server = z.infer<typeof schemaServer>;

export const schemaBaseServer = z.object({
    server_id: z.number(),
    address: z.string(),
    port: z.number(),
    ip: z.string(),
    name: z.string(),
    short_name: z.string(),
    region: z.string(),
    cc: z.string(),
    players: z.number(),
    max_players: z.number(),
    bots: z.number(),
    humans: z.number(),
    map: z.string(),
    game_types: z.array(z.string()),
    latitude: z.number(),
    longitude: z.number(),
    distance: z.number() // calculated on load
});

export type BaseServer = z.infer<typeof schemaBaseServer>;

export const schemaServerSimple = z.object({
    server_id: z.number(),
    server_name: z.string(),
    server_name_long: z.string(),
    colour: z.string()
});

export type ServerSimple = z.infer<typeof schemaServerSimple>;

export const schemaLocation = z.object({
    latitude: z.number(),
    longitude: z.number()
});

export const schemaSaveServerOpts = z.object({
    short_name: z.string(),
    name: z.string(),
    address: z.string(),
    port: z.number(),
    rcon: z.string(),
    password: z.string(),
    reserved_slots: z.number(),
    region: z.string(),
    cc: z.string(),
    lat: z.number(),
    lon: z.number(),
    is_enabled: z.boolean(),
    enable_stats: z.boolean(),
    log_secret: z.number(),
    address_internal: z.string(),
    sdr_enabled: z.boolean(),
    discord_seed_role_ids: z.array(z.string())
});

export type SaveServerOpts = z.infer<typeof schemaSaveServerOpts>;

export const schemaUserServers = z.object({
    servers: z.array(schemaBaseServer),
    lat_long: schemaLocation
});

export type UserServers = z.infer<typeof schemaUserServers>;

export const schemaServerRow = schemaBaseServer.extend({
    copy: z.string(),
    connect: z.string()
});
