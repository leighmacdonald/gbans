import { Theme } from '@mui/material';
import { z } from 'zod';
import { BanReasonEnum } from './bans.ts';
import { schemaTimeStamped } from './chrono.ts';
import { schemaDemoFile } from './demo.ts';
import { schemaPerson } from './people.ts';

export const ReportStatus = {
    Any: -1,
    Opened: 0,
    NeedMoreInfo: 1,
    ClosedWithoutAction: 2,
    ClosedWithAction: 3
} as const;

export const ReportStatusEnum = z.nativeEnum(ReportStatus);
export type ReportStatusEnum = z.infer<typeof ReportStatusEnum>;

export const ReportStatusCollection = [
    ReportStatus.Any,
    ReportStatus.Opened,
    ReportStatus.NeedMoreInfo,
    ReportStatus.ClosedWithoutAction,
    ReportStatus.ClosedWithAction
];

export const reportStatusString = (rs: ReportStatusEnum): string => {
    switch (rs) {
        case ReportStatus.NeedMoreInfo:
            return 'Need More Info';
        case ReportStatus.ClosedWithoutAction:
            return 'Closed Without Action';
        case ReportStatus.ClosedWithAction:
            return 'Closed With Action';
        case ReportStatus.Opened:
            return 'Opened';
        case ReportStatus.Any:
            return 'Any';
    }
};

export const reportStatusColour = (rs: ReportStatusEnum, theme: Theme): string => {
    switch (rs) {
        case ReportStatus.NeedMoreInfo:
            return theme.palette.warning.main;
        case ReportStatus.ClosedWithoutAction:
            return theme.palette.error.main;
        case ReportStatus.ClosedWithAction:
            return theme.palette.success.main;
        default:
            return theme.palette.info.main;
    }
};

export const schemaReport = z
    .object({
        report_id: z.number(),
        source_id: z.string(),
        target_id: z.string(),
        description: z.string(),
        report_status: ReportStatusEnum,
        deleted: z.boolean(),
        reason: BanReasonEnum,
        reason_text: z.string(),
        demo_name: z.string(),
        demo_tick: z.number(),
        demo_id: z.number(),
        person_message_id: z.number()
    })
    .merge(schemaTimeStamped);

export type Report = z.infer<typeof schemaReport>;

export const schemaBasicUserInfo = z.object({
    steam_id: z.string(),
    personaname: z.string(),
    avatarhash: z.string()
});
export type BasicUserInfo = z.infer<typeof schemaBasicUserInfo>;

export const schemaBanAppealMessage = z
    .object({
        ban_id: z.number(),
        ban_message_id: z.number(),
        author_id: z.string(),
        message_md: z.string(),
        deleted: z.boolean()
    })
    .merge(schemaTimeStamped)
    .merge(schemaBasicUserInfo);

export type BanAppealMessage = z.infer<typeof schemaBanAppealMessage>;

export const schemaReportMessage = z
    .object({
        report_id: z.number(),
        report_message_id: z.number(),
        author_id: z.string(),
        message_md: z.string(),
        deleted: z.boolean()
    })
    .merge(schemaTimeStamped)
    .merge(schemaBasicUserInfo);

export type ReportMessage = z.infer<typeof schemaReportMessage>;

export const schemaCreateReportRequest = z.object({
    target_id: z.string(),
    description: z.string().min(10),
    reason: BanReasonEnum,
    reason_text: z.string(),
    demo_id: z.number(),
    demo_tick: z.number(),
    person_message_id: z.number()
});

export type CreateReportRequest = z.infer<typeof schemaCreateReportRequest>;

export const schemaReportWithAuthor = z
    .object({
        author: schemaPerson,
        subject: schemaPerson,
        demo: schemaDemoFile
    })
    .merge(schemaReport);

export type ReportWithAuthor = z.infer<typeof schemaReportWithAuthor>;

export const schemaCreateReportMessage = z.object({
    body_md: z.string()
});

export type CreateReportMessage = z.infer<typeof schemaCreateReportMessage>;
