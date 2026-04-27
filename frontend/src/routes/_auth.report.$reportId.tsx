import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import AccountBalanceIcon from "@mui/icons-material/AccountBalance";
import GavelIcon from "@mui/icons-material/Gavel";
import InfoIcon from "@mui/icons-material/Info";
import SendIcon from "@mui/icons-material/Send";
import VolumeOffIcon from "@mui/icons-material/VolumeOff";
import Avatar from "@mui/material/Avatar";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemAvatar from "@mui/material/ListItemAvatar";
import ListItemText from "@mui/material/ListItemText";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMemo } from "react";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ProfileInfoBox } from "../component/ProfileInfoBox.tsx";
import { ReportModPanel } from "../component/ReportModPanel.tsx";
import { ReportViewComponent } from "../component/ReportViewComponent.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { SteamIDList } from "../component/SteamIDList.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { AppealState, BanReason, BanType } from "../rpc/ban/v1/ban_pb.ts";
import { get } from "../rpc/ban/v1/ban-BanService_connectquery.ts";
import { ReportStatus, type ReportWithAuthorValid } from "../rpc/ban/v1/report_pb.ts";
import { report } from "../rpc/ban/v1/report-ReportService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { avatarHashToURL, reportStatusColour } from "../util/strings.ts";
import { renderTimeDistance, renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_auth/report/$reportId")({
	component: ReportView,
	head: ({ match }) => ({
		meta: [
			{ name: "description", content: "View a report" },
			match.context.title(`Report #${match.params.reportId}`),
		],
	}),
});

function ReportView() {
	const { reportId } = Route.useParams();
	const { appInfo } = Route.useRouteContext();
	const { hasPermission } = useAuth();
	const theme = useTheme();
	const navigate = useNavigate();

	const { data: reportResp, isLoading } = useQuery(report, { reportId: Number(reportId) });

	const { data: ban, isLoading: isLoadingBan } = useQuery(
		get,
		{},
		{ enabled: Boolean(reportResp?.report?.subject?.steamId) },
	);

	const renderBan = useMemo(() => {
		if (isLoading || isLoadingBan || !ban?.ban || ban.ban?.banId === 0) {
			return;
		}

		return (
			<ContainerWithHeader
				title={ban.ban.banType === BanType.BANNED ? "Banned" : "Muted"}
				iconLeft={ban.ban.banType === BanType.BANNED ? <GavelIcon /> : <VolumeOffIcon />}
			>
				<List dense={true}>
					<ListItem>
						<ListItemText primary={"Reason"} secondary={BanReason[ban.ban.reason]} />
					</ListItem>
					{ban.ban.reasonText !== "" && (
						<ListItem>
							<ListItemText primary={"Custom Reason"} secondary={ban.ban.note} />
						</ListItem>
					)}
					<ListItem>
						<ListItemText primary={"Ban ID"} secondary={ban.ban.banId} />
					</ListItem>
					<ListItem>
						<ListItemText primary={"Note"} secondary={ban.ban.note} />
					</ListItem>
					<ListItem>
						<ListItemText primary={"Evasion OK"} secondary={ban.ban.evadeOk ? "Yes" : "No"} />
					</ListItem>
					<ListItem>
						<ListItemText primary={"Appeal State"} secondary={AppealState[ban.ban.appealState]} />
					</ListItem>
					<ListItem>
						<ListItemText primary={"Creation Date"} secondary={renderTimestamp(ban.ban.createdOn)} />
					</ListItem>
					<ListItem>
						<ListItemText primary={"Valid Until Date"} secondary={renderTimestamp(ban.ban.validUntil)} />
					</ListItem>
					<ListItem>
						<ListItemText
							primary={"Expires"}
							secondary={renderTimeDistance(timestampDate(ban.ban.validUntil as Timestamp), new Date())}
						/>
					</ListItem>
					<ListItem>
						<ListItemText
							primary={"Author"}
							secondary={
								<Link component={RouterLink} to={`/profile/${ban.ban.sourceId}`}>
									{ban.ban.sourcePersonaName}
								</Link>
							}
						/>
					</ListItem>
				</List>
			</ContainerWithHeader>
		);
	}, [ban, isLoadingBan, isLoading]);

	const reportStatusView = useMemo(() => {
		return (
			<ContainerWithHeader title={"Report Status"} iconLeft={<AccountBalanceIcon />}>
				<Typography
					padding={2}
					variant={"h4"}
					align={"center"}
					sx={{
						color: "#111111",
						backgroundColor: reportStatusColour(
							reportResp?.report?.report?.reportStatus ?? ReportStatus.OPENED_UNSPECIFIED,
							theme,
						),
					}}
				>
					{ReportStatus[reportResp?.report?.report?.reportStatus ?? ReportStatus.OPENED_UNSPECIFIED]}
				</Typography>
			</ContainerWithHeader>
		);
	}, [reportResp?.report?.report?.reportStatus, theme]);

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 8 }}>
				{reportResp?.report && (
					<ReportViewComponent
						report={reportResp?.report as ReportWithAuthorValid}
						assetURL={appInfo.assetUrl}
					/>
				)}
			</Grid>
			<Grid size={{ xs: 12, md: 4 }}>
				<div>
					<Grid container spacing={2}>
						<Grid size={{ xs: 6, md: 12 }}>
							{reportResp?.report?.report?.targetId && (
								<ProfileInfoBox steamId={reportResp?.report?.report?.targetId} />
							)}
						</Grid>
						{renderBan && <Grid size={{ xs: 6, md: 12 }}>{renderBan}</Grid>}
						<Grid size={{ xs: 6, md: 12 }}>
							<SteamIDList steamId={reportResp?.report?.report?.sourceId ?? ""} />
						</Grid>
						<Grid size={{ xs: 6, md: 12 }}>{reportStatusView}</Grid>
						<Grid size={{ xs: 6, md: 12 }}>
							<ContainerWithHeader title={"Report Details"} iconLeft={<InfoIcon />}>
								<List sx={{ width: "100%" }}>
									<ListItem
										sx={{
											"&:hover": {
												cursor: "pointer",
												backgroundColor: theme.palette.background.paper,
											},
										}}
										onClick={async () => {
											await navigate({
												to: `/profile/${reportResp?.report?.report?.sourceId}`,
											});
										}}
									>
										<ListItemAvatar>
											<Avatar src={avatarHashToURL(reportResp?.report?.author?.avatarHash)}>
												<SendIcon />
											</Avatar>
										</ListItemAvatar>
										<ListItemText primary={reportResp?.report?.author?.name} secondary={"Author"} />
									</ListItem>
									{reportResp?.report?.report?.reason && (
										<ListItem
											sx={{
												"&:hover": {
													cursor: "pointer",
													backgroundColor: theme.palette.background.paper,
												},
											}}
										>
											<ListItemText
												primary={"Reason"}
												secondary={BanReason[reportResp?.report?.report?.reason]}
											/>
										</ListItem>
									)}
									{reportResp?.report?.report?.reason &&
										reportResp?.report?.report?.reasonText !== "" && (
											<ListItem
												sx={{
													"&:hover": {
														cursor: "pointer",
														backgroundColor: theme.palette.background.paper,
													},
												}}
											>
												<ListItemText
													primary={"Custom Reason"}
													secondary={reportResp?.report?.report?.reasonText}
												/>
											</ListItem>
										)}
								</List>
							</ContainerWithHeader>
						</Grid>
						{hasPermission(Privilege.MODERATOR) && (
							<Grid size={{ xs: 6, md: 12 }}>
								<ReportModPanel reportId={Number(reportId)} />
							</Grid>
						)}
					</Grid>
				</div>
			</Grid>
		</Grid>
	);
}
