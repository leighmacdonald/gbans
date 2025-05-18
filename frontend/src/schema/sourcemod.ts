import { z } from 'zod';
import { schemaTimeStamped } from './chrono.ts';
import { schemaLocation } from './server.ts';

export const AuthType = z.enum(['steam', 'name', 'ip']);
export type AuthType = z.infer<typeof AuthType>;

export const OverrideType = z.enum(['command', 'group']);
export type OverrideType = z.infer<typeof OverrideType>;

export const OverrideAccess = z.enum(['allow', 'deny']);
export type OverrideAccess = z.infer<typeof OverrideAccess>;

export const Flags = z.enum([
    'z',
    'a',
    'b',
    'c',
    'd',
    'e',
    'f',
    'g',
    'h',
    'i',
    'j',
    'k',
    'l',
    'm',
    'n',
    'o',
    'p',
    'q',
    'r',
    's',
    't'
]);

export type Flags = z.infer<typeof Flags>;

export const schemaSMGroups = z
    .object({
        group_id: z.number(),
        flags: z.string(),
        name: z.string().min(1),
        immunity_level: z.number().min(0).max(100)
    })
    .merge(schemaTimeStamped);

export const schemaSMAdmin = z
    .object({
        admin_id: z.number(),
        steam_id: z.string(),
        auth_type: AuthType,
        identity: z.string(),
        password: z.string(),
        flags: z.string(),
        name: z.string(),
        immunity: z.number().min(0).max(100),
        groups: z.array(schemaSMGroups)
    })
    .merge(schemaTimeStamped);

export const schemaSMGroupImmunity = z.object({
    group_immunity_id: z.number(),
    group: schemaSMGroups,
    other: schemaSMGroups,
    created_on: z.date()
});

export const schemaSMOverrides = z
    .object({
        override_id: z.number(),
        type: OverrideType,
        name: z.string(),
        flags: z.string()
    })
    .merge(schemaTimeStamped);

export const schemaSMGroupOverrides = z
    .object({
        group_override_id: z.number(),
        group_id: z.number(),
        type: OverrideType,
        name: z.string(),
        access: OverrideAccess
    })
    .merge(schemaTimeStamped);

export const schemaSMAdminGroups = z.object({
    admin_id: z.number(),
    group_id: z.number(),
    inherit_order: z.number()
});
export type SMAdminGroups = z.infer<typeof schemaSMAdminGroups>;

export type SMGroupOverrides = z.infer<typeof schemaSMGroupOverrides>;
export type SMOverrides = z.infer<typeof schemaSMOverrides>;
export type SMAdmin = z.infer<typeof schemaSMAdmin>;
export type SMGroups = z.infer<typeof schemaSMGroups>;
export type SMGroupImmunity = z.infer<typeof schemaSMGroupImmunity>;

export type Location = z.infer<typeof schemaLocation>;
