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
import { renderDateTime } from "../util/time.ts";
import { reports } from "../rpc/ban/v1/report-ReportService_connectquery.ts";
import { useSuspenseQuery } from "@connectrpc/connect-query";
import { ReportStatus, type ReportWithAuthor } from "../rpc/ban/v1/report_pb.ts";
import { BanReason } from "../rpc/ban/v1/ban_pb.ts";

const columnHelper = createMRTColumnHelper<ReportWithAuthor>();
const defaultOptions = createDefaultTableOptions<ReportWithAuthor>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "report_id" });
const validateSearch = makeSchemaState("report_id");

export const Route = createFileRoute("/_mod/admin/reports")({
	component: AdminReports,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Reports" }, match.context.title("Reports")],
	}),
});

function AdminReports() {
	const navigate = useNavigate();
	const search = Route.useSearch();
	const theme = useTheme();
	const { data, isLoading, isError } = useSuspenseQuery(reports);

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
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("report.reportId", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => (
					<TextLink
						color={"primary"}
						to={`/report/$reportId`}
						params={{ reportId: String(cell.getValue()) }}
						marginRight={2}
					>
						#{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("report.reportStatus", {
				header: "Status",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(ReportStatus).map((status) => ({
					label: String(status),
					value: status,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.report?.reportStatus);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "report_status", [cell.getValue()])}>
						{ReportStatus[cell.getValue()]}
					</TextLink>
				),
			}),
			columnHelper.accessor("report.sourceId", {
				header: "Reporter",
				grow: true,
				enableColumnFilter: true,
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
						steam_id={row.original.author.steam_id}
						personaname={row.original.author.name}
						avatar_hash={row.original.author.avatarhash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "source_id", row.original.source_id)}
						>
							{row.original.author.name ?? row.original.author.steam_id}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("report.targetId", {
				header: "Subject",
				grow: true,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.target_id.toLowerCase();
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
						steam_id={row.original.subject.steam_id}
						personaname={row.original.subject.name}
						avatar_hash={row.original.subject?.avatarHash}
					>
						<RouterLink
							style={{
								color:
									theme.palette.mode === "dark"
										? theme.palette.primary.light
										: theme.palette.primary.dark,
							}}
							to={Route.fullPath}
							search={setColumnFilter(search, "target_id", row.original.targetId)}
						>
							{row.original.subject.name ?? row.original.subject?.steamId}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("report.reason", {
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				filterVariant: "multi-select",
				header: "Reason",
				grow: false,
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(BanReason.UNSPECIFIED) ||
						filterValue.includes(row.original.report?.reason)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "reason", [cell.getValue()])}>
						{BanReasons[cell.getValue() as BanReasonEnum]}
					</TextLink>
				),
			}),
			columnHelper.accessor("report.reasonText", {
				filterVariant: "text",
				grow: false,
				header: "Custom Reason",
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("report.createdOn", {
				header: "Created",
				grow: false,
				filterVariant: "date",
				Cell: ({ cell }) => (
					<Tooltip title={formatDistanceToNowStrict(cell.getValue(), { addSuffix: true })}>
						<Typography>{renderDateTime(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),
			columnHelper.accessor("report.updatedOn", {
				header: "Updated",
				grow: false,
				filterVariant: "date",
				Cell: ({ cell }) => (
					<Tooltip title={formatDistanceToNowStrict(cell.getValue(), { addSuffix: true })}>
						<Typography>{renderDateTime(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),
		];
	}, [search, theme]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data.reports,
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
		enableFilters: true,
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				reason_text: false,
				created_on: false,
				report_status: true,
				updated_on: true,
				report_id: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"User Reports"} />
			</Grid>
		</Grid>
	);
}
