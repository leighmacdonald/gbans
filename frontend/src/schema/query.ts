import { z } from 'zod/v4';
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

export const schemaBanCIDRQueryFilter = schemaBanQueryCommon.extend({
    ip: z.ipv4().optional()
});

export type BanCIDRQueryFilter = z.infer<typeof schemaBanCIDRQueryFilter>;

export const schemaBanGroupQueryFilter = schemaBanQueryCommon.extend({
    group_id: z.number().optional()
});

export type BanGroupQueryFilter = z.infer<typeof schemaBanGroupQueryFilter>;

export const schemaBanASNQueryFilter = schemaBanQueryCommon.extend({
    as_num: z.number().optional()
});

export type BanASNQueryFilter = z.infer<typeof schemaBanASNQueryFilter>;

export const schemaReportQueryFilter = z.object({
    deleted: z.boolean().optional(),
    source_id: z.string().optional()
});

export type ReportQueryFilter = z.infer<typeof schemaReportQueryFilter>;

export const schemaCallbackLink = z.object({
    url: z.url()
});

export type CallbackLink = z.infer<typeof schemaCallbackLink>;
