import { ReportQueryFilter } from '../schema/query.ts';
import {
    CreateReportMessage,
    CreateReportRequest,
    Report,
    ReportMessage,
    ReportStatusEnum,
    ReportWithAuthor
} from '../schema/report.ts';
import { transformTimeStampedDates, transformTimeStampedDatesList } from '../util/time.ts';
import { apiCall } from './common';

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

export const apiCreateReportMessage = async (report_id: number, body_md: string) =>
    transformTimeStampedDates(
        await apiCall<ReportMessage, CreateReportMessage>(`/api/report/${report_id}/messages`, 'POST', { body_md })
    );

export const apiReportSetState = async (report_id: number, stateAction: ReportStatusEnum) =>
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
