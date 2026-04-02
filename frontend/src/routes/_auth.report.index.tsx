import EditNotificationsIcon from "@mui/icons-material/EditNotifications";
import InfoIcon from "@mui/icons-material/Info";
import VisibilityIcon from "@mui/icons-material/Visibility";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import MenuItem from "@mui/material/MenuItem";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { type JSX, useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiCreateReport, apiGetUserReports } from "../api";
import { ButtonLink } from "../component/ButtonLink.tsx";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { mdEditorRef } from "../component/form/field/MarkdownField.tsx";
import { IconButtonLink } from "../component/IconButtonLink.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { PlayerMessageContext } from "../component/PlayerMessageContext.tsx";
import { ReportStatusIcon } from "../component/ReportStatusIcon.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { BanReason, BanReasons, banReasonsReportCollection } from "../schema/bans.ts";
import {
	type CreateReportRequest,
	ReportStatus,
	type ReportWithAuthor,
	reportStatusString,
	schemaCreateReportRequest,
} from "../schema/report.ts";
import { commonTableSearchSchema } from "../util/table.ts";
import { emptyOrNullString } from "../util/types.ts";

const validateSearch = commonTableSearchSchema.extend({
	rows: z.number().optional(),
	sortColumn: z.enum(["report_status", "created_on"]).optional(),
	steam_id: z.string().optional(),
	demo_id: z.number().optional(),
	person_message_id: z.number().optional(),
});

export const Route = createFileRoute("/_auth/report/")({
	component: ReportCreate,
	validateSearch,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Create a new player report" }, match.context.title("Create Report")],
	}),
});

function ReportCreate() {
	const { profile } = useAuth();
	const canReport = useMemo(() => {
		return profile.steam_id && profile.ban_id === 0;
	}, [profile]);

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 8 }}>
				<Stack spacing={2}>
					{canReport ? (
						<ReportCreateForm />
					) : (
						<ContainerWithHeader title={"Permission Denied"}>
							<Typography variant={"body1"} padding={2}>
								You are unable to report players while you are currently banned/muted.
							</Typography>
							<ButtonGroup sx={{ padding: 2 }}>
								<ButtonLink
									variant={"contained"}
									color={"primary"}
									to={"/ban/$ban_id"}
									params={{ ban_id: profile.ban_id.toString() }}
								>
									Appeal Ban
								</ButtonLink>
							</ButtonGroup>
						</ContainerWithHeader>
					)}

					<UserReportHistory />
				</Stack>
			</Grid>
			<Grid size={{ xs: 12, md: 4 }}>
				<ContainerWithHeader title={"Reporting Guide"} iconLeft={<InfoIcon />}>
					<List>
						<ListItem>
							<ListItemText>
								Once your report is posted, it will be reviewed by a moderator. If further details are
								required you will be notified about it.
							</ListItemText>
						</ListItem>
						<ListItem>
							<ListItemText>
								If you wish to link to a specific SourceTV recording, you can find them listed{" "}
								<Link component={RouterLink} to={"/stv"}>
									here
								</Link>
								. Once you find the recording you want, you may select the report icon which will open a
								new report with the demo attached. From there you will optionally be able to enter a
								specific tick if you have one.
							</ListItemText>
						</ListItem>
						<ListItem>
							<ListItemText>
								Reports that are made in bad faith, or otherwise are considered to be trolling will be
								closed, and the reporter will be banned.
							</ListItemText>
						</ListItem>

						<ListItem>
							<ListItemText>
								Its only possible to open a single report against a particular player. If you wish to
								add more evidence or discuss further an existing report, please open the existing report
								and add it by creating a new message there. You can see your current report history
								below.
							</ListItemText>
						</ListItem>
					</List>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}

const columnHelper = createMRTColumnHelper<ReportWithAuthor>();
const defaultOptions = createDefaultTableOptions<ReportWithAuthor>();

const UserReportHistory = () => {
	const { data, isLoading, isError } = useQuery({
		queryKey: ["history"],
		queryFn: async ({ signal }) => {
			return await apiGetUserReports(signal);
		},
	});

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("report_status", {
				header: "Status",
				size: 150,
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(ReportStatus).map((status) => ({
					label: reportStatusString(status),
					value: status,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(ReportStatus.Any) ||
						filterValue.includes(row.original.report_status)
					);
				},
				Cell: ({ cell }) => {
					return (
						<Stack direction={"row"} spacing={1}>
							<ReportStatusIcon reportStatus={cell.getValue()} />
							<Typography variant={"body1"}>{reportStatusString(cell.getValue())}</Typography>
						</Stack>
					);
				},
			}),
			columnHelper.accessor("subject", {
				header: "Player",
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.subject.name.toLowerCase();
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
						personaname={
							emptyOrNullString(row.original.subject.name)
								? row.original.subject.steam_id
								: row.original.subject.name
						}
						avatar_hash={row.original.subject.avatarhash}
					/>
				),
			}),
			columnHelper.accessor("report_id", {
				header: "View",
				enableColumnActions: false,
				enableColumnFilter: false,
				enableSorting: false,
				size: 85,
				grow: false,
				Cell: ({ cell }) => (
					<ButtonGroup variant={"text"}>
						<IconButtonLink
							color={"primary"}
							to={`/report/$reportId`}
							params={{ reportId: String(cell.getValue()) }}
						>
							<Tooltip title={"View"}>
								<VisibilityIcon />
							</Tooltip>
						</IconButtonLink>
					</ButtonGroup>
				),
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			pagination: {
				pageSize: 10,
				pageIndex: 0,
			},
			sorting: [{ id: "report_id", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
			},
		},
	});

	return <SortableTable table={table} title={"Your Report History"} />;
};

const ReportCreateForm = (): JSX.Element => {
	const { demo_id, steam_id, person_message_id } = Route.useSearch();
	const { sendFlash, sendError } = useUserFlashCtx();
	const [isCustom, setIsCustom] = useState(false);

	const defaultValues: z.infer<typeof schemaCreateReportRequest> = {
		description: "",
		demo_id: demo_id ?? 0,
		demo_tick: 0,
		person_message_id: person_message_id ?? 0,
		target_id: steam_id ?? "",
		reason: person_message_id ? BanReason.Language : BanReason.Cheating,
		reason_text: "",
	};

	const mutation = useMutation({
		mutationFn: async (variables: CreateReportRequest) => {
			const ac = new AbortController();
			return await apiCreateReport(variables, ac.signal);
		},
		onSuccess: async (data) => {
			mdEditorRef.current?.setMarkdown("");
			await navigate({
				to: "/report/$reportId",
				params: { reportId: String(data.report_id) },
			});
			sendFlash("success", "Created report successfully");
		},
		onError: sendError,
	});

	const navigate = useNavigate();

	const form = useAppForm({
		onSubmit: ({ value }) => {
			mutation.mutate({
				demo_id: value.demo_id ?? 0,
				target_id: value.target_id,
				demo_tick: value.demo_tick,
				reason: value.reason,
				reason_text: value.reason_text,
				description: value.description,
				person_message_id: value.person_message_id,
			});
		},
		validators: {
			onSubmitAsync: schemaCreateReportRequest,
		},
		defaultValues,
	});

	return (
		<ContainerWithHeader title={"Create New Report"} iconLeft={<EditNotificationsIcon />} spacing={2} marginTop={3}>
			<form
				id={"reportForm"}
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<Grid container spacing={2}>
					<Grid size={{ xs: 12 }}>
						<form.AppField
							name={"target_id"}
							children={(field) => {
								return <field.SteamIDField disabled={Boolean(steam_id)} label={"SteamID"} />;
							}}
						/>
					</Grid>
					<Grid size={{ xs: 6 }}>
						<form.AppField
							name={"reason"}
							children={(field) => {
								return (
									<field.SelectField
										label={"Ban Reason"}
										items={banReasonsReportCollection}
										handleChange={(value) => {
											setIsCustom(value === BanReason.Custom);
											field.handleChange(value);
										}}
										renderItem={(r) => {
											return (
												<MenuItem value={r} key={`reason-${r}`}>
													{BanReasons[r]}
												</MenuItem>
											);
										}}
									/>
								);
							}}
						/>
					</Grid>

					<Grid size={{ xs: 6 }}>
						<form.AppField
							name={"reason_text"}
							children={(field) => {
								return (
									<field.TextField
										fullWidth
										disabled={!isCustom}
										label="Custom Reason"
										helperText={"You must set the reason to Custom to use this field"}
									/>
								);
							}}
						/>
					</Grid>
					{Boolean(demo_id) && (
						<>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"demo_id"}
									children={(field) => {
										return <field.TextField disabled={Boolean(demo_id)} label="Demo ID" />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"demo_tick"}
									children={(field) => {
										return (
											<field.TextField disabled={!demo_id} label="Demo Tick" variant="outlined" />
										);
									}}
								/>
							</Grid>
						</>
					)}
					{person_message_id !== undefined && person_message_id > 0 && (
						<Grid size={{ md: 12 }}>
							<PlayerMessageContext playerMessageId={person_message_id} padding={5} />
						</Grid>
					)}
					<Grid size={{ xs: 12 }}>
						<form.AppField
							name={"description"}
							children={(field) => {
								return <field.MarkdownField label={"Message (Markdown)"} />;
							}}
						/>
					</Grid>
					<Grid size={{ xs: 12 }}>
						<form.AppForm>
							<ButtonGroup>
								<form.ResetButton />
								<form.SubmitButton />
							</ButtonGroup>
						</form.AppForm>
					</Grid>
				</Grid>
			</form>
		</ContainerWithHeader>
	);
};
