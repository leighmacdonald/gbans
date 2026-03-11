/** biome-ignore-all lint/correctness/noChildrenProp: form */
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
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, MaterialReactTable, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import { apiGetBans } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { BanModal } from "../component/modal/BanModal.tsx";
import { UnbanModal } from "../component/modal/UnbanModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
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
	const navigate = useNavigate({ from: Route.fullPath });
	const search = Route.useSearch();
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

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			//setColumnFilters(initColumnFilter(value));
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
		//setColumnFilters([]);
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
		return [
			columnHelper.accessor("ban_id", {
				enableColumnFilter: false,
				size: 75,
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
				// filterFn: (row, _, filterValue) => {
				// 	return filterValue === BanReason.Any || row.original.reason === filterValue;
				// },
				header: "Reason",
				Cell: ({ cell }) => <Typography>{BanReasons[cell.getValue() as BanReasonEnum]}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				filterVariant: "date-range",
				grow: false,
				size: 100,
				Cell: ({ cell }) => <Typography>{renderDate(cell.getValue() as Date)}</Typography>,
			}),
			columnHelper.accessor("valid_until", {
				header: "Duration",
				enableColumnFilter: false,
				grow: true,
				filterVariant: "date-range",
				size: 100,
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
				size: 30,
				enableColumnFilter: false,
				grow: false,
				filterVariant: "checkbox",
				header: "Evade Ok",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("deleted", {
				size: 30,
				enableColumnFilter: false,
				grow: false,
				filterVariant: "checkbox",
				meta: { tooltip: "Deleted / Expired Bans" },
				header: "Unbanned",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("report_id", {
				header: "Rep.",
				size: 60,
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
		initialState: {
			...defaultOptions.initialState,

			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				valid_until: true,
				created_on: false,
				updated_on: true,
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
							<MaterialReactTable table={table} />
						</Grid>
					</Grid>
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
}
