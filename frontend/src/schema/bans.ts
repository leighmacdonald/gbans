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
    dur15m: 'PT15M',
    dur6h: 'P6H',
    dur12h: 'P12H',
    dur24h: 'P1D',
    dur48h: 'P2D',
    dur72h: 'P3D',
    dur1w: 'P1W',
    dur2w: 'P2W',
    dur1M: 'P1M',
    dur6M: 'P6M',
    dur1y: 'P1y',
    durInf: 'P0',
    durCustom: ''
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

export const schemaBan = schemaTimeStamped.extend({
    target_id: z.string(),
    source_id: z.string(),
    ban_id: z.number(),
    report_id: z.number(),
    last_ip: z.ipv4().optional(),
    evade_ok: z.boolean(),
    ban_type: BanTypeEnum,
    reason: BanReasonEnum,
    reason_text: z.string(),
    unban_reason_text: z.string(),
    note: z.string(),
    origin: OriginEnum,
    cidr: z.cidrv4().optional(),
    appeal_state: AppealStateEnum,
    name: z.string(),
    deleted: z.boolean(),

    valid_until: z.date(),
    created_on: z.date(),
    updated_on: z.date(),

    source_personaname: z.string(),
    source_avatarhash: z.string(),
    target_personaname: z.string(),
    target_avatarhash: z.string()
});

export type BanRecord = z.infer<typeof schemaBan>;

export const schemaUnbanPayload = z.object({
    unban_reason_text: z.string()
});

export type UnbanPayload = z.infer<typeof schemaUnbanPayload>;

export const schemaBanPayload = z.object({
    reason: BanReasonEnum,
    reason_text: z.string(),
    source_id: z.string().optional(),
    target_id: z.string(),
    duration: z.string(),
    note: z.string(),
    report_id: z.number().optional(),
    evade_ok: z.boolean(),
    ban_type: BanTypeEnum,
    cidr: z.cidrv4().optional(),
    demo_name: z.string(),
    demo_tick: z.number(),
    origin: OriginEnum
});

export type BanOpts = z.infer<typeof schemaBanPayload>;

export const schemaSbBanRecord = z.object({
    ban_id: z.number(),
    site_id: z.number(),
    site_name: z.string(),
    persona_name: z.string(),
    steam_id: z.string(),
    reason: z.string(),
    duration: z.number(),
    permanent: z.boolean(),
    created_on: z.date()
});

export type sbBanRecord = z.infer<typeof schemaSbBanRecord>;

export const schemaAppealQueryFilter = z.object({
    deleted: z.boolean().optional()
});

export type AppealQueryFilter = z.infer<typeof schemaAppealQueryFilter>;

export const schemaUpdateBanPayload = z.object({
    reason: BanReasonEnum,
    reason_text: z.string(),
    note: z.string(),
    valid_until: z.date().optional(),
    evade_ok: z.boolean(),
    ban_type: BanTypeEnum
});

export type UpdateBanPayload = z.infer<typeof schemaUpdateBanPayload>;

export const schemaUpdateBodyMD = z.object({
    body_md: z.string()
});

export type BodyMDMessage = z.infer<typeof schemaUpdateBodyMD>;
