import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import PageviewIcon from "@mui/icons-material/Pageview";
import { Tooltip, Typography } from "@mui/material";
import Grid from "@mui/material/Grid";
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
import { renderTimestamp } from "../util/time.ts";

const validateSearch = makeSchemaState("createdOn");
const columnHelper = createMRTColumnHelper<MatchOverview>();
const defaultOptions = createDefaultTableOptions<MatchOverview>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "createdOn" });

export const Route = createFileRoute("/_auth/matches/")({
	component: MatchesIndexPage,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: () => ({
		meta: [{ name: "description", content: "Player Match History" }],
	}),
});

function MatchesIndexPage() {
	const search = Route.useSearch();

	const navigate = useNavigate();
	const { data: mapResp, isLoading: isLoadingMaps } = useQuery(mapList);
	const { data: bucketList, isLoading: isLoadingBuckets } = useQuery(buckets);
	const { data: serverList, isLoading: isLoadingServers } = useQuery(servers);
	const { data, isLoading, isError, error } = useQuery(
		queryMatches,
		{
			filter: {
				limit: String(search.pagination?.pageSize ?? 25),
				desc: search.sorting?.find((sort) => sort)?.desc ?? true,
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: search.sorting?.find((sort) => sort)?.id ?? "createdOn",
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
		console.log(matchList);
		return matchList;
	}, [data]);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
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
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setPagination: OnChangeFn<MRT_PaginationState> = useCallback(
		(updater) => {
			navigate({
				to: Route.fullPath,
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
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						bucketList?.buckets.find((f) => f.statsBucketId === row.original.statsBucketId)
							?.statsBucketId === row.original.statsBucketId
					);
				},
				grow: false,
				header: "Bucket",
				Cell: ({ cell }) => (
					<Typography sx={{ color: stringToColour(cell.getValue() ?? "") }}>
						{toTitleCase(cell.getValue())}
					</Typography>
				),
			}),
			columnHelper.accessor("createdOn", {
				header: "Created On",
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
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.serverId);
				},
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
				// filterFn: (row, _, filterValue) => {
				// 	return filterValue.length === 0 || Boolean(mapSet.find((f) => f.mapId === row.original.map?.mapId));
				// },
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
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			sorting: search.sorting,
			pagination: search.pagination,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				sourceId: false,
				targetId: true,
				reason: true,
				evadeOk: false,
				deleted: false,
				validUntil: true,
				createdOn: true,
				active: false,
				reportId: true,
				cidr: false,
			},
		},
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
