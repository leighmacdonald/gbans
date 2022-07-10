import { apiCall, AuthorQueryFilter, TimeStamped } from './common';
import { Person, UserProfile } from './profile';
import { SteamID } from './const';

export enum ReportStatus {
    Opened,
    NeedMoreInfo,
    ClosedWithoutAction,
    ClosedWithAction
}

export enum BanReason {
    Custom = 1,
    External = 2,
    Cheating = 3,
    Racism = 4,
    Harassment = 5,
    Exploiting = 6,
    WarningsExceeded = 7,
    Spam = 8,
    Language = 9
}

export const BanReasons: Record<BanReason, string> = {
    [BanReason.Custom]: 'Custom',
    [BanReason.External]: '3rd party',
    [BanReason.Cheating]: 'Cheating',
    [BanReason.Racism]: 'Racism',
    [BanReason.Harassment]: 'Person Harassment',
    [BanReason.Exploiting]: 'Exploiting',
    [BanReason.WarningsExceeded]: 'Warnings Exceeding',
    [BanReason.Spam]: 'Spam',
    [BanReason.Language]: 'Language'
};

export interface Report extends TimeStamped {
    report_id: number;
    reported_id: number;
    title: string;
    description: string;
    report_status: ReportStatus;
    deleted: boolean;
    media_ids?: number[];
}

export interface BaseUploadedMedia extends TimeStamped {
    author_id: number;
    mime_type: string;
    size: number;
    name: string;
    contents: Uint8Array;
    deleted: boolean;
}

export interface ReportMedia extends BaseUploadedMedia {
    report_media_id: number;
    report_id: number;
}

export interface ReportMessagesResponse {
    message: ReportMessage;
    author: UserProfile;
}

export interface ReportMessage extends TimeStamped {
    report_message_id: number;
    report_id: number;
    author_id: number;
    contents: string;
    deleted: boolean;
}

export interface Appeal extends TimeStamped {
    appeal_id: number;
}

export interface UserUploadedFile {
    content: string;
    name: string;
    mime: string;
    size: number;
}

export interface CreateReportRequest {
    steam_id: SteamID;
    title: string;
    description: string;
    media: UserUploadedFile[];
}

export type BanState = 'banned' | 'closed';

export interface ReportWithAuthor {
    author: Person;
    report: Report;
}

export const apiCreateReport = async (opts: CreateReportRequest) =>
    await apiCall<Report, CreateReportRequest>('/api/report', 'POST', opts);

export const apiGetReport = async (report_id: number) =>
    await apiCall<ReportWithAuthor>(`/api/report/${report_id}`, 'GET');

export const apiGetReports = async (opts: AuthorQueryFilter) =>
    await apiCall<ReportWithAuthor[], AuthorQueryFilter>(
        `/api/reports`,
        'POST',
        opts
    );

export const apiGetReportMessages = async (report_id: number) =>
    await apiCall<ReportMessagesResponse[], CreateReportRequest>(
        `/api/report/${report_id}/messages`,
        'GET'
    );

export interface GetLogsRequest {
    steam_id: string;
    limit: number;
}

export interface UserMessageLog extends TimeStamped {
    created_on: Date;
    message: string;
}

export const apiGetLogs = async (steam_id: string, limit: number) =>
    await apiCall<UserMessageLog[], GetLogsRequest>(`/api/logs/query`, 'POST', {
        limit,
        steam_id
    });

export interface CreateReportMessage {
    message: string;
}

export const apiCreateReportMessage = async (
    report_id: number,
    message: string
) =>
    await apiCall<ReportMessage, CreateReportMessage>(
        `/api/report/${report_id}/messages`,
        'POST',
        { message }
    );

export const apiReportSetState = async (
    report_id: number,
    stateAction: ReportStatus
) =>
    await apiCall(`/api/report_status/${report_id}`, 'POST', {
        status: stateAction
    });
