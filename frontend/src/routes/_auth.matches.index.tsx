import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import PageviewIcon from "@mui/icons-material/Pageview";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import prettyMilliseconds from "pretty-ms";
import { useCallback, useMemo } from "react";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	filterValueNumber,
	filterValueNumberArray,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { renderTableError } from "../error.tsx";
import { servers } from "../rpc/servers/v1/servers-ServersService_connectquery.ts";
import type { MatchOverview } from "../rpc/stats/v1/stats_pb.ts";
import { buckets, mapList, queryMatches } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { stringToColour } from "../util/colours.ts";
import { toTitleCase } from "../util/strings.ts";
import { dateToTimestamp, renderTimestamp } from "../util/time.ts";

const validateSearch = makeSchemaState("startTime");
const columnHelper = createMRTColumnHelper<MatchOverview>();
const defaultOptions = createDefaultTableOptions<MatchOverview>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "startTime" });

export const Route = createFileRoute("/_auth/matches/")({
	component: MatchesIndexPage,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: match.context.meta({ title: "Match History", description: "Recent match history." }),
	}),
});

function MatchesIndexPage() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const { data: mapResp, isLoading: isLoadingMaps } = useQuery(mapList);
	const { data: bucketList, isLoading: isLoadingBuckets } = useQuery(buckets);
	const { data: serverList, isLoading: isLoadingServers } = useQuery(servers);
	const startTimeRange = filterValue("startTime", search.columnFilters) as [unknown, unknown] | undefined;

	// No explicit bucket in the URL means the default bucket (casual). The
	// bucket filter always resolves to a concrete group so the filter dropdown
	// and the query agree on a single group.
	const defaultBucketId = useMemo(() => {
		return bucketList?.buckets.find((bucket) => bucket.isDefault)?.statsBucketId;
	}, [bucketList]);
	const effectiveColumnFilters = useMemo<MRT_ColumnFiltersState>(() => {
		const filters = search.columnFilters ?? [];
		if (defaultBucketId == null || filters.some((f) => f.id === "statsBucketName")) {
			return filters;
		}
		return [...filters, { id: "statsBucketName", value: defaultBucketId }];
	}, [search.columnFilters, defaultBucketId]);

	const { data, isLoading, isError, error } = useQuery(
		queryMatches,
		{
			statsBucketId: filterValueNumber("statsBucketName", effectiveColumnFilters),
			serverId: filterValueNumberArray("serverId", search.columnFilters),
			mapId: filterValueNumber("map.name", search.columnFilters),
			startTimeStart: dateToTimestamp(startTimeRange?.[0]),
			startTimeEnd: dateToTimestamp(startTimeRange?.[1]),
			filter: {
				limit: String(search.pagination?.pageSize ?? 25),
				desc: search.sorting?.find((sort) => sort)?.desc ?? true,
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: search.sorting?.find((sort) => sort)?.id ?? "startTime",
			},
		},
		{ enabled: !isLoadingServers && !isLoadingBuckets && !isLoadingMaps },
	);

	const mapSet = useMemo(() => {
		if (!mapResp?.maps) {
			return [];
		}

		return mapResp.maps.toSorted((a, b) => (a.name > b.name ? 1 : -1));
	}, [mapResp]);

	const realMatches = useMemo(() => {
		const matchList = data?.matches ?? [];
		return matchList;
	}, [data]);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
				to: "/matches",
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		(updater) => {
			navigate({
				to: "/matches",
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
		(updater) => {
			navigate({
				to: "/matches",
				search: {
					...search,
					pagination: search.pagination
						? typeof updater === "function"
							? updater(search.pagination)
							: updater
						: undefined,
				},
			});
		},
		[search, navigate],
	);
	const columns = useMemo(
		() => [
			columnHelper.accessor("statsBucketName", {
				filterVariant: "select",
				filterSelectOptions: bucketList?.buckets.map((bucket) => ({
					label: toTitleCase(bucket.bucketName),
					value: bucket.statsBucketId,
				})),
				grow: false,
				header: "Bucket",
				Cell: ({ row, cell }) => (
					<TextLink
						to={"/matches"}
						search={setColumnFilter(search, "statsBucketName", row.original.statsBucketId)}
						sx={{ color: stringToColour(cell.getValue() ?? "") }}
					>
						{toTitleCase(cell.getValue())}
					</TextLink>
				),
			}),
			columnHelper.accessor("startTime", {
				header: "Start Time",
				filterVariant: "datetime-range",
				grow: false,
				Cell: ({ cell }) => (
					<Tooltip
						title={formatDistanceToNowStrict(timestampDate(cell.getValue() as Timestamp), {
							addSuffix: true,
						})}
					>
						<Typography>{renderTimestamp(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),

			columnHelper.accessor("serverId", {
				header: "Server",
				grow: true,
				enableSorting: false,
				size: 250,
				filterVariant: "multi-select",
				filterSelectOptions: serverList?.servers.map((server) => ({
					label: server.serverNameLong,
					value: server.serverId,
				})),
				Cell: ({ row, cell }) => (
					<Tooltip title={row.original.serverNameShort}>
						<TextLink
							to={"/matches"}
							search={setColumnFilter(search, "serverId", [cell.getValue()])}
							sx={{ color: stringToColour(row.original.serverName ?? "") }}
						>
							{row.original.serverName}
						</TextLink>
					</Tooltip>
				),
			}),

			columnHelper.accessor("map.name", {
				enableColumnFilter: true,
				enableSorting: false,
				grow: true,
				header: "Map",
				filterVariant: "select",
				filterSelectOptions: mapSet.map((map) => ({
					label: map.name,
					value: map.mapId,
				})),
				Cell: ({ row, cell }) => (
					<TextLink
						to={"/matches"}
						search={setColumnFilter(search, "map.name", row.original.map?.mapId)}
						sx={{ color: stringToColour(String(cell.getValue() ?? "")) }}
					>
						{cell.getValue()}
					</TextLink>
				),
			}),

			columnHelper.accessor("duration", {
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				header: "Duration",
				Cell: ({ cell }) => {
					return <Typography>{prettyMilliseconds(Number(cell.getValue()))}</Typography>;
				},
			}),
		],
		[search, serverList, bucketList, mapSet],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: realMatches,
		rowCount: Number(data?.count ?? 0),
		enableFilters: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(1),
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: effectiveColumnFilters,
			sorting: search.sorting,
			pagination: search.pagination,
		},
		initialState: defaultOptions.initialState,
		muiToolbarAlertBannerProps: renderTableError(error),
		enableRowActions: true,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<Tooltip title={"View Match"} key={1}>
					<IconButtonLink color={"error"} to={"/match/$matchId"} params={{ matchId: row.original.matchId }}>
						<PageviewIcon />
					</IconButtonLink>
				</Tooltip>
			</RowActionContainer>
		),
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 12 }}>
				<SortableTable table={table} title={"Match History"} />
			</Grid>
		</Grid>
	);
}
