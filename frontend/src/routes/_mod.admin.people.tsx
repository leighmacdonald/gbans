import { create } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	filterValueBool,
	filterValueNumber,
	filterValueString,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
	sortValueDefault,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { renderTableError } from "../error.tsx";
import { type Person, QueryRequestSchema, VisibilityState } from "../rpc/person/v1/person_pb.ts";
import { query } from "../rpc/person/v1/person-PersonService_connectquery.ts";

const defaultValues = makeSchemaDefaults({ defaultColumn: "createdOn" });
const validateSearch = makeSchemaState("createdOn");
const columnHelper = createMRTColumnHelper<Person>();
const defaultOptions = createDefaultTableOptions<Person>();

export const Route = createFileRoute("/_mod/admin/people")({
	component: AdminPeople,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "People" }, match.context.title("People")],
	}),
});

function AdminPeople() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();

	const opts = useMemo(() => {
		const sort = search.sorting ? sortValueDefault(search.sorting, "createdOn") : undefined;
		const steamId = filterValueString("steamId", search.columnFilters);

		const o = create(QueryRequestSchema, {
			filter: {
				desc: sort ? sort.desc : true,
				limit: String(search.pagination?.pageSize ?? 25n),
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination?.pageSize : 0n),
				orderBy: sort ? sort.id : "createdOn",
			},
		});

		const steamIds = steamId && steamId !== "" ? [steamId] : [];
		const vacBans = filterValueNumber("vacBans", search.columnFilters);
		const gameBans = filterValueNumber("gameBans", search.columnFilters);
		const communityBanned = filterValueBool("communityBanned", search.columnFilters);

		if (vacBans !== undefined) {
			o.vacBans = vacBans;
		}

		if (gameBans !== undefined) {
			o.gameBans = gameBans;
		}
		if (communityBanned !== undefined) {
			o.communityBanned = communityBanned;
		}

		if (steamIds.length > 0) {
			o.steamIds = steamIds;
		}

		return o;
	}, [search]);

	const { data, isLoading, isError, isRefetching, error } = useQuery(query, opts);

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

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("steamId", {
				header: "SteamID",
				grow: true,
				Cell: ({ row }) => {
					return (
						<PersonCell
							steamId={row.original.steamId}
							personaName={row.original.personaName}
							avatarHash={row.original.avatarHash}
						>
							<RouterLink
								style={{
									color:
										theme.palette.mode === "dark"
											? theme.palette.primary.light
											: theme.palette.primary.dark,
								}}
								to={Route.fullPath}
								search={setColumnFilter(search, "steamId", row.original.steamId)}
							>
								{row.original.personaName ?? row.original.steamId}
							</RouterLink>
						</PersonCell>
					);
				},
			}),
			columnHelper.accessor("visibilityState", {
				header: "Visibility",
				grow: false,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>
						{cell.getValue() === VisibilityState.PUBLIC ? "Public" : "Private"}
					</Typography>
				),
			}),
			columnHelper.accessor("vacBans", {
				header: "Vac Bans",
				grow: false,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>{cell.getValue() > 0 ? cell.getValue() : ""}</Typography>
				),
			}),
			columnHelper.accessor("communityBanned", {
				header: "Comm Ban",
				grow: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),

			columnHelper.accessor("timeCreated", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => {
					const value = cell.getValue();
					if (!value) {
						return;
					}

					return <TableCellRelativeDateField date={timestampDate(value)} />;
				},
			}),

			columnHelper.accessor("createdOn", {
				header: "First Seen",
				grow: false,
				Cell: ({ cell }) => {
					const value = cell.getValue();
					if (!value) {
						return;
					}

					return <TableCellRelativeDateField date={timestampDate(value)} />;
				},
			}),
		];
	}, [theme, search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.people : [],
		rowCount: Number(data ? data.count : 0),
		enableFilters: true,
		state: {
			columnFilters: search.columnFilters,
			isLoading: isLoading || isRefetching,
			pagination: search.pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				steamId: true,
				sourceId: true,
				body: true,
				createdOn: true,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		muiToolbarAlertBannerProps: renderTableError(error),
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Player Search"} />
			</Grid>
		</Grid>
	);
}
