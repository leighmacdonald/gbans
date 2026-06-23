import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { Link, Paper, Stack, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/system";
import { createFileRoute, stripSearchParams } from "@tanstack/react-router";
import { useMemo } from "react";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { BLUCard } from "../component/stats/BLUCard.tsx";
import { assembleMatch } from "../component/stats/match.ts";
import { OverallTable } from "../component/stats/OverallTable.tsx";
import { REDCard } from "../component/stats/REDCard.tsx";
import { RoundTable } from "../component/stats/RoundTable.tsx";
import { makeSchemaDefaults, makeSchemaState } from "../component/table/options.ts";
import { Team } from "../rpc/stats/v1/stats_pb.ts";
import { match } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { durationString, renderDateTime } from "../util/time.ts";

const validateSearch = makeSchemaState("points");
const defaultValues = makeSchemaDefaults({ defaultColumn: "points" });

export const Route = createFileRoute("/_auth/match/$matchId")({
	component: MatchPage,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: () => ({
		meta: [{ name: "description", content: "Player Match History" }],
	}),
});

function MatchPage() {
	const theme = useTheme();
	const { matchId } = Route.useParams();
	const { data, isLoading, isError, error } = useQuery(match, { matchId });

	const summary = useMemo(() => {
		if (!data?.match) {
			return undefined;
		}
		const m = assembleMatch(data.match);
		// console.log(m);
		return m;
	}, [data]);

	const winner = useMemo(() => {
		if (!data?.match?.overview) {
			return "";
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

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 12 }} component={Paper} padding={1}>
				<Grid container component={Paper} sx={{ backgroundColor: theme.palette.primary.main }} padding={1}>
					<Grid size={{ md: 6, xs: 12 }}>
						<Typography variant={"subtitle1"}> {data?.match?.overview?.serverName}</Typography>
					</Grid>
					<Grid size={{ md: 6, xs: 12 }}>
						<Typography variant={"subtitle1"} textAlign={"right"}>
							{data?.match?.overview?.hostname}
						</Typography>
					</Grid>
					<Grid size={{ md: 6, xs: 12 }}>
						<Typography variant={"subtitle1"}> {data?.match?.overview?.map?.name}</Typography>
					</Grid>
					{data?.match?.overview?.createdOn && (
						<Grid size={{ md: 6, xs: 12 }}>
							<Typography textAlign={"right"} variant={"subtitle1"}>
								{renderDateTime(timestampDate(data.match.overview.createdOn))}
							</Typography>
						</Grid>
					)}
					<Grid size={{ md: 6, xs: 12 }}>
						<Typography variant={"subtitle1"}>
							{durationString(Number(data?.match?.overview?.duration ?? 0) * 1000)}
						</Typography>
					</Grid>

					<Grid size={{ md: 6, xs: 12 }}>
						<Typography textAlign={"right"}>
							<Link color="textPrimary" href={`/asset/${data?.match?.overview?.assetId}`}>
								Download STV
							</Link>
						</Typography>
					</Grid>
				</Grid>

				<Stack direction="row" padding={4}>
					<REDCard score={data?.match?.overview?.scoreRed ?? 0} winner={winner === Team.RED} />
					<BLUCard score={data?.match?.overview?.scoreBlu ?? 0} winner={winner === Team.BLU} />
				</Stack>

				<Stack spacing={4}>
					<OverallTable
						data={summary}
						matchId={matchId}
						isError={isError}
						error={error}
						isLoading={isLoading}
					/>
					<RoundTable data={summary?.rounds ?? []} />
				</Stack>
			</Grid>
		</Grid>
	);
}
