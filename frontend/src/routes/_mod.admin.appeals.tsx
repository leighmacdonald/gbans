import FilterListIcon from "@mui/icons-material/FilterList";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, MaterialReactTable, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { z } from "zod/v4";
import { apiGetAppeals, appealStateString } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import {
	AppealState,
	AppealStateCollection,
	type AppealStateEnum,
	BanReason,
	BanReasons,
	type BanRecord,
} from "../schema/bans.ts";
import { commonTableSearchSchema } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<BanRecord>();
const defaultOptions = createDefaultTableOptions<BanRecord>();

const appealSearchSchema = commonTableSearchSchema.extend({
	sortColumn: z
		.enum(["report_id", "source_id", "target_id", "appeal_state", "reason", "created_on", "updated_on"])
		.optional(),
	source_id: z.string().optional(),
	target_id: z.string().optional(),
	appeal_state: z.enum(AppealState).optional(),
});

const schema = z.object({
	source_id: z.string(),
	target_id: z.string(),
	appeal_state: z.enum(AppealState),
});

export const Route = createFileRoute("/_mod/admin/appeals")({
	component: AdminAppeals,
	validateSearch: (search) => appealSearchSchema.parse(search),
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Appeals" }, match.context.title("Appeals")],
	}),
	loader: async () => {
		const data = await apiGetAppeals({});
		return { data };
	},
});

function AdminAppeals() {
	const navigate = useNavigate({ from: Route.fullPath });
	const search = Route.useSearch();

	const defaultValues: z.infer<typeof schema> = {
		source_id: search.source_id ?? "",
		target_id: search.target_id ?? "",
		appeal_state: search.appeal_state ?? AppealState.Any,
	};
	const { data } = useQuery({
		queryKey: ["appeals"],
		queryFn: async () => {
			return (await apiGetAppeals({})) ?? [];
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({
				to: "/admin/appeals",
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onChange: schema,
		},
		defaultValues,
	});

	const clear = async () => {
		//reset();
		form.setFieldValue("appeal_state", AppealState.Any);
		form.setFieldValue("source_id", "");
		form.setFieldValue("target_id", "");

		await form.handleSubmit();
		await navigate({
			to: "/admin/appeals",
			search: (prev) => ({
				...prev,
				source_id: undefined,
				target_id: undefined,
				appeal_state: undefined,
			}),
		});
	};
	const columns = useMemo(
		() => [
			columnHelper.accessor("ban_id", {
				enableColumnFilter: false,
				header: "ID",
				grow: false,
				size: 30,
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
				enableColumnFilter: true,
				header: "Status",
				size: 150,
				grow: false,
				Cell: ({ cell }) => {
					return <Typography variant={"body1"}>{appealStateString(cell.getValue())}</Typography>;
				},
			}),
			columnHelper.accessor("source_id", {
				enableColumnFilter: true,
				header: "Author",
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.source_id}
						personaname={row.original.source_personaname}
						avatar_hash={row.original.source_avatarhash}
					/>
				),
			}),
			columnHelper.accessor("target_id", {
				enableColumnFilter: true,
				header: "Subject",
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.target_id}
						personaname={row.original.target_personaname}
						avatar_hash={row.original.target_avatarhash}
					/>
				),
			}),
			columnHelper.accessor("reason", {
				header: "Reason",
				size: 150,
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				filterVariant: "multi-select",
				Cell: ({ cell }) => <Typography>{BanReasons[cell.getValue()]}</Typography>,
			}),
			columnHelper.accessor("reason_text", {
				enableColumnFilter: false,
				header: "Custom",
				filterVariant: "text",
				grow: true,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				size: 120,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
			columnHelper.accessor("updated_on", {
				header: "Last Active",
				enableColumnFilter: false,
				size: 120,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
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
									name={"appeal_state"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Appeal Status"}
												items={AppealStateCollection}
												renderItem={(item) => {
													return (
														<MenuItem value={item} key={`rs-${item}`}>
															{appealStateString(item as AppealStateEnum)}
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
				<ContainerWithHeader title={"Recent Open Appeal Activity"}>
					<MaterialReactTable table={table} />
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
