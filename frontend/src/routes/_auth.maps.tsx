import { create } from "@bufbuild/protobuf";
import { useQuery } from "@connectrpc/connect-query";
import MapIcon from "@mui/icons-material/Map";
import PageviewIcon from "@mui/icons-material/Pageview";
import Box from "@mui/material/Box";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Select from "@mui/material/Select";
import Stack from "@mui/material/Stack";
import ToggleButton from "@mui/material/ToggleButton";
import ToggleButtonGroup from "@mui/material/ToggleButtonGroup";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import Grid from "@mui/system/Grid";
import { createFileRoute, redirect, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import z from "zod/v4";
import { CompactNumber } from "../component/CompactNumber.tsx";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { LoadingSpinner } from "../component/LoadingSpinner.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { MapStatsPrefixChart, MapStatsTopChart, MapStatsTrendChart } from "../component/stats/MapStatsCharts.tsx";
import {
	buildMapStatsChart,
	formatPlaytimeHours,
	type MapStatsMetric,
	type MapStatsRange,
	mapStatsMetricLabel,
	mapStatsMetricOptions,
	mapStatsRangeLabel,
	mapStatsRangeOptions,
	mapStatsWindow,
} from "../component/stats/mapStats.ts";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	dateTimeColumnSize,
	filterValueString,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderTableError } from "../error.tsx";
import { FilterSchema } from "../rpc/database/query/v1/filter_pb.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import {
	type MapStat,
	QueryMapStatsHistoryRequestSchema,
	QueryMapStatsRequestSchema,
} from "../rpc/stats/v1/stats_pb.ts";
import { buckets, queryMapStats, queryMapStatsHistory } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { humanCount, toTitleCase } from "../util/strings.ts";
import { dateToTimestamp, renderTimestamp, renderTimestampDateTime } from "../util/time.ts";

const defaultValues = { ...makeSchemaDefaults({ defaultColumn: "totalDurationMs" }), range: "90d", metric: "duration" };
const validateSearch = z
	.object({
		range: z.enum(["7d", "30d", "90d", "1y", "all"]).default("90d"),
		statsBucketId: z.number().optional(),
		metric: z.enum(["duration", "matches", "decided"]).default("duration"),
	})
	.extend(makeSchemaState("totalDurationMs").shape);

const columnHelper = createMRTColumnHelper<MapStat>();
const defaultOptions = createDefaultTableOptions<MapStat>();

export const Route = createFileRoute("/_auth/maps")({
	component: MapsComponent,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.statsEnabled);
		if (!context.auth?.hasPermission(Permission.STATS_READ)) {
			throw redirect({ to: "/permission" });
		}
	},
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: match.context.meta({ title: "Map Stats", description: "Per-map statistics." }),
	}),
});

// SummaryCard is a small stat box for the per-range summary strip.
const SummaryCard = ({ label, value }: { label: string; value: string }) => (
	<Grid size={{ xs: 6, sm: 4, md: 2 }}>
		<Paper variant="outlined" sx={{ padding: 1.5, height: "100%" }}>
			<Typography variant="overline" color="textSecondary" sx={{ letterSpacing: 1 }}>
				{label}
			</Typography>
			<Typography variant="h5" sx={{ fontWeight: 700, lineHeight: 1.3 }}>
				{value}
			</Typography>
		</Paper>
	</Grid>
);

function MapsComponent() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const { data: statBuckets, isLoading: isLoadingStatBuckets } = useQuery(buckets, {});

	// No explicit bucket in the URL means the default bucket (casual).
	const defaultBucketId = useMemo(() => {
		return statBuckets?.buckets.find((bucket) => bucket.isDefault)?.statsBucketId;
	}, [statBuckets]);
	const effectiveBucketId = search.statsBucketId ?? defaultBucketId;
	const bucketName = useMemo(() => {
		return statBuckets?.buckets.find((b) => b.statsBucketId === effectiveBucketId)?.bucketName ?? "All Groups";
	}, [effectiveBucketId, statBuckets]);

	const window = useMemo(() => mapStatsWindow(search.range), [search.range]);

	const qOpts = useMemo(() => {
		const sort = search.sorting?.find((sort) => sort);
		return create(QueryMapStatsRequestSchema, {
			statsBucketId: effectiveBucketId ?? 1,
			start: dateToTimestamp(window.start),
			end: dateToTimestamp(window.end),
			search: filterValueString("mapName", search.columnFilters) ?? "",
			filter: create(FilterSchema, {
				limit: String(search.pagination?.pageSize ?? 25),
				desc: sort ? sort.desc : true,
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: sort ? sort.id : "totalDurationMs",
			}),
		});
	}, [search, effectiveBucketId, window]);

	const { data, isLoading, isError, error, isRefetching } = useQuery(queryMapStats, qOpts, {
		enabled: !isLoadingStatBuckets,
		retry: false,
	});

	const historyOpts = useMemo(() => {
		return create(QueryMapStatsHistoryRequestSchema, {
			statsBucketId: effectiveBucketId ?? 1,
			start: dateToTimestamp(window.start),
			end: dateToTimestamp(window.end),
			timeBucket: window.group,
			topN: 6,
		});
	}, [effectiveBucketId, window]);

	const { data: historyData, isLoading: isLoadingHistory } = useQuery(queryMapStatsHistory, historyOpts, {
		enabled: !isLoadingStatBuckets,
		retry: false,
	});

	const chartData = useMemo(
		() => buildMapStatsChart(historyData?.series ?? [], search.metric, window.group, window.start, window.end),
		[historyData, search.metric, window],
	);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
					// Filters change the result set, so return to the first page.
					pagination: search.pagination ? { ...search.pagination, pageIndex: 0 } : undefined,
				},
			});
		},
		[search, navigate],
	);

	const setPagination: OnChangeFn<MRT_PaginationState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					pagination:
						typeof updater === "function"
							? updater(search.pagination ?? { pageIndex: 0, pageSize: 25 })
							: updater,
				},
			});
		},
		[search, navigate],
	);

	const setRange = useCallback(
		async (range: MapStatsRange) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					range,
					pagination: search.pagination ? { ...search.pagination, pageIndex: 0 } : undefined,
				},
			});
		},
		[search, navigate],
	);

	const setMetric = useCallback(
		async (metric: MapStatsMetric | "") => {
			if (!metric) {
				return;
			}
			await navigate({ to: Route.fullPath, search: { ...search, metric } });
		},
		[search, navigate],
	);

	const columns = useMemo(
		() => [
			columnHelper.accessor("mapName", {
				header: "Map",
				grow: true,
				Cell: ({ row, cell }) => (
					<TextLink
						to={"/matches"}
						search={setColumnFilter(search, "map.name", row.original.mapId)}
						sx={{ color: stringToColour(String(cell.getValue() ?? "")) }}
					>
						{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("matchesPlayed", {
				header: "Matches",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("matchesDecided", {
				header: "Decided",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("matchesDrawn", {
				header: "Drawn",
				size: 90,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("redWins", {
				header: "Red Wins",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("bluWins", {
				header: "Blu Wins",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("totalDurationMs", {
				header: "Playtime",
				size: 110,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <Typography>{formatPlaytimeHours(cell.getValue())}</Typography>,
			}),
			columnHelper.accessor("roundsPlayed", {
				header: "Rounds",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("stalemateRounds", {
				header: "Stalemates",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("uniquePlayers", {
				header: "Players",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("playerAppearances", {
				header: "Appearances",
				size: 110,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("kills", {
				header: "Kills",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("damage", {
				header: "Damage",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("healing", {
				header: "Healing",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("captures", {
				header: "Captures",
				size: 100,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("capturesBlocked", {
				header: "Blocked Caps",
				size: 110,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <CompactNumber value={cell.getValue()} />,
			}),
			columnHelper.accessor("firstPlayedAt", {
				header: "First Played",
				size: dateTimeColumnSize,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => <Typography>{renderTimestamp(cell.getValue())}</Typography>,
			}),
			columnHelper.accessor("lastPlayedAt", {
				header: "Last Played",
				size: dateTimeColumnSize,
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => (
					<Tooltip title={renderTimestampDateTime(cell.getValue())}>
						<Typography>{renderTimestamp(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),
		],
		[search],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.maps ?? [],
		rowCount: Number(data?.count ?? 0),
		state: {
			isLoading: isLoading || isRefetching,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			columnFilters: search.columnFilters,
			pagination: search.pagination,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				mapName: true,
				matchesPlayed: true,
				matchesDecided: true,
				matchesDrawn: true,
				totalDurationMs: true,
				roundsPlayed: true,
				uniquePlayers: true,
				lastPlayedAt: true,
				redWins: false,
				bluWins: false,
				stalemateRounds: false,
				playerAppearances: false,
				kills: false,
				damage: false,
				healing: false,
				captures: false,
				capturesBlocked: false,
				firstPlayedAt: false,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(1),
		muiToolbarAlertBannerProps: renderTableError(error),
		enableRowActions: true,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<Tooltip title={"View Matches"} key={1}>
					<IconButtonLink
						color={"error"}
						to={"/matches"}
						search={setColumnFilter(search, "map.name", row.original.mapId)}
					>
						<PageviewIcon />
					</IconButtonLink>
				</Tooltip>
			</RowActionContainer>
		),
	});

	if (isError) {
		return <ErrorDetails error={error} />;
	}

	const summary = data?.summary;
	const summaryLoading = isLoading || isLoadingHistory;

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Paper sx={{ padding: 2.5, overflow: "hidden" }}>
					<Stack spacing={1.5}>
						<Stack
							direction={{ xs: "column", sm: "row" }}
							spacing={1}
							sx={{ justifyContent: "space-between", alignItems: { sm: "center" } }}
						>
							<Box>
								<Typography variant="overline" color="textSecondary" sx={{ letterSpacing: 1.5 }}>
									Map Stats
								</Typography>
								<Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
									<MapIcon sx={{ fontSize: 40 }} />
									<Typography variant="h4" sx={{ fontWeight: 700, lineHeight: 1.1 }}>
										{mapStatsRangeLabel(search.range)} · {toTitleCase(bucketName)}
									</Typography>
								</Stack>
							</Box>
							<Stack
								direction="row"
								spacing={1.5}
								useFlexGap
								sx={{ flexWrap: "wrap", alignItems: "center" }}
							>
								<Select
									size="small"
									value={search.range}
									onChange={(e) => void setRange(e.target.value as MapStatsRange)}
									sx={{ minWidth: 140 }}
								>
									{mapStatsRangeOptions.map((option) => (
										<MenuItem key={option.value} value={option.value}>
											{option.label}
										</MenuItem>
									))}
								</Select>
								<Select
									size="small"
									value={effectiveBucketId ?? ""}
									onChange={(e) =>
										void navigate({
											to: Route.fullPath,
											search: { ...search, statsBucketId: Number(e.target.value) },
										})
									}
									sx={{ minWidth: 140 }}
								>
									{(statBuckets?.buckets ?? [])
										.filter((b) => b.isEnabled)
										.map((b) => (
											<MenuItem key={b.statsBucketId} value={b.statsBucketId}>
												{toTitleCase(b.bucketName)}
											</MenuItem>
										))}
								</Select>
								<ToggleButtonGroup
									exclusive
									size="small"
									value={search.metric}
									onChange={(_, metric) => void setMetric(metric)}
								>
									{mapStatsMetricOptions.map((option) => (
										<ToggleButton key={option.value} value={option.value}>
											{option.label}
										</ToggleButton>
									))}
								</ToggleButtonGroup>
							</Stack>
						</Stack>
						<Grid container spacing={1.5}>
							<SummaryCard
								label="Maps Played"
								value={summaryLoading ? "…" : humanCount(Number(summary?.mapsPlayed ?? 0))}
							/>
							<SummaryCard
								label="Matches"
								value={summaryLoading ? "…" : humanCount(Number(summary?.matchesPlayed ?? 0))}
							/>
							<SummaryCard
								label="Decided"
								value={summaryLoading ? "…" : humanCount(Number(summary?.matchesDecided ?? 0))}
							/>
							<SummaryCard
								label="Drawn"
								value={summaryLoading ? "…" : humanCount(Number(summary?.matchesDrawn ?? 0))}
							/>
							<SummaryCard
								label="Playtime"
								value={summaryLoading ? "…" : formatPlaytimeHours(summary?.totalDurationMs ?? "0")}
							/>
							<SummaryCard
								label="Rounds"
								value={summaryLoading ? "…" : humanCount(Number(summary?.roundsPlayed ?? 0))}
							/>
						</Grid>
					</Stack>
				</Paper>
			</Grid>
			<Grid size={{ xs: 12, lg: 8 }}>
				<Paper sx={{ padding: 2 }}>
					<Typography variant="h6" sx={{ marginBottom: 1 }}>
						{mapStatsMetricLabel(search.metric)} Over Time
					</Typography>
					{isLoadingHistory ? (
						<LoadingSpinner />
					) : (
						<MapStatsTrendChart data={chartData} metric={search.metric} />
					)}
				</Paper>
			</Grid>
			<Grid size={{ xs: 12, lg: 4 }}>
				<Paper sx={{ padding: 2 }}>
					<Typography variant="h6" sx={{ marginBottom: 1 }}>
						Top Maps · Playtime
					</Typography>
					{isLoading ? <LoadingSpinner /> : <MapStatsTopChart topMaps={data?.topMaps ?? []} />}
				</Paper>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<Paper sx={{ padding: 2 }}>
					<Typography variant="h6" sx={{ marginBottom: 1 }}>
						Maps by Type · {mapStatsMetricLabel(search.metric)}
					</Typography>
					{isLoading ? (
						<LoadingSpinner />
					) : (
						<MapStatsPrefixChart prefixes={data?.prefixes ?? []} metric={search.metric} />
					)}
				</Paper>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={`${mapStatsRangeLabel(search.range)} · ${toTitleCase(bucketName)}`}
					iconLeft={<MapIcon />}
				/>
			</Grid>
		</Grid>
	);
}
