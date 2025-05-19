import { z } from 'zod';
import { AppealStateEnum } from './bans.ts';

export const schemaQueryFilter = z.object({
    offset: z.number().optional(),
    limit: z.number().optional(),
    desc: z.boolean().optional(),
    query: z.string().optional(),
    order_by: z.string().optional(),
    deleted: z.boolean().optional(),
    flagged_only: z.boolean().optional()
});

export const schemaBanQueryCommon = z
    .object({
        source_id: z.string().optional(),
        target_id: z.string().optional(),
        appeal_state: AppealStateEnum.optional(),
        deleted: z.boolean().optional()
    })
    .merge(schemaQueryFilter);

export type BanQueryCommon = z.infer<typeof schemaBanQueryCommon>;

export type BanSteamQueryFilter = BanQueryCommon;

export const schemaBanCIDRQueryFilter = z
    .object({
        ip: z.string().ip({ version: 'v4' }).optional()
    })
    .merge(schemaBanQueryCommon);

export type BanCIDRQueryFilter = z.infer<typeof schemaBanCIDRQueryFilter>;

export const schemaBanGroupQueryFilter = z
    .object({
        group_id: z.number().optional()
    })
    .merge(schemaBanQueryCommon);

export type BanGroupQueryFilter = z.infer<typeof schemaBanGroupQueryFilter>;

export const schemaBanASNQueryFilter = z
    .object({
        as_num: z.number().optional()
    })
    .merge(schemaBanQueryCommon);

export type BanASNQueryFilter = z.infer<typeof schemaBanASNQueryFilter>;

export const schemaReportQueryFilter = z.object({
    deleted: z.boolean().optional(),
    source_id: z.string().optional()
});

export type ReportQueryFilter = z.infer<typeof schemaReportQueryFilter>;

export const schemaCallbackLink = z.object({
    url: z.string().url()
});

export type CallbackLink = z.infer<typeof schemaCallbackLink>;
