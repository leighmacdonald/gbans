/** biome-ignore-all lint/correctness/noChildrenProp: form */

import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import UndoIcon from "@mui/icons-material/Undo";
import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
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
import { BanModal } from "../component/modal/BanModal.tsx";
import { UnbanModal } from "../component/modal/UnbanModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { type Ban, BanReason } from "../rpc/ban/v1/ban_pb.ts";
import { query } from "../rpc/ban/v1/ban-BanService_connectquery.ts";
import { isPermanentBan } from "../util/table.ts";
import { renderTimestamp } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Ban>();
const defaultOptions = createDefaultTableOptions<Ban>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "banId" });
const validateSearch = makeSchemaState("banId");

export const Route = createFileRoute("/_mod/admin/bans")({
	component: AdminBans,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
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

	const { data, isLoading, isError } = useQuery(query, {});

	const onNewBanSteam = useCallback(async () => {
		try {
			const ban = (await NiceModal.show(BanModal, {})) as Ban;
			queryClient.setQueryData(["bans"], [...(data?.bans ?? []), ban]);
		} catch (e) {
			sendFlash("error", `Error trying to set up ban: ${e}`);
		}
	}, [queryClient, sendFlash, data]);

	const onUnban = useCallback(
		async (ban: Ban) => {
			try {
				await NiceModal.show(UnbanModal, {
					banId: ban.banId,
					personaName: ban.targetPersonaName,
				});
				queryClient.setQueryData(
					["bans"],
					(data?.bans ?? []).filter((b) => b.banId !== ban.banId),
				);
				sendFlash("success", "Unbanned player successfully");
			} catch (e) {
				sendFlash("error", `Error trying to unban: ${e}`);
			}
		},
		[queryClient, sendFlash, data],
	);

	const onEdit = useCallback(
		async (ban: Ban) => {
			try {
				const updated = (await NiceModal.show(BanModal, {
					banId: ban.banId,
					personaName: ban.targetPersonaName,
					existing: ban,
				})) as Ban;
				queryClient.setQueryData(
					["bans"],
					(data?.bans ?? []).map((b) => (b.banId === updated.banId ? updated : b)),
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
			columnHelper.accessor("banId", {
				grow: false,
				header: "Ban ID",
				Cell: ({ cell }) => (
					<TextLink to={`/ban/$banId`} params={{ banId: String(cell.getValue()) }}>
						{`#${cell.getValue()}`}
					</TextLink>
				),
			}),
			columnHelper.accessor("sourceId", {
				header: "Author",
				enableSorting: false,
				grow: false,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.sourcePersonaName.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.sourceId.toString().includes(query) || row.original.sourceId === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => {
					return (
						<PersonCell
							steamId={row.original.sourceId}
							personaName={row.original.sourcePersonaName}
							avatarHash={row.original.sourceAvatarHash}
						>
							<RouterLink
								style={{
									color:
										theme.palette.mode === "dark"
											? theme.palette.primary.light
											: theme.palette.primary.dark,
								}}
								to={Route.fullPath}
								search={setColumnFilter(search, "source_id", row.original.sourceId)}
							>
								{row.original.sourcePersonaName ?? row.original.sourceId}
							</RouterLink>
						</PersonCell>
					);
				},
			}),
			columnHelper.accessor("targetId", {
				header: "Subject",
				grow: false,
				enableSorting: false,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.targetPersonaName.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.targetId.toString().includes(query) || row.original.targetId === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						steamId={row.original.targetId}
						personaName={row.original.targetPersonaName}
						avatarHash={row.original.targetAvatarHash}
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
							{row.original.targetPersonaName ?? row.original.targetId}
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
					label: BanReason[reason as BanReason],
					value: reason,
				})),
				filterVariant: "multi-select",
				header: "Reason",
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(BanReason.UNSPECIFIED) ||
						filterValue.includes(row.original.reason)
					);
				},
				Cell: ({ cell }) => (
					<TextLink to={Route.fullPath} search={setColumnFilter(search, "reason", [cell.getValue()])}>
						{BanReason[cell.getValue() as BanReason]}
					</TextLink>
				),
			}),
			columnHelper.accessor("createdOn", {
				header: "Created",
				filterVariant: "date-range",
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
			columnHelper.accessor("validUntil", {
				header: "Duration",
				enableColumnFilter: false,
				grow: false,
				filterVariant: "date-range",
				Cell: ({ row }) => {
					return typeof row.original === "undefined" ? (
						""
					) : isPermanentBan(
							timestampDate(row.original.createdOn as Timestamp),
							timestampDate(row.original.validUntil as Timestamp),
						) ? (
						"Permanent"
					) : (
						<TableCellRelativeDateField
							date={timestampDate(row.original.createdOn as Timestamp)}
							compareDate={timestampDate(row.original.validUntil as Timestamp)}
						/>
					);
				},
			}),
			columnHelper.accessor("evadeOk", {
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
			columnHelper.accessor("reportId", {
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
		data: data?.bans ?? [],
		enableFilters: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
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
		renderRowActions: ({ row }) => (
			<RowActionContainer>
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
				</IconButton>
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
				</IconButton>
			</RowActionContainer>
		),
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
