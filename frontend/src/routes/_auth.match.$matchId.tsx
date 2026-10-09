import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import DownloadIcon from "@mui/icons-material/Download";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { useMemo } from "react";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { EventFeedTable } from "../component/stats/EventFeedTable.tsx";
import { MatchChatTable } from "../component/stats/MatchChatTable.tsx";
import { MatchEventMap } from "../component/stats/MatchEventMap.tsx";
import { assembleMatch } from "../component/stats/match.ts";
import { OverallTable } from "../component/stats/OverallTable.tsx";
import { RoundTable } from "../component/stats/RoundTable.tsx";
import { ScoreBanner } from "../component/stats/ScoreBanner.tsx";
import { TeamTotalsStrip } from "../component/stats/TeamTotalsStrip.tsx";
import { makeSchemaDefaults, makeSchemaState } from "../component/table/options.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { Team } from "../rpc/stats/v1/stats_pb.ts";
import { match } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { tf2Fonts } from "../theme.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { durationString, renderDateTime } from "../util/time.ts";

const validateSearch = makeSchemaState("points");
const defaultValues = makeSchemaDefaults({ defaultColumn: "points" });

export const Route = createFileRoute("/_auth/match/$matchId")({
	component: MatchPage,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(
			(context.appInfo.statsEnabled && context.auth?.hasPermission(Permission.BAN_WRITE)) ?? false,
		);
	},
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: () => ({
		meta: [{ name: "description", content: "Player Match History" }],
	}),
});

function MatchPage() {
	const { matchId } = Route.useParams();
	const { data, isLoading, isError, error } = useQuery(match, { matchId });

	const summary = useMemo(() => {
		if (!data?.match) {
			return undefined;
		}
		return assembleMatch(data.match);
	}, [data]);

	const winner = useMemo(() => {
		if (!data?.match?.overview) {
			return Team.UNASSIGNED_UNSPECIFIED;
		}
		return data.match.overview.scoreRed > data.match.overview.scoreBlu
			? Team.RED
			: data.match.overview.scoreRed < data.match.overview.scoreBlu
				? Team.BLU
				: Team.UNASSIGNED_UNSPECIFIED;
	}, [data]);

	if (isLoading) {
		return <LoadingPlaceholder />;
	}

	const overview = data?.match?.overview;

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Paper sx={{ padding: 2 }}>
					<Stack spacing={1}>
						<Typography variant="h5" sx={{ ...tf2Fonts }}>
							{overview?.serverName} BLU vs RED
						</Typography>
						<Stack direction="row" spacing={2} useFlexGap sx={{ flexWrap: "wrap" }}>
							<Typography variant="body1" sx={{ fontWeight: 700 }}>
								{overview?.map?.name}
							</Typography>
							<Typography variant="body2" color="textSecondary">
								{durationString(Number(overview?.duration ?? 0) * 1000)}
							</Typography>
							{overview?.createdOn && (
								<Typography variant="body2" color="textSecondary">
									{renderDateTime(timestampDate(overview.createdOn))}
								</Typography>
							)}
						</Stack>
						<Stack
							direction={{ xs: "column", sm: "row" }}
							spacing={1}
							sx={{ justifyContent: "space-between", alignItems: { sm: "center" } }}
						>
							<Typography variant="body2" color="textSecondary">
								{overview?.hostname}
							</Typography>
							{overview?.assetId && (
								<Button
									variant="contained"
									size="small"
									startIcon={<DownloadIcon />}
									href={`/asset/${overview.assetId}`}
								>
									Download STV
								</Button>
							)}
						</Stack>
					</Stack>
				</Paper>
			</Grid>

			<Grid size={{ xs: 12 }}>
				<ScoreBanner scoreBlu={overview?.scoreBlu ?? 0} scoreRed={overview?.scoreRed ?? 0} winner={winner} />
			</Grid>

			{summary && (
				<Grid size={{ xs: 12 }}>
					<TeamTotalsStrip totals={summary.teamTotals} />
				</Grid>
			)}

			<Grid size={{ xs: 12 }}>
				<OverallTable data={summary} matchId={matchId} isError={isError} error={error} isLoading={isLoading} />
			</Grid>

			<Grid size={{ xs: 12 }}>
				<RoundTable data={summary?.rounds ?? []} />
			</Grid>

			{summary && summary.chatFeed.length > 0 && (
				<Grid size={{ xs: 12 }}>
					<MatchChatTable chat={summary.chatFeed} players={summary.players} />
				</Grid>
			)}

			{summary && (
				<Grid size={{ xs: 12 }}>
					<MatchEventMap summary={summary} mapName={summary.info.mapName} />
				</Grid>
			)}

			{summary && summary.events.length > 0 && (
				<Grid size={{ xs: 12 }}>
					<EventFeedTable events={summary.events} players={summary.players} />
				</Grid>
			)}
		</Grid>
	);
}
