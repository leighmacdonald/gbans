import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiGetAppeals, appealStateString } from "../api";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { TextLink } from "../component/TextLink.tsx";
import {
	createDefaultTableOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { AppealState, BanReason, BanReasons, type BanRecord } from "../schema/bans.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<BanRecord>();
const defaultOptions = createDefaultTableOptions<BanRecord>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "ban_id" });
const validateSearch = makeSchemaState("ban_id");

export const Route = createFileRoute("/_mod/admin/appeals")({
	component: AdminAppeals,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Appeals" }, match.context.title("Appeals")],
	}),
});

function AdminAppeals() {
	const navigate = useNavigate();
	const theme = useTheme();
	const search = Route.useSearch();
	const { data, isLoading, isError } = useQuery({
		queryKey: ["appeals"],
		queryFn: async ({ signal }) => {
			return (await apiGetAppeals({}, signal)) ?? [];
		},
	});

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
			columnHelper.accessor("ban_id", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => (
					<TextLink
						color={"primary"}
						to={`/ban/$ban_id`}
						params={{ ban_id: String(cell.getValue()) }}
						marginRight={2}
					>
						#{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("appeal_state", {
				header: "Status",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(AppealState).map((reason) => ({
					label: appealStateString(reason),
					value: reason,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(AppealState.Any) ||
						filterValue.includes(row.original.appeal_state)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "appeal_state", [cell.getValue()])}>
						{appealStateString(cell.getValue())}
					</TextLink>
				),
			}),
			columnHelper.accessor("source_id", {
				header: "Author",
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.source_id.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.source_id.includes(query) || row.original.source_id === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.source_id}
						personaname={row.original.source_personaname}
						avatar_hash={row.original.source_avatarhash}
					>
						<RouterLink
							to={Route.fullPath}
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							search={setColumnFilter(search, "source_id", row.original.source_id)}
						>
							{row.original.source_personaname ?? row.original.source_id}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				enableColumnFilter: true,
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.target_personaname.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.target_id.includes(query) || row.original.target_id === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						steam_id={row.original.target_id}
						personaname={row.original.target_personaname}
						avatar_hash={row.original.target_avatarhash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "target_id", row.original.target_id)}
						>
							{row.original.target_personaname ?? row.original.target_id}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("reason", {
				filterVariant: "multi-select",
				header: "Reason",
				size: 150,
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(BanReason.Any) ||
						filterValue.includes(row.original.reason)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "reason", [cell.getValue()])}>
						{BanReasons[cell.getValue()]}
					</TextLink>
				),
			}),
			columnHelper.accessor("reason_text", {
				header: "Custom",
				filterVariant: "text",
				grow: true,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				filterVariant: "date",
				Cell: ({ cell }) => (
					<Tooltip title={formatDistanceToNowStrict(cell.getValue(), { addSuffix: true })}>
						<Typography>{renderDateTime(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),
			columnHelper.accessor("updated_on", {
				header: "Last Active",
				enableColumnFilter: false,
				Cell: ({ cell }) => (
					<Tooltip title={formatDistanceToNowStrict(cell.getValue(), { addSuffix: true })}>
						<Typography>{renderDateTime(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),
		],
		[search, theme],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			sorting: search.sorting,
			pagination: search.pagination,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				reason_text: true,
				created_on: false,
				updated_on: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Ban Appeals"} />
			</Grid>
		</Grid>
	);
}
