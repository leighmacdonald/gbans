import { z } from 'zod/v4';
import { BanReasonEnum } from './bans.ts';

export const FilterAction = {
    Kick: 0,
    Mute: 1,
    Ban: 2
} as const;

export const FilterActionEnum = z.nativeEnum(FilterAction);
export type FilterActionEnum = z.infer<typeof FilterActionEnum>;

export const FilterActionCollection = [FilterAction.Kick, FilterAction.Mute, FilterAction.Ban];

export const filterActionString = (fa: FilterActionEnum) => {
    switch (fa) {
        case FilterAction.Ban:
            return 'Ban';
        case FilterAction.Kick:
            return 'Kick';
        case FilterAction.Mute:
            return 'Mute';
    }
};

export const schemaFilter = z.object({
    filter_id: z.number().optional(),
    author_id: z.string().optional(),
    pattern: z.string(),
    is_regex: z.boolean(),
    is_enabled: z.boolean().optional(),
    trigger_count: z.number().optional(),
    action: FilterActionEnum,
    duration: z.string(),
    weight: z.number(),
    created_on: z.date().optional(),
    updated_on: z.date().optional()
});

export type Filter = z.infer<typeof schemaFilter>;

export const schemaUserWarning = z.object({
    warn_reason: BanReasonEnum,
    message: z.string(),
    matched: z.string(),
    matched_filter: schemaFilter,
    created_on: z.date(),
    personaname: z.string(),
    avatar: z.string(),
    server_name: z.string(),
    server_id: z.number(),
    steam_id: z.string(),
    current_total: z.number()
});

export type UserWarning = z.infer<typeof schemaUserWarning>;

export const schemaWarningState = z.object({
    max_weight: z.number(),
    current: z.array(schemaUserWarning)
});

export type WarningState = z.infer<typeof schemaWarningState>;
