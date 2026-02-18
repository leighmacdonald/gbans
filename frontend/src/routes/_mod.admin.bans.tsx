import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import FilterListIcon from "@mui/icons-material/FilterList";
import GavelIcon from "@mui/icons-material/Gavel";
import UndoIcon from "@mui/icons-material/Undo";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import MenuItem from "@mui/material/MenuItem";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
	type ColumnFiltersState,
	createColumnHelper,
	type PaginationState,
	type SortingState,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiGetBans } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { ModalBan, ModalUnban } from "../component/modal";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { Title } from "../component/Title";
import { FullTable } from "../component/table/FullTable.tsx";
import { TableCellBool } from "../component/table/TableCellBool.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import {
	AppealState,
	AppealStateEnum,
	BanReason,
	BanReasonEnum,
	BanReasons,
	type BanRecord,
	banReasonsCollection,
} from "../schema/bans.ts";
import { schemaBanQueryOpts } from "../schema/query.ts";
import { initColumnFilter, initPagination, isPermanentBan, RowsPerPage } from "../util/table.ts";
import { renderDate } from "../util/time.ts";
import { emptyOrNullString } from "../util/types.ts";

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
});

function AdminBans() {
	const queryClient = useQueryClient();
	const navigate = useNavigate({ from: Route.fullPath });
	const search = Route.useSearch();
	const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
	const [sorting] = useState<SortingState>([{ id: "ban_id", desc: true }]);
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));
	const { sendFlash } = useUserFlashCtx();

	const { data: bans, isLoading } = useQuery({
		queryKey: ["bans"],
		queryFn: async () => {
			return await apiGetBans({ deleted: true });
		},
	});

	const onNewBanSteam = async () => {
		try {
			const ban = await NiceModal.show<BanRecord>(ModalBan, {});
			queryClient.setQueryData(["bans"], [...(bans ?? []), ban]);
		} catch (e) {
			sendFlash("error", `Error trying to set up ban: ${e}`);
		}
	};

	const defaultValues: z.infer<typeof schemaBanQueryOpts> = {
		source_id: search.source_id ?? "",
		target_id: search.target_id ?? "",
		appeal_state: search.appeal_state ?? AppealState.Any,
		groups_only: search.groups_only ?? false,
		deleted: search.deleted ?? false,
		cidr: search.cidr,
		cidr_only: search.cidr_only ?? false,
		reason: search.reason ?? BanReason.Any,
		include_groups: search.include_groups ?? true,
	};

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			setColumnFilters(initColumnFilter(value));
			await navigate({
				to: "/admin/bans",
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onSubmit: schemaBanQueryOpts,
		},
		defaultValues,
	});

	const clear = async () => {
		setColumnFilters([]);
		form.reset();
		await navigate({
			to: "/admin/bans",
			search: (prev) => ({
				...prev,
				source_id: undefined,
				target_id: undefined,
				reason: undefined,
				valid_until: undefined,
				groups_only: undefined,
				cidr: undefined,
				cidr_only: undefined,
				appeal_state: undefined,
				networks_only: undefined,
				include_groups: undefined,
				sortColumn: undefined,
				sortOrder: undefined,
			}),
		});
	};

	const columns = useMemo(() => {
		const onUnban = async (ban: BanRecord) => {
			try {
				await NiceModal.show(ModalUnban, {
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
		};

		const onEdit = async (ban: BanRecord) => {
			try {
				const updated = await NiceModal.show<BanRecord>(ModalBan, {
					banId: ban.ban_id,
					personaName: ban.target_personaname,
					existing: ban,
				});
				queryClient.setQueryData(
					["bans"],
					(bans ?? []).map((b) => (b.ban_id === updated.ban_id ? updated : b)),
				);
			} catch (e) {
				sendFlash("error", `Error trying to edit ban: ${e}`);
			}
		};

		return makeColumns(onEdit, onUnban);
	}, [bans, queryClient, sendFlash]);

	const filtered = useMemo(() => {
		return bans?.filter((b) => {
			if (!b) {
				return false;
			}
			if (!search.deleted && b.deleted) {
				return false;
			}
			if (search.cidr_only && emptyOrNullString(b.cidr)) {
				return false;
			}

			if (search.reason && search.reason >= 0 && b.reason !== search.reason) {
				return false;
			}

			return true;
		});
	}, [bans, search]);

	return (
		<Grid container spacing={2}>
			<Title>Bans</Title>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Filters"} iconLeft={<FilterListIcon />} marginTop={2}>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Grid container spacing={2}>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"source_id"}
									children={(field) => {
										return <field.SteamIDField label={"Author Steam ID"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"target_id"}
									children={(field) => {
										return <field.SteamIDField label={"Subject Steam ID"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"cidr"}
									children={(field) => {
										return <field.TextField label={"CIDR Range/IP Address"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"reason"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Ban Reason"}
												items={banReasonsCollection}
												renderItem={(i) => {
													if (i === undefined) {
														return null;
													}
													return (
														<MenuItem value={i} key={`${i}-${BanReasons[i]}`}>
															{BanReasons[i]}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"cidr_only"}
									children={(field) => {
										return <field.CheckboxField label={"CIDR/IP Bans Only"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"groups_only"}
									children={(field) => {
										return <field.CheckboxField label={"Show groups only"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"include_groups"}
									children={(field) => {
										return <field.CheckboxField label={"Show groups"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"deleted"}
									children={(field) => {
										return <field.CheckboxField label={"Show deleted/expired"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppForm>
									<ButtonGroup>
										<form.ClearButton onClick={clear} />
										<form.ResetButton />
										<form.SubmitButton />
									</ButtonGroup>
								</form.AppForm>
							</Grid>
						</Grid>
					</form>
				</ContainerWithHeader>
			</Grid>

			<Grid size={{ xs: 12 }}>
				<ContainerWithHeaderAndButtons
					title={"Steam Ban History"}
					marginTop={0}
					iconLeft={<GavelIcon />}
					buttons={[
						<Button
							key={`ban-steam`}
							variant={"contained"}
							color={"success"}
							startIcon={<AddIcon />}
							sx={{ marginRight: 2 }}
							onClick={onNewBanSteam}
						>
							Create
						</Button>,
					]}
				>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<FullTable
								columnFilters={columnFilters}
								pagination={pagination}
								setPagination={setPagination}
								data={filtered ?? []}
								isLoading={isLoading}
								columns={columns}
								sorting={sorting}
								toOptions={{ from: Route.fullPath }}
							/>
						</Grid>
					</Grid>
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
}

const columnHelper = createColumnHelper<BanRecord>();

const makeColumns = (onEdit: (ban: BanRecord) => Promise<void>, onUnban: (ban: BanRecord) => Promise<void>) => [
	columnHelper.accessor("ban_id", {
		enableColumnFilter: false,
		size: 50,
		header: "Ban ID",
		cell: (info) => (
			<TextLink to={`/ban/$ban_id`} params={{ ban_id: String(info.getValue()) }}>
				{`#${info.getValue()}`}
			</TextLink>
		),
	}),
	columnHelper.accessor("source_id", {
		header: "Author",
		cell: (info) => {
			return typeof info.row.original === "undefined" ? (
				""
			) : (
				<PersonCell
					steam_id={info.row.original.source_id}
					personaname={info.row.original.source_personaname}
					avatar_hash={info.row.original.source_avatarhash}
				/>
			);
		},
	}),
	columnHelper.accessor("target_id", {
		header: "Subject",
		cell: (info) => {
			return typeof info.row.original === "undefined" ? (
				""
			) : (
				<PersonCell
					showCopy={true}
					steam_id={info.row.original.target_id}
					personaname={info.row.original.target_personaname}
					avatar_hash={info.row.original.target_avatarhash}
				/>
			);
		},
	}),
	columnHelper.accessor("cidr", {
		enableColumnFilter: true,
		size: 150,
		// filterFn: (row, _, filterValue) => {
		//     return filterValue == BanReason.Any || row.original.reason == filterValue;
		// },
		header: "CIDR/IP",
		cell: (info) => <Typography>{info.getValue()}</Typography>,
	}),
	columnHelper.accessor("reason", {
		enableColumnFilter: true,
		size: 150,
		filterFn: (row, _, filterValue) => {
			return filterValue === BanReason.Any || row.original.reason === filterValue;
		},
		header: "Reason",
		cell: (info) => <Typography>{BanReasons[info.getValue() as BanReasonEnum]}</Typography>,
	}),
	columnHelper.accessor("created_on", {
		header: "Created",
		size: 100,
		cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>,
	}),
	columnHelper.accessor("valid_until", {
		header: "Duration",
		size: 100,
		cell: (info) => {
			return typeof info.row.original === "undefined" ? (
				""
			) : isPermanentBan(info.row.original.created_on, info.row.original.valid_until) ? (
				"Permanent"
			) : (
				<TableCellRelativeDateField
					date={info.row.original.created_on}
					compareDate={info.row.original.valid_until}
				/>
			);
		},
	}),
	columnHelper.accessor("evade_ok", {
		meta: {
			tooltip: "Evasion OK. Players connecting from the same ip will not be banned.",
		},
		size: 30,
		header: "E",
		cell: (info) => <TableCellBool enabled={info.getValue()} />,
	}),
	columnHelper.accessor("deleted", {
		size: 30,
		meta: { tooltip: "Deleted / Expired Bans" },
		filterFn: (row, _, filterValue) => {
			return filterValue ? true : !row.original.deleted;
		},
		header: "D",
		cell: (info) => <TableCellBool enabled={info.getValue()} />,
	}),
	columnHelper.accessor("report_id", {
		header: "Rep.",
		size: 60,
		meta: { tooltip: "Linked report" },
		cell: (info) =>
			Boolean(info.getValue()) && (
				<TextLink to={`/report/$reportId`} params={{ reportId: String(info.getValue()) }}>
					{`#${info.getValue()}`}
				</TextLink>
			),
	}),
	columnHelper.display({
		id: "edit",
		size: 30,
		cell: (info) => (
			<IconButton
				color={"warning"}
				onClick={async () => {
					await onEdit(info.row.original);
				}}
			>
				<Tooltip title={"Edit Ban"}>
					<EditIcon />
				</Tooltip>
			</IconButton>
		),
	}),
	columnHelper.display({
		id: "unban",
		size: 30,
		cell: (info) => (
			<IconButton
				color={"success"}
				onClick={async () => {
					await onUnban(info.row.original);
				}}
			>
				<Tooltip title={"Remove Ban"}>
					<UndoIcon />
				</Tooltip>
			</IconButton>
		),
	}),
];
