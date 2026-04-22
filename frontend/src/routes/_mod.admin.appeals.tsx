import { useTheme } from "@mui/material";
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
import { useCallback, useMemo } from "react";
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
import { renderTimestamp } from "../util/time.ts";
import { appeals } from "../rpc/ban/v1/appeal-AppealService_connectquery.ts";
import { AppealState, BanReason } from "../rpc/ban/v1/ban_pb.ts";
import { useSuspenseQuery } from "@connectrpc/connect-query";
import type { AppealOverview } from "../rpc/ban/v1/appeal_pb.ts";
import { enumValues } from "../util/lists.ts";
import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";

const columnHelper = createMRTColumnHelper<AppealOverview>();
const defaultOptions = createDefaultTableOptions<AppealOverview>();
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
	const { data, isLoading, isError } = useSuspenseQuery(appeals);

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
			columnHelper.accessor("ban.banId", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => (
					<TextLink
						color={"primary"}
						to={`/ban/$banId`}
						params={{ banId: String(cell.getValue()) }}
						marginRight={2}
					>
						#{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("ban.appealState", {
				header: "Status",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: enumValues(AppealState).map((reason) => ({
					label: AppealState[reason],
					value: reason,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(AppealState.OPEN_UNSPECIFIED) ||
						filterValue.includes(row.original.ban?.appealState)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "appeal_state", [cell.getValue()])}>
						{AppealState[cell.getValue()]}
					</TextLink>
				),
			}),
			columnHelper.accessor("ban.sourceId", {
				header: "Author",
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.ban?.sourceId.toString();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.ban?.sourceId.toString().includes(query) || row.original.ban?.sourceId === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						steam_id={BigInt(row.original.ban?.sourceId ?? 0n)}
						personaname={row.original.sourcePersonaName}
						avatar_hash={row.original.sourceAvatarHash}
					>
						<RouterLink
							to={Route.fullPath}
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							search={setColumnFilter(search, "source_id", row.original.ban.sourceId)}
						>
							{row.original.sourcePersonaName ?? row.original.ban.sourceId}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("ban.targetId", {
				header: "Subject",
				enableColumnFilter: true,
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.targetPersonaName.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.ban.targetId.includes(query) || row.original.ban.targetId === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						steam_id={BigInt(row.original.ban?.targetId ?? 0n)}
						personaname={row.original.targetPersonaName}
						avatar_hash={row.original.targetAvatarHash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "target_id", row.original.ban?.targetId)}
						>
							{row.original.targetPersonaName ?? row.original.ban?.targetId}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("ban.reason", {
				filterVariant: "multi-select",
				header: "Reason",
				size: 150,
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReason[reason as BanReason].toString(),
					value: reason,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(BanReason.UNSPECIFIED) ||
						filterValue.includes(row.original.ban?.reason)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "reason", [cell.getValue()])}>
						{BanReason[cell.getValue() as BanReason]}
					</TextLink>
				),
			}),
			columnHelper.accessor("ban.reasonText", {
				header: "Custom",
				filterVariant: "text",
				grow: true,
			}),
			columnHelper.accessor("ban.createdOn", {
				header: "Created",
				filterVariant: "date",
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
			columnHelper.accessor("ban.updatedOn", {
				header: "Last Active",
				enableColumnFilter: false,
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
		],
		[search, theme],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.appeals ?? [],
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
