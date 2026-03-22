import type { ReportQueryFilter } from "../schema/query.ts";
import type {
	CreateReportMessage,
	CreateReportRequest,
	Report,
	ReportMessage,
	ReportStatusEnum,
	ReportWithAuthor,
} from "../schema/report.ts";
import { transformTimeStampedDates, transformTimeStampedDatesList } from "../util/time.ts";
import { apiCall } from "./common";

export const apiCreateReport = async (opts: CreateReportRequest, signal: AbortSignal) =>
	await apiCall<Report, CreateReportRequest>(signal, "/api/report", "POST", opts);

export const apiGetReport = async (report_id: number, signal: AbortSignal) =>
	await apiCall<ReportWithAuthor>(signal, `/api/report/${report_id}`);

export const apiGetReports = async (signal: AbortSignal, opts?: ReportQueryFilter) => {
	const resp = await apiCall<ReportWithAuthor[], ReportQueryFilter>(signal, `/api/reports`, "POST", opts);
	return resp.map(transformTimeStampedDates);
};

export const apiGetUserReports = async (signal: AbortSignal) => {
	const resp = await apiCall<ReportWithAuthor[], ReportQueryFilter>(signal, `/api/reports/user`);
	return resp.map(transformTimeStampedDates);
};

export const apiGetReportMessages = async (report_id: number, signal: AbortSignal) =>
	transformTimeStampedDatesList(
		await apiCall<ReportMessage[], CreateReportRequest>(signal, `/api/report/${report_id}/messages`),
	);

export const apiCreateReportMessage = async (report_id: number, body_md: string, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<ReportMessage, CreateReportMessage>(signal, `/api/report/${report_id}/messages`, "POST", {
			body_md,
		}),
	);

export const apiReportSetState = async (report_id: number, stateAction: ReportStatusEnum, signal: AbortSignal) =>
	await apiCall(signal, `/api/report_status/${report_id}`, "POST", {
		status: stateAction,
	});

export const apiUpdateReportMessage = async (report_message_id: number, message: string, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<ReportMessage>(signal, `/api/report/message/${report_message_id}`, "POST", {
			body_md: message,
		}),
	);

export const apiDeleteReportMessage = async (report_message_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/report/message/${report_message_id}`, "DELETE", {});
