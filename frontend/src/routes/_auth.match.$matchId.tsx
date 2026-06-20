import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import Grid from "@mui/material/Grid";
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
import type { PersonDisplay } from "../rpc/person/v1/person_core_pb.ts";
import { type Match, type MatchChatLog, Team } from "../rpc/stats/v1/stats_pb.ts";
import { match } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { emptyOrNullString } from "../util/types.ts";

const validateSearch = makeSchemaState("matchId");
const columnHelper = createMRTColumnHelper<MatchRow>();
const defaultOptions = createDefaultTableOptions<MatchRow>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "createdOn" });

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
	hs: number;
	cap: number;
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
	// const rounds = [];

	for (let i = 0; i < data.rounds.length; i++) {
		for (let p = 0; p < data.rounds[i].players.length; p++) {
			const steamId = data?.rounds[i].players[p].person?.steamId ?? "";
			if (emptyOrNullString(steamId)) {
				continue;
			}
			if (!Object.hasOwn(summaries, steamId)) {
				summaries[steamId] = {
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
					player: data.players[steamId],
					team: Team[Team.UNASSIGNED_UNSPECIFIED],
				};
			}
			const rp = data.rounds[i].players[p];
			const sm = summaries[steamId];
			sm.as += Number(rp.airshots);
			sm.hs += Number(rp.headshots);
			sm.bs += Number(rp.backstabs);
			sm.assists += Number(rp.assists);
			sm.cap += Number(rp.captures);
			sm.damage += Number(rp.damage);
			sm.deaths += Number(rp.deaths);
			sm.dt += Number(rp.damageTaken);
			sm.dtm += Number(rp.damageTaken);
			sm.kills += Number(rp.kills);
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
		summaries: Object.values(summaries).toSorted((a, b) => b.kills - a.kills),
		chat: data.chatLogs,
	};
	return m;
};

const colSize = 100;

function ProfileMatchesPage() {
	const { matchId } = Route.useParams();
	const search = Route.useSearch();
	const navigate = useNavigate();
	const { data, isLoading, isError, error } = useQuery(match, { matchId });

	const columns = useMemo(
		() => [
			columnHelper.accessor("player", {
				grow: false,
				header: "Player",
				Cell: ({ row }) => (
					<PersonCell
						steamId={row.original.player.steamId}
						avatarHash={row.original.player.avatarHash}
						personaName={row.original.player.name}
					/>
				),
			}),
			columnHelper.accessor("kills", {
				grow: false,
				header: "K",
				size: colSize,
			}),
			columnHelper.accessor("assists", {
				grow: false,
				header: "A",
				size: colSize,
			}),
			columnHelper.accessor("deaths", {
				grow: false,
				header: "D",
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
		console.log(m);
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
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 12 }}>
				<SortableTable table={table} title={"Match History"} hidePagination={true} />
			</Grid>
		</Grid>
	);
}
