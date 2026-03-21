/** biome-ignore-all lint/correctness/noChildrenProp: form */
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import UndoIcon from "@mui/icons-material/Undo";
import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiGetBans } from "../api";
import { BanModal } from "../component/modal/BanModal.tsx";
import { UnbanModal } from "../component/modal/UnbanModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { BanReason, type BanReasonEnum, BanReasons, type BanRecord } from "../schema/bans.ts";
import { isPermanentBan } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<BanRecord>();
const defaultOptions = createDefaultTableOptions<BanRecord>();
const validateSearch = makeSchemaState("ban_id");

export const Route = createFileRoute("/_mod/admin/bans")({
	component: AdminBans,
	validateSearch,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Bans" }, match.context.title("Bans")],
	}),
});

function AdminBans() {
	const queryClient = useQueryClient();
	const search = Route.useSearch();
	const theme = useTheme();
	const { sendFlash } = useUserFlashCtx();
	const navigate = useNavigate();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["bans"],
		queryFn: async () => {
			return (await apiGetBans({ deleted: true })) ?? [];
		},
	});

	const onNewBanSteam = useCallback(async () => {
		try {
			const ban = (await NiceModal.show(BanModal, {})) as BanRecord;
			queryClient.setQueryData(["bans"], [...(data ?? []), ban]);
		} catch (e) {
			sendFlash("error", `Error trying to set up ban: ${e}`);
		}
	}, [queryClient, sendFlash, data]);

	const onUnban = useCallback(
		async (ban: BanRecord) => {
			try {
				await NiceModal.show(UnbanModal, {
					banId: ban.ban_id,
					personaName: ban.target_personaname,
				});
				queryClient.setQueryData(
					["bans"],
					(data ?? []).filter((b) => b.ban_id !== ban.ban_id),
				);
				sendFlash("success", "Unbanned player successfully");
			} catch (e) {
				sendFlash("error", `Error trying to unban: ${e}`);
			}
		},
		[queryClient, sendFlash, data],
	);

	const onEdit = useCallback(
		async (ban: BanRecord) => {
			try {
				const updated = (await NiceModal.show(BanModal, {
					banId: ban.ban_id,
					personaName: ban.target_personaname,
					existing: ban,
				})) as BanRecord;
				queryClient.setQueryData(
					["bans"],
					(data ?? []).map((b) => (b.ban_id === updated.ban_id ? updated : b)),
				);
			} catch (e) {
				sendFlash("error", `Error trying to edit ban: ${e}`);
			}
		},
		[queryClient, sendFlash, data],
	);

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
				grow: false,
				header: "Ban ID",
				Cell: ({ cell }) => (
					<TextLink to={`/ban/$ban_id`} params={{ ban_id: String(cell.getValue()) }}>
						{`#${cell.getValue()}`}
					</TextLink>
				),
			}),
			columnHelper.accessor("source_id", {
				header: "Author",
				enableSorting: false,
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.source_personaname.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.source_id.includes(query) || row.original.source_id === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => {
					return (
						<PersonCell
							steam_id={row.original.source_id}
							personaname={row.original.source_personaname}
							avatar_hash={row.original.source_avatarhash}
						>
							<RouterLink
								style={{ color: theme.palette.primary.light }}
								to={Route.fullPath}
								search={setColumnFilter(search, "source_id", row.original.source_id)}
							>
								{row.original.source_personaname ?? row.original.source_id}
							</RouterLink>
						</PersonCell>
					);
				},
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				grow: true,
				enableSorting: false,
				enableColumnFilter: true,
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
							style={{ color: theme.palette.primary.light }}
							to={Route.fullPath}
							search={setColumnFilter(search, "target_id", row.original.target_id)}
						>
							{row.original.target_personaname ?? row.original.target_id}
						</RouterLink>
					</PersonCell>
				),
			}),
			columnHelper.accessor("cidr", {
				enableColumnFilter: true,
				grow: false,
				filterVariant: "text",
				header: "CIDR/IP",
			}),
			columnHelper.accessor("reason", {
				enableColumnFilter: true,
				enableSorting: false,
				grow: false,
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				filterVariant: "multi-select",
				header: "Reason",
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(BanReason.Any) ||
						filterValue.includes(row.original.reason)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "reason", [cell.getValue()])}>
						{BanReasons[cell.getValue() as BanReasonEnum]}
					</TextLink>
				),
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				filterVariant: "date-range",
				grow: false,
				Cell: ({ cell }) => (
					<Tooltip title={formatDistanceToNowStrict(cell.getValue(), { addSuffix: true })}>
						<Typography>{renderDateTime(cell.getValue())}</Typography>
					</Tooltip>
				),
			}),
			columnHelper.accessor("valid_until", {
				header: "Duration",
				enableColumnFilter: false,
				grow: false,
				filterVariant: "date-range",
				Cell: ({ row }) => {
					return typeof row.original === "undefined" ? (
						""
					) : isPermanentBan(row.original.created_on, row.original.valid_until) ? (
						"Permanent"
					) : (
						<TableCellRelativeDateField
							date={row.original.created_on}
							compareDate={row.original.valid_until}
						/>
					);
				},
			}),
			columnHelper.accessor("evade_ok", {
				meta: {
					tooltip: "Evasion OK. Players connecting from the same ip will not be banned.",
				},
				enableColumnFilter: false,
				grow: false,
				filterVariant: "checkbox",
				header: "Evade",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("deleted", {
				enableColumnFilter: false,
				grow: false,
				filterVariant: "checkbox",
				meta: { tooltip: "Deleted / Expired Bans" },
				header: "Expired",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("report_id", {
				header: "Report",
				grow: false,
				meta: { tooltip: "Linked report" },
				Cell: ({ cell }) =>
					Boolean(cell.getValue()) && (
						<TextLink to={`/report/$reportId`} params={{ reportId: String(cell.getValue()) }}>
							{`#${cell.getValue()}`}
						</TextLink>
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
		enableFacetedValues: true,
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
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				evade_ok: false,
				deleted: false,
				valid_until: true,
				created_on: true,
				active: false,
				report_id: true,
				cidr: false,
			},
		},

		enableRowActions: true,
		renderRowActionMenuItems: ({ row }) => [
			<IconButton
				key={"edit"}
				color={"warning"}
				onClick={async () => {
					await onEdit(row.original);
				}}
			>
				<Tooltip title={"Edit Ban"}>
					<EditIcon />
				</Tooltip>
			</IconButton>,
			<IconButton
				key={"remove"}
				color={"success"}
				onClick={async () => {
					await onUnban(row.original);
				}}
			>
				<Tooltip title={"Remove Ban"}>
					<UndoIcon />
				</Tooltip>
			</IconButton>,
		],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"Bans"}
					buttons={[
						<Tooltip title="Create new ban" key="create-new-ban">
							<IconButton
								key={`ban-steam`}
								onClick={onNewBanSteam}
								sx={{ color: "primary.contrastText" }}
							>
								<AddIcon />
							</IconButton>
						</Tooltip>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
