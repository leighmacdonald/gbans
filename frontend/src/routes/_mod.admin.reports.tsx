import FilterListIcon from "@mui/icons-material/FilterList";
import ReportIcon from "@mui/icons-material/Report";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, MaterialReactTable, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import { apiGetReports } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { BanReason, BanReasons } from "../schema/bans.ts";
import {
	ReportStatus,
	ReportStatusCollection,
	type ReportStatusEnum,
	type ReportWithAuthor,
	reportStatusString,
} from "../schema/report.ts";
import { commonTableSearchSchema } from "../util/table.ts";

const columnHelper = createMRTColumnHelper<ReportWithAuthor>();
const defaultOptions = createDefaultTableOptions<ReportWithAuthor>();

const reportsSearchSchema = commonTableSearchSchema.extend({
	sortColumn: z
		.enum(["report_id", "source_id", "target_id", "report_status", "reason", "created_on", "updated_on"])
		.optional(),
	source_id: z.string().optional(),
	target_id: z.string().optional(),
	deleted: z.boolean().optional(),
	report_status: z.enum(ReportStatus).optional(),
});

export const Route = createFileRoute("/_mod/admin/reports")({
	component: AdminReports,
	validateSearch: (search) => reportsSearchSchema.parse(search),
	loader: async ({ context, abortController }) => {
		const reports = await context.queryClient.fetchQuery({
			queryKey: ["adminReports"],
			queryFn: async () => {
				return apiGetReports({ deleted: false }, abortController);
			},
		});
		return { reports: reports ?? [] };
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Reports" }, match.context.title("Reports")],
	}),
});

function AdminReports() {
	const navigate = useNavigate({ from: Route.fullPath });
	const search = Route.useSearch();
	const { reports } = Route.useLoaderData();

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({
				to: "/admin/reports",
				replace: true,
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onSubmit: z.object({
				source_id: z.string(),
				target_id: z.string(),
				report_status: z.enum(ReportStatus),
			}),
		},
		defaultValues: {
			source_id: search.source_id ?? "",
			target_id: search.target_id ?? "",
			report_status: search.report_status ?? ReportStatus.Any,
		},
	});

	const clear = useCallback(async () => {
		form.reset();
		await navigate({
			to: "/admin/reports",
			search: (prev) => ({
				...prev,
				source_id: undefined,
				target_id: undefined,
				report_status: undefined,
			}),
		});
	}, [form, navigate]);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("report_id", {
				enableColumnFilter: false,
				header: "ID",
				size: 30,
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
			columnHelper.accessor("report_status", {
				size: 120,
				header: "Status",
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(ReportStatus).map((status) => ({
					label: reportStatusString(status),
					value: status,
				})),
				Cell: ({ cell }) => {
					return (
						<Stack direction={"row"} spacing={1}>
							<Typography variant={"body1"}>{reportStatusString(cell.getValue())}</Typography>
						</Stack>
					);
				},
			}),
			columnHelper.accessor("source_id", {
				header: "Reporter",
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.author.steam_id}
						personaname={row.original.author.name}
						avatar_hash={row.original.author.avatarhash}
					/>
				),
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.subject.steam_id}
						personaname={row.original.subject.name}
						avatar_hash={row.original.subject.avatarhash}
					/>
				),
			}),
			columnHelper.accessor("reason", {
				enableColumnFilter: false,
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				header: "Reason",
				size: 100,
				Cell: ({ cell }) => <Typography>{BanReasons[cell.getValue()]}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				enableColumnFilter: false,
				size: 100,
				header: "Created",
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
			columnHelper.accessor("updated_on", {
				enableColumnFilter: false,
				size: 100,
				header: "Updated",
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: reports,
		enableFilters: true,
		initialState: {
			...defaultOptions.initialState,

			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
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
				<ContainerWithHeader title={"Filters"} iconLeft={<FilterListIcon />} marginTop={2}>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Grid container spacing={2}>
							<Grid size={{ xs: 6, md: 4 }}>
								<form.AppField
									name={"source_id"}
									children={(field) => {
										return <field.SteamIDField label={"Author Steam ID"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 6, md: 4 }}>
								<form.AppField
									name={"target_id"}
									children={(field) => {
										return <field.SteamIDField label={"Subject Steam ID"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 6, md: 4 }}>
								<form.AppField
									name={"report_status"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Report Status"}
												fullWidth
												items={ReportStatusCollection}
												renderItem={(item) => {
													return (
														<MenuItem value={item} key={`rs-${item}`}>
															{reportStatusString(item as ReportStatusEnum)}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppForm>
									<ButtonGroup>
										<form.ResetButton onClick={clear} />
										<form.SubmitButton />
									</ButtonGroup>
								</form.AppForm>
							</Grid>
						</Grid>
					</form>
				</ContainerWithHeader>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Current User Reports"} iconLeft={<ReportIcon />}>
					<MaterialReactTable table={table} />
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
