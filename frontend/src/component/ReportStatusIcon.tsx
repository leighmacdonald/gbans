import CancelPresentationIcon from "@mui/icons-material/CancelPresentation";
import GavelIcon from "@mui/icons-material/Gavel";
import NewReleasesIcon from "@mui/icons-material/NewReleases";
import QuizIcon from "@mui/icons-material/Quiz";
import Tooltip from "@mui/material/Tooltip";
import { ReportStatus } from "../rpc/ban/v1/report_pb.ts";

export const ReportStatusIcon = ({ reportStatus }: { reportStatus: ReportStatus }) => {
	switch (reportStatus) {
		case ReportStatus.NEED_MORE_INFO:
			return (
				<Tooltip title={"Needs more information"}>
					<QuizIcon color={"warning"} />
				</Tooltip>
			);
		case ReportStatus.CLOSED_WITHOUT_ACTION:
			return (
				<Tooltip title={"Report closed with no action"}>
					<CancelPresentationIcon color={"action"} />
				</Tooltip>
			);
		case ReportStatus.CLOSED_WITH_ACTION:
			return (
				<Tooltip title={"Report closed with action"}>
					<GavelIcon color={"error"} />
				</Tooltip>
			);
		default:
			return (
				<Tooltip title={"New report"}>
					<NewReleasesIcon color={"success"} />
				</Tooltip>
			);
	}
};
