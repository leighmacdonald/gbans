/** biome-ignore-all lint/correctness/noChildrenProp: form */
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import UndoIcon from "@mui/icons-material/Undo";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import { apiGetBans } from "../api";
import { BanModal } from "../component/modal/BanModal.tsx";
import { UnbanModal } from "../component/modal/UnbanModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { AppealStateEnum, BanReason, BanReasonEnum, BanReasons, type BanRecord } from "../schema/bans.ts";
import { isPermanentBan, RowsPerPage } from "../util/table.ts";
import { renderDate } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<BanRecord>();
const defaultOptions = createDefaultTableOptions<BanRecord>();

const searchSchema = z.object({
	pageIndex: z.number().optional().catch(0),
	pageSize: z.number().optional().catch(RowsPerPage.TwentyFive),
	sortOrder: z.enum(["desc", "asc"]).optional().catch("desc"),
	sortColumn: z.enum(["ban_id", "source_id", "target_id", "reason", "created_on", "updated_on"]).optional(),
	source_id: z.string().optional(),
	target_id: z.string().optional(),
	appeal_state: AppealStateEnum.optional(),
	groups_only: z.boolean().optional(),
	deleted: z.boolean().optional(),
	cidr: z.string().optional(),
	cidr_only: z.boolean().optional(),
	reason: BanReasonEnum.optional(),
	include_groups: z.boolean().optional(),
});

export const Route = createFileRoute("/_mod/admin/bans")({
	component: AdminBans,
	validateSearch: (search) => searchSchema.parse(search),
	loader: async ({ context }) => {
		const bans = await context.queryClient.fetchQuery({
			queryKey: ["bans"],

			queryFn: async () => {
				return await apiGetBans({ deleted: true });
			},
		});

		return { bans };
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Bans" }, match.context.title("Bans")],
	}),
});

function AdminBans() {
	const queryClient = useQueryClient();
	const { bans } = Route.useLoaderData();
	const { sendFlash } = useUserFlashCtx();

	const onNewBanSteam = async () => {
		try {
			const ban = (await NiceModal.show(BanModal, {})) as BanRecord;
			queryClient.setQueryData(["bans"], [...(bans ?? []), ban]);
		} catch (e) {
			sendFlash("error", `Error trying to set up ban: ${e}`);
		}
	};

	const onUnban = useCallback(
		async (ban: BanRecord) => {
			try {
				await NiceModal.show(UnbanModal, {
					banId: ban.ban_id,
					personaName: ban.target_personaname,
				});
				queryClient.setQueryData(
					["bans"],
					(bans ?? []).filter((b) => b.ban_id !== ban.ban_id),
				);
				sendFlash("success", "Unbanned player successfully");
			} catch (e) {
				sendFlash("error", `Error trying to unban: ${e}`);
			}
		},
		[queryClient, sendFlash, bans],
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
					(bans ?? []).map((b) => (b.ban_id === updated.ban_id ? updated : b)),
				);
			} catch (e) {
				sendFlash("error", `Error trying to edit ban: ${e}`);
			}
		},
		[queryClient, sendFlash, bans],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("ban_id", {
				size: 125,
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
					return typeof row.original === "undefined" ? (
						""
					) : (
						<PersonCell
							steam_id={row.original.source_id}
							personaname={row.original.source_personaname}
							avatar_hash={row.original.source_avatarhash}
						/>
					);
				},
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				grow: true,
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
				Cell: ({ row }) => {
					return typeof row.original === "undefined" ? (
						""
					) : (
						<PersonCell
							showCopy={true}
							steam_id={row.original.target_id}
							personaname={row.original.target_personaname}
							avatar_hash={row.original.target_avatarhash}
						/>
					);
				},
			}),
			columnHelper.accessor("cidr", {
				enableColumnFilter: true,
				size: 150,
				grow: false,
				filterVariant: "text",
				// filterFn: (row, _, filterValue) => {
				//     return filterValue == BanReason.Any || row.original.reason == filterValue;
				// },
				header: "CIDR/IP",
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("reason", {
				enableColumnFilter: true,
				size: 150,
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
				Cell: ({ cell }) => <Typography>{BanReasons[cell.getValue() as BanReasonEnum]}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				filterVariant: "date-range",
				grow: false,
				Cell: ({ cell }) => <Typography>{renderDate(cell.getValue() as Date)}</Typography>,
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
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: bans,
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		initialState: {
			...defaultOptions.initialState,

			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				evade_ok: false,
				deleted: false,
				valid_until: true,
				created_on: false,
				updated_on: true,
				active: false,
				report_id: true,
				cidr: false,
			},
		},
		enableRowActions: true,
		renderTopToolbarCustomActions: () => {
			return <Typography variant="h3">Bans</Typography>;
		},
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
								sx={{ marginRight: 2, color: "primary.contrastText" }}
								onClick={onNewBanSteam}
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
