import { apiGetReportMessages } from "../api";

export const reportMessagesQueryOptions = (reportId: number) => ({
	queryKey: ["reportMessages", { reportID: reportId }],
	queryFn: async () => {
		return (await apiGetReportMessages(reportId)) ?? [];
	},
});
