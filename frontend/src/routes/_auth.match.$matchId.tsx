import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import { Box, Link, Paper, Stack, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/system";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, type MRT_SortingState, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { PersonCell } from "../component/PersonCell.tsx";
import {
	createDefaultTableOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderTableError } from "../error.tsx";
import blu_logo from "../icons/blu_logo.png";
import red_logo from "../icons/red_logo.png";
import type { PersonDisplay } from "../rpc/person/v1/person_core_pb.ts";
import { type Match, type MatchChatLog, Team } from "../rpc/stats/v1/stats_pb.ts";
import { match } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { blu, red } from "../theme.ts";
import { renderDateTime } from "../util/time.ts";
import { emptyOrNullString } from "../util/types.ts";

const validateSearch = makeSchemaState("points");
const columnHelper = createMRTColumnHelper<MatchRow>();
const defaultOptions = createDefaultTableOptions<MatchRow>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "points" });

export const Route = createFileRoute("/_auth/match/$matchId")({
	component: ProfileMatchesPage,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: () => ({
		meta: [{ name: "description", content: "Player Match History" }],
	}),
});

type MatchRow = {
	player: PersonDisplay;
	team: string;
	classes: string[];
	points: number;
	kills: number;
	assists: number;
	deaths: number;
	damage: number;
	kad: number;
	kd: number;
	dt: number;
	dtm: number;
	hp: number;
	as: number;
	bs: number;
	bsk: number;
	hs: number;
	hsk: number;
	wasHs: number;
	wasBs: number;
	cap: number;
	healing: number;
	drops: number;

	shots: number;
	hits: number;
};

type MatchInfo = {
	hostname: string;
	scoreRed: number;
	scoreBlu: number;
	duration: number;
	mapId: number;
	mapName: string;
	createdOn: Date;
};

type MatchView = {
	info: MatchInfo;
	summaries: MatchRow[];
	chat: MatchChatLog[];
};

const assembleMatch = (data: Match): MatchView => {
	if (!data.overview) {
		throw "invalid overview";
	}
	const summaries: Record<string, MatchRow> = {};
	//const players = data.players;
	//const rounds = [];

	for (let i = 0; i < data.rounds.length; i++) {
		for (let p = 0; p < data.rounds[i].players.length; p++) {
			const steamId = data?.rounds[i].players[p].person?.steamId ?? "";
			if (emptyOrNullString(steamId)) {
				continue;
			}
			if (!Object.hasOwn(summaries, steamId)) {
				const po = data.players[String(steamId)];
				if (!po) {
					continue;
				}
				summaries[steamId] = {
					bsk: 0,
					drops: 0,
					hits: 0,
					hsk: 0,
					points: 0,
					shots: 0,
					wasBs: 0,
					wasHs: 0,
					as: 0,
					bs: 0,
					assists: 0,
					cap: 0,
					classes: [],
					damage: 0,
					deaths: 0,
					dt: 0,
					dtm: 0,
					hp: 0,
					kad: 0,
					kd: 0,
					kills: 0,
					hs: 0,
					player: po,
					healing: 0,
					team: Team[Team.UNASSIGNED_UNSPECIFIED],
				};
			}
			const rp = data.rounds[i].players[p];
			const sm = summaries[steamId];

			sm.bsk += Number(rp.backstabKills);
			sm.drops += Number(rp.drops);
			sm.hits += Number(rp.hits);
			sm.hsk += Number(rp.headshotKills);
			sm.points += Number(rp.points);
			sm.healing += Number(rp.healing);
			sm.shots += Number(rp.shots);
			sm.wasBs += Number(rp.wasBackstabbed);
			sm.wasHs += Number(rp.wasHeadshot);
			sm.as += Number(rp.airshots);
			sm.hs += Number(rp.headshots);
			sm.bs += Number(rp.backstabs);
			sm.assists += Number(rp.assists);
			sm.cap += Number(rp.captures);
			sm.damage += Number(rp.scoreboardDamage);
			sm.deaths += Number(rp.scoreboardDeaths);
			sm.dt += Number(rp.damageTaken);
			sm.dtm += Number(rp.damageTaken);
			sm.kills += Number(rp.scoreboardKills);
			// summaries[data.rounds[i].players[p].steamId].name = data.rounds[i].players[p].;
			sm.team = Team[rp.team];
		}
	}

	const m: MatchView = {
		info: {
			createdOn: timestampDate(data.overview.createdOn as Timestamp),
			duration: Number(data.overview.duration),
			hostname: data.overview.hostname,
			mapName: data.overview.map?.name as string,
			mapId: data.overview.map?.mapId as number,
			scoreRed: data.overview.scoreRed,
			scoreBlu: data.overview.scoreBlu,
		},
		summaries: Object.values(summaries).toSorted((a, b) => b.points - a.points),
		chat: data.chatLogs,
	};
	return m;
};

const colSize = 100;

function ProfileMatchesPage() {
	const { matchId } = Route.useParams();
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();
	const { data, isLoading, isError, error } = useQuery(match, { matchId });

	const columns = useMemo(
		() => [
			columnHelper.accessor("player", {
				grow: false,
				header: "Player",
				sortingFn: (rowA, rowB) => {
					return rowA.original.player.name.toLocaleLowerCase() > rowB.original.player.name.toLocaleLowerCase()
						? -1
						: 1;
				},
				Cell: ({ cell }) => {
					const v = cell.getValue();
					return <PersonCell steamId={v.steamId} avatarHash={v.avatarHash} personaName={v.name} />;
				},
			}),
			columnHelper.accessor("points", {
				grow: false,
				header: "Pnt",
				sortDescFirst: true,
				size: colSize,
			}),
			columnHelper.accessor("kills", {
				grow: false,
				header: "K",
				sortDescFirst: true,
				size: colSize,
			}),
			columnHelper.accessor("assists", {
				grow: false,
				header: "A",
				sortDescFirst: true,
				size: colSize,
			}),
			columnHelper.accessor("deaths", {
				grow: false,
				header: "D",
				sortDescFirst: true,
				size: colSize,
			}),
			columnHelper.accessor("healing", {
				grow: false,
				header: "H",
				sortDescFirst: true,
				size: colSize,
			}),
			columnHelper.accessor("damage", {
				grow: false,
				header: "DA",
				size: colSize,
			}),
			columnHelper.accessor("dt", {
				grow: false,
				header: "DT",
				size: colSize,
			}),
			columnHelper.accessor("dtm", {
				grow: false,
				header: "DT/M",
				size: colSize,
			}),
			columnHelper.accessor("as", {
				grow: false,
				header: "AS",
				size: colSize,
			}),
			columnHelper.accessor("bs", {
				grow: false,
				header: "BS",
				size: colSize,
			}),
			columnHelper.accessor("cap", {
				grow: false,
				header: "CAP",
			}),
			columnHelper.accessor("shots", {
				grow: true,
				header: "S/H (%)",
				sortDescFirst: true,
				Cell: ({ row }) => {
					const hitPct = ((row.original.shots / row.original.hits) * 100).toFixed(2);
					return (
						<Typography>
							{row.original.shots}/{row.original.hits} ({hitPct})
						</Typography>
					);
				},
			}),
		],
		[],
	);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
				params: { matchId },
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate, matchId],
	);

	const summary = useMemo(() => {
		if (!data?.match) {
			return undefined;
		}
		const m = assembleMatch(data.match);
		// console.log(m);
		return m;
	}, [data]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: summary ? summary.summaries : [],
		enableFilters: false,
		enableFacetedValues: false,
		enableColumnActions: false,
		onSortingChange: setSorting,
		enablePagination: false,
		// displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading,
			showAlertBanner: isError,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				team: true,
				name: true,
			},
		},
		muiToolbarAlertBannerProps: renderTableError(error),
		enableRowActions: false,
		muiTableBodyRowProps: ({ row }) => ({
			style: {
				backgroundColor: `${row.original.team === "red" ? "#012344" : "#f33333"} !important`, // Conditional color
			},
		}),
	});

	const winner = useMemo(() => {
		if (!data?.match?.overview) {
			return "";
		}
		return data.match.overview.scoreRed > data.match.overview.scoreBlu
			? "red"
			: data.match.overview.scoreRed < data.match.overview.scoreBlu
				? "blu"
				: "";
	}, [data]);

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
							{" "}
							{formatDuration(Number(data?.match?.overview?.duration ?? 0))}
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
				<Stack direction="row" padding={2}>
					<REDCard score={data?.match?.overview?.scoreRed ?? 0} winner={winner === "red"} />
					<BLUCard score={data?.match?.overview?.scoreBlu ?? 0} winner={winner === "blu"} />
				</Stack>
				<SortableTable table={table} title={"Overall Stats"} hidePagination={true} />
			</Grid>
		</Grid>
	);
}

const REDCard = ({ score, winner }: { score: number; winner: boolean }) => {
	return (
		<Box
			sx={{
				backgroundColor: red,
				background: `url(${red_logo})`,
				backgroundRepeat: "no-repeat",
				// backgroundSize: "cover",
				backgroundPosition: "left",
			}}
			flex={1}
			height={65}
			textAlign={"right"}
			paddingRight={2}
			paddingTop={1}
		>
			{/*<BoxImg src={red_logo} />*/}
			<Typography variant="h1" fontFamily={"TF2 Build"} color={winner ? "success" : "error"}>
				{score}
			</Typography>
		</Box>
	);
};

const BLUCard = ({ score, winner }: { score: number; winner: boolean }) => {
	return (
		<Box
			sx={{
				backgroundColor: blu,
				background: `url(${blu_logo})`,
				backgroundRepeat: "no-repeat",
				// backgroundSize: "cover",
				backgroundPosition: "right",
			}}
			flex={1}
			height={65}
			paddingLeft={2}
			paddingTop={1}
		>
			<Typography variant="h1" fontFamily={"TF2 Build"} color={winner ? "success" : "error"}>
				{score}
			</Typography>
		</Box>
	);
};

function formatDuration(ms: number): string {
	const seconds = Math.floor(ms / 1000);
	const minutes = Math.floor(seconds / 60);
	const hours = Math.floor(minutes / 60);

	const hrs = hours > 0 ? `${hours}h ` : "";
	const mins = minutes % 60 > 0 ? `${minutes % 60}m ` : "";
	const secs = seconds % 60 > 0 || ms === 0 ? `${seconds % 60}s` : "";

	return `${hrs}${mins}${secs}`.trim();
}
