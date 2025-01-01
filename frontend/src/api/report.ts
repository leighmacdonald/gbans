import { Theme } from '@mui/material';
import { TimeStamped, transformTimeStampedDates, transformTimeStampedDatesList } from '../util/time.ts';
import { BanReason } from './bans';
import { apiCall, ReportQueryFilter } from './common';
import { DemoFile } from './demo.ts';
import { Person } from './profile';

export enum ReportStatus {
    Any = -1,
    Opened = 0,
    NeedMoreInfo = 1,
    ClosedWithoutAction = 2,
    ClosedWithAction = 3
}

export const ReportStatusCollection = [
    ReportStatus.Any,
    ReportStatus.Opened,
    ReportStatus.NeedMoreInfo,
    ReportStatus.ClosedWithoutAction,
    ReportStatus.ClosedWithAction
];

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
    source_id: string;
    target_id: string;
    description: string;
    report_status: ReportStatus;
    deleted: boolean;
    reason: BanReason;
    reason_text: string;
    demo_name: string;
    demo_tick: number;
    demo_id: number;
    person_message_id: number;
}

export interface BasicUserInfo {
    steam_id: string;
    personaname: string;
    avatarhash: string;
}

export interface BanAppealMessage extends TimeStamped, BasicUserInfo {
    ban_id: number;
    ban_message_id: number;
    author_id: string;
    message_md: string;
    deleted: boolean;
}

export interface ReportMessage extends TimeStamped, BasicUserInfo {
    report_id: number;
    report_message_id: number;
    author_id: string;
    message_md: string;
    deleted: boolean;
}

export interface CreateReportRequest {
    target_id: string;
    description: string;
    reason: BanReason;
    reason_text: string;
    demo_id: number;
    demo_tick: number;
    person_message_id: number;
}

export interface ReportWithAuthor extends Report {
    author: Person;
    subject: Person;
    demo: DemoFile;
}

export const apiCreateReport = async (opts: CreateReportRequest) =>
    await apiCall<Report, CreateReportRequest>('/api/report', 'POST', opts);

export const apiGetReport = async (report_id: number, abortController?: AbortController) =>
    await apiCall<ReportWithAuthor>(`/api/report/${report_id}`, 'GET', abortController);

export const apiGetReports = async (opts?: ReportQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<ReportWithAuthor[], ReportQueryFilter>(`/api/reports`, 'POST', opts, abortController);
    return resp.map(transformTimeStampedDates);
};

export const apiGetUserReports = async (abortController?: AbortController) => {
    const resp = await apiCall<ReportWithAuthor[], ReportQueryFilter>(
        `/api/reports/user`,
        'GET',
        undefined,
        abortController
    );
    return resp.map(transformTimeStampedDates);
};

export const apiGetReportMessages = async (report_id: number, abortController?: AbortController) =>
    transformTimeStampedDatesList(
        await apiCall<ReportMessage[], CreateReportRequest>(
            `/api/report/${report_id}/messages`,
            'GET',
            undefined,
            abortController
        )
    );

export interface CreateReportMessage {
    body_md: string;
}

export const apiCreateReportMessage = async (report_id: number, body_md: string) =>
    transformTimeStampedDates(
        await apiCall<ReportMessage, CreateReportMessage>(`/api/report/${report_id}/messages`, 'POST', { body_md })
    );

export const apiReportSetState = async (report_id: number, stateAction: ReportStatus) =>
    await apiCall(`/api/report_status/${report_id}`, 'POST', {
        status: stateAction
    });

export const apiUpdateReportMessage = async (report_message_id: number, message: string) =>
    transformTimeStampedDates(
        await apiCall<ReportMessage>(`/api/report/message/${report_message_id}`, 'POST', {
            body_md: message
        })
    );

export const apiDeleteReportMessage = async (report_message_id: number) =>
    await apiCall(`/api/report/message/${report_message_id}`, 'DELETE', {});
