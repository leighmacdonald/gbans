import { apiCall, TimeStamped } from './common';

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
    media_ids: number[];
}

export interface ReportMedia extends TimeStamped {
    report_media_id: number;
    report_id: number;
    author_id: number;
    mime_type: string;
    size: number;
    contents: Uint8Array;
    deleted: boolean;
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

export interface UploadedFile {
    content: string;
    name: string;
    mime: string;
    size: number;
}
export interface CreateReportRequest {
    steam_id: string;
    title: string;
    description: string;
    media: UploadedFile[];
}

export type BanState = 'banned' | 'closed';

export interface UserReportHistory {
    name: string;
    target: string;
    target_avatar: string;
    state: BanState;
    updated_on: Date;
    created_on: Date;
}

export const apiCreateReport = async (opts: CreateReportRequest) => {
    const resp = await apiCall<Report, CreateReportRequest>(
        '/api/report',
        'POST',
        opts
    );
    return resp.json as Report;
};

export const apiGetReport = async (report_id: number) => {
    const resp = await apiCall<Report, CreateReportRequest>(
        `/api/report/${report_id}`,
        'GET'
    );
    return resp.json as Report;
};

export const apiGetReportMessages = async (report_id: number) => {
    const resp = await apiCall<Report, CreateReportRequest>(
        `/api/report/${report_id}/messages`,
        'GET'
    );
    return resp.json as ReportMessage[];
};
