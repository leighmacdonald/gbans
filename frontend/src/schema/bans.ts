import { z } from 'zod/v4';
import { schemaTimeStamped } from './chrono.ts';

export const Origin = {
    System: 0,
    Bot: 1,
    Web: 2,
    InGame: 3
} as const;

export const OriginEnum = z.enum(Origin);
export type OriginEnum = z.infer<typeof OriginEnum>;

export const BanReason = {
    Any: -1,
    Custom: 1,
    External: 2,
    Cheating: 3,
    Racism: 4,
    Harassment: 5,
    Exploiting: 6,
    WarningsExceeded: 7,
    Spam: 8,
    Language: 9,
    Profile: 10,
    ItemDescriptions: 11,
    BotHost: 12,
    Evading: 13,
    Username: 14
} as const;

export const BanReasonEnum = z.enum(BanReason);
export type BanReasonEnum = z.infer<typeof BanReasonEnum>;

export const Duration = {
    dur15m: '15m',
    dur6h: '6h',
    dur12h: '12h',
    dur24h: '24h',
    dur48h: '48h',
    dur72h: '72h',
    dur1w: '1w',
    dur2w: '2w',
    dur1M: '1M',
    dur6M: '6M',
    dur1y: '1y',
    durInf: '0',
    durCustom: 'custom'
} as const;

export const DurationEnum = z.enum(Duration);
export type DurationEnum = z.infer<typeof DurationEnum>;

export const DurationCollection = [
    Duration.dur15m,
    Duration.dur6h,
    Duration.dur12h,
    Duration.dur24h,
    Duration.dur48h,
    Duration.dur72h,
    Duration.dur1w,
    Duration.dur2w,
    Duration.dur1M,
    Duration.dur6M,
    Duration.dur1y,
    Duration.durInf,
    Duration.durCustom
];

export const BanReasons: Record<BanReasonEnum, string> = {
    [BanReason.Any]: 'Any',
    [BanReason.Custom]: 'Custom',
    [BanReason.External]: '3rd party',
    [BanReason.Cheating]: 'Cheating',
    [BanReason.Racism]: 'Racism',
    [BanReason.Harassment]: 'Personal Harassment',
    [BanReason.Exploiting]: 'Exploiting',
    [BanReason.WarningsExceeded]: 'Warnings Exceeded',
    [BanReason.Spam]: 'Spam',
    [BanReason.Language]: 'Language',
    [BanReason.Profile]: 'Inappropriate Steam Profile',
    [BanReason.ItemDescriptions]: 'Item Name/Descriptions',
    [BanReason.BotHost]: 'Bot Host',
    [BanReason.Evading]: 'Evading',
    [BanReason.Username]: 'Inappropriate Username'
};

export const banReasonsCollection = [
    BanReason.Any,
    BanReason.Cheating,
    BanReason.Racism,
    BanReason.Harassment,
    BanReason.Exploiting,
    BanReason.WarningsExceeded,
    BanReason.Spam,
    BanReason.Language,
    BanReason.Profile,
    BanReason.ItemDescriptions,
    BanReason.External,
    BanReason.Custom,
    BanReason.BotHost,
    BanReason.Evading,
    BanReason.Username
];

export const banReasonsReportCollection = [
    BanReason.Cheating,
    BanReason.Racism,
    BanReason.Harassment,
    BanReason.Exploiting,
    BanReason.WarningsExceeded,
    BanReason.Spam,
    BanReason.Language,
    BanReason.Profile,
    BanReason.ItemDescriptions,
    BanReason.External,
    BanReason.Custom,
    BanReason.BotHost,
    BanReason.Evading,
    BanReason.Username
];
export const BanType = {
    Unknown: -1,
    OK: 0,
    NoComm: 1,
    Banned: 2
} as const;

export const BanTypeEnum = z.enum(BanType);
export type BanTypeEnum = z.infer<typeof BanTypeEnum>;

export const BanTypeCollection = [BanType.OK, BanType.NoComm, BanType.Banned];

export const AppealState = {
    Any: -1,
    Open: 0,
    Denied: 1,
    Accepted: 2,
    Reduced: 3,
    NoAppeal: 4
} as const;

export const AppealStateEnum = z.enum(AppealState);
export type AppealStateEnum = z.infer<typeof AppealStateEnum>;

export const AppealStateCollection = [
    AppealState.Any,
    AppealState.Open,
    AppealState.Denied,
    AppealState.Accepted,
    AppealState.Reduced,
    AppealState.NoAppeal
];

const schemaBanBase = schemaTimeStamped.extend({
    valid_until: z.date(),
    reason: BanReasonEnum,
    ban_type: BanTypeEnum,
    reason_text: z.string(),
    source_id: z.string(),
    target_id: z.string(),
    deleted: z.boolean(),
    unban_reason_text: z.string(),
    note: z.string(),
    origin: OriginEnum,
    appeal_state: AppealStateEnum,
    source_personaname: z.string(),
    source_avatarhash: z.string(),
    target_personaname: z.string(),
    target_avatarhash: z.string()
});

export const schemaSteamBanRecord = schemaBanBase.extend({
    ban_id: z.number(),
    report_id: z.number(),
    ban_type: BanTypeEnum,
    include_friends: z.boolean(),
    evade_ok: z.boolean()
});

export type SteamBanRecord = z.infer<typeof schemaSteamBanRecord>;

export const schemaGroupBanRecord = schemaBanBase.extend({
    ban_group_id: z.number(),
    group_id: z.string(),
    group_name: z.string()
});

export type GroupBanRecord = z.infer<typeof schemaGroupBanRecord>;

export const schemaCIDRBanRecord = schemaBanBase.extend({
    net_id: z.number(),
    cidr: z.cidrv4()
});

export type CIDRBanRecord = z.infer<typeof schemaCIDRBanRecord>;

export const schemaASNBanRecord = schemaBanBase.extend({
    ban_asn_id: z.number(),
    as_num: z.number().positive()
});

export type ASNBanRecord = z.infer<typeof schemaASNBanRecord>;

export const schemaUnbanPayload = z.object({
    unban_reason_text: z.string()
});

export type UnbanPayload = z.infer<typeof schemaUnbanPayload>;

const schemaBanBasePayload = z.object({
    reason: BanReasonEnum,
    reason_text: z.string(),
    target_id: z.string(),
    duration: DurationEnum,
    valid_until: z.date().optional(),
    note: z.string()
});

export const schemaBanPayloadSteam = schemaBanBasePayload.extend({
    report_id: z.number().optional(),
    include_friends: z.boolean(),
    evade_ok: z.boolean(),
    ban_type: BanTypeEnum,
    duration_custom: z.date().optional()
});

export type BanPayloadSteam = z.infer<typeof schemaBanPayloadSteam>;

export const schemaBanPayloadCIDR = schemaBanBasePayload.extend({
    cidr: z.cidrv4()
});

export type BanPayloadCIDR = z.infer<typeof schemaBanPayloadCIDR>;

export const schemaBanPayloadASN = schemaBanBasePayload.extend({
    as_num: z.number()
});

export type BanPayloadASN = z.infer<typeof schemaBanPayloadASN>;

export const schemaBanPayloadGroup = z.object({
    target_id: z.string(),
    duration: DurationEnum,
    valid_until: z.date().optional(),
    note: z.string(),
    group_id: z.string()
});

export type BanPayloadGroup = z.infer<typeof schemaBanPayloadGroup>;

export const schemaSbBanRecord = z.object({
    ban_id: z.number(),
    site_id: z.number(),
    site_name: z.string(),
    persona_name: z.string(),
    steam_id: z.string(),
    reason: z.string(),
    duration: z.number(),
    permanent: z.string(),
    created_on: z.date()
});

export type sbBanRecord = z.infer<typeof schemaSbBanRecord>;

export const schemaAppealQueryFilter = z.object({
    deleted: z.boolean().optional()
});

export type AppealQueryFilter = z.infer<typeof schemaAppealQueryFilter>;

const schemaUpdateBanPayload = z.object({
    reason: BanReasonEnum,
    reason_text: z.string(),
    note: z.string(),
    valid_until: z.date().optional()
});

export type UpdateBanPayload = z.infer<typeof schemaUpdateBanPayload>;

export const schemaUpdateBanSteamPayload = z
    .object({
        include_friends: z.boolean(),
        evade_ok: z.boolean(),
        ban_type: BanTypeEnum
    })
    .merge(schemaUpdateBanPayload);

export type UpdateBanSteamPayload = z.infer<typeof schemaUpdateBanSteamPayload>;

export const schemaUpdateBanASNPayload = z.object({
    target_id: z.string(),
    reason: BanReasonEnum,
    as_num: z.number(),
    reason_text: z.string(),
    note: z.string(),
    valid_until: z.date().optional()
});

export type UpdateBanASNPayload = z.infer<typeof schemaUpdateBanASNPayload>;

export const schemaUpdateBanGroupPayload = z.object({
    target_id: z.string(),
    note: z.string(),
    valid_until: z.date().optional()
});

export type UpdateBanGroupPayload = z.infer<typeof schemaUpdateBanGroupPayload>;

export const schemaUpdateBodyMD = z.object({
    body_md: z.string()
});

export type BodyMDMessage = z.infer<typeof schemaUpdateBodyMD>;
