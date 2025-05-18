import { z } from 'zod';
import { schemaQueryFilter } from './query.ts';

const DetectionTypes = [
    'unknown',
    'silent_aim',
    'aim_snap',
    'too_many_conn',
    'interp',
    'bhop',
    'cmdnum_spike',
    'eye_angles',
    'invalid_user_cmd',
    'oob_cvar',
    'cheat_cvar'
] as const;

export const Detections = z.enum(DetectionTypes);
export type Detections = z.infer<typeof Detections>;

export const schemaStacEntry = z.object({
    anticheat_id: z.number(),
    steam_id: z.string(),
    server_id: z.number(),
    server_name: z.string(),
    demo_id: z.number().optional(), // Since it's a pointer, it can be null if not set
    demo_name: z.string(),
    demo_tick: z.number(),
    name: z.string(),
    detection: Detections,
    triggered: z.number(),
    summary: z.string(),
    raw_log: z.string(),
    created_on: z.date(),
    personaname: z.string(),
    avatar: z.string(),
    query: z.string()
});

export type StacEntry = z.infer<typeof schemaStacEntry>;

export const schemaAnticheatQuery = z
    .object({
        name: z.string().optional(),
        steam_id: z.string().optional(),
        server_id: z.number().optional(),
        summary: z.string().optional(),
        detection: Detections
    })
    .merge(schemaQueryFilter);

export type AnticheatQuery = z.infer<typeof schemaAnticheatQuery>;
