import { apiCall, AuthorQueryFilter, TimeStamped } from './common';
import { Person, UserProfile } from './profile';
import { SteamID } from './const';
import { Theme } from '@mui/material';
import { BanReason } from './bans';

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

export interface Report extends TimeStamped {
    report_id: number;
    author_id: SteamID;
    reported_id: SteamID;
    description: string;
    report_status: ReportStatus;
    deleted: boolean;
    reason: BanReason;
    reason_text: string;
}

export interface ReportMessagesResponse {
    message: UserMessage;
    author: UserProfile;
}

export interface UserMessage extends TimeStamped {
    parent_id: number;
    message_id: number;
    author_id: SteamID;
    contents: string;
    deleted: boolean;
}

export interface CreateReportRequest {
    steam_id: SteamID;
    description: string;
    reason: BanReason;
    reason_text: string;
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

export const apiGetReports = async (opts?: AuthorQueryFilter<Report>) =>
    await apiCall<ReportWithAuthor[], AuthorQueryFilter<Report>>(
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
    await apiCall<UserMessage, CreateReportMessage>(
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
