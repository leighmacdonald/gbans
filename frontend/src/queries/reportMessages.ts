import { apiGetReportMessages } from "../api";

export const reportMessagesQueryOptions = (reportId: number) => ({
	queryKey: ["reportMessages", { reportID: reportId }],
	queryFn: async () => {
		const ac = new AbortController();
		return (await apiGetReportMessages(reportId, ac.signal)) ?? [];
	},
});
