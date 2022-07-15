import { apiCall, AuthorQueryFilter, TimeStamped } from './common';
import { Person, UserProfile } from './profile';
import { SteamID } from './const';
import { Theme } from '@mui/material';

export enum ReportStatus {
    Any = -1,
    Opened = 0,
    NeedMoreInfo = 1,
    ClosedWithoutAction = 2,
    ClosedWithAction = 3
}

export const reportStatusString = (rs: ReportStatus): string => {
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
export const reportStatusColour = (rs: ReportStatus, theme: Theme): string => {
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
export enum BanReason {
    Custom = 1,
    External = 2,
    Cheating = 3,
    Racism = 4,
    Harassment = 5,
    Exploiting = 6,
    WarningsExceeded = 7,
    Spam = 8,
    Language = 9,
    Profile = 10,
    ItemDescriptions = 11,
    BotHost = 12
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
    [BanReason.Language]: 'Language',
    [BanReason.Profile]: 'Profile',
    [BanReason.ItemDescriptions]: 'Item Name/Descriptions',
    [BanReason.BotHost]: 'Bot Host'
};

export const banReasonsList = [
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
    BanReason.Custom
];

export interface Report extends TimeStamped {
    report_id: number;
    author_id: SteamID;
    reported_id: SteamID;
    description: string;
    report_status: ReportStatus;
    deleted: boolean;
}

export interface ReportMessagesResponse {
    message: ReportMessage;
    author: UserProfile;
}

export interface ReportMessage extends TimeStamped {
    report_message_id: number;
    report_id: number;
    author_id: SteamID;
    contents: string;
    deleted: boolean;
}

export interface Appeal extends TimeStamped {
    appeal_id: number;
}

export interface CreateReportRequest {
    steam_id: SteamID;
    description: string;
}

export interface ReportWithAuthor {
    author: Person;
    report: Report;
    subject: Person;
}

export const apiCreateReport = async (opts: CreateReportRequest) =>
    await apiCall<Report, CreateReportRequest>('/api/report', 'POST', opts);

export const apiGetReport = async (report_id: number) =>
    await apiCall<ReportWithAuthor>(`/api/report/${report_id}`, 'GET');

export const apiGetReports = async (opts?: AuthorQueryFilter) =>
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

export const apiUpdateReportMessage = async (
    report_message_id: number,
    message: string
) =>
    await apiCall(`/api/report/message/${report_message_id}`, 'POST', {
        body_md: message
    });

export const apiDeleteReportMessage = async (report_message_id: number) =>
    await apiCall(`/api/report/message/${report_message_id}`, 'DELETE', {});
