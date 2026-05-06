import { useMutation, useQuery } from "@connectrpc/connect-query";
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
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { type JSX, useMemo, useState } from "react";
import { z } from "zod/v4";
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
import { BanReason } from "../rpc/ban/v1/ban_pb.ts";
import { ReportStatus, type ReportWithAuthor } from "../rpc/ban/v1/report_pb.ts";
import { reportCreate, reports } from "../rpc/ban/v1/report-ReportService_connectquery.ts";
import { enumValues } from "../util/lists.ts";
import { commonTableSearchSchema } from "../util/table.ts";
import { emptyOrNullString } from "../util/types.ts";

const validateSearch = commonTableSearchSchema.extend({
	rows: z.number().optional(),
	sortColumn: z.enum(["report_status", "created_on"]).optional(),
	steamId: z.bigint().optional(),
	demoId: z.number().optional(),
	personMessageId: z.bigint().optional(),
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
		return profile.steamId && profile.steamId === 0n;
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
									to={"/ban/$banId"}
									params={{ banId: profile.banId.toString() }}
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
	const { data, isLoading, isError } = useQuery(reports);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("report.reportStatus", {
				header: "Status",
				size: 150,
				filterVariant: "multi-select",
				filterSelectOptions: enumValues(ReportStatus).map((status) => ({
					label: ReportStatus[status],
					value: status,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(ReportStatus.OPENED_UNSPECIFIED) ||
						filterValue.includes(row.original.report?.reportStatus)
					);
				},
				Cell: ({ cell }) => {
					return (
						<Stack direction={"row"} spacing={1}>
							<ReportStatusIcon reportStatus={cell.getValue()} />
							<Typography variant={"body1"}>{ReportStatus[cell.getValue()]}</Typography>
						</Stack>
					);
				},
			}),
			columnHelper.accessor("subject", {
				header: "Player",
				filterFn: (row, _, filterValue) => {
					const query: string = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = String(row.original.subject?.name.toLowerCase());
					if (value !== "" && value.includes(query)) {
						return true;
					}
					if (
						row.original.subject?.steamId.toString().includes(query) ||
						row.original.subject?.steamId.toString() === query
					) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						steamId={String(row.original.subject?.steamId)}
						personaName={String(
							emptyOrNullString(row.original.subject?.name)
								? row.original.subject?.steamId.toString()
								: row.original.subject?.name,
						)}
						avatarHash={String(row.original.subject?.avatarHash)}
					/>
				),
			}),
			columnHelper.accessor("report.reportId", {
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
		data: data?.reports ?? [],
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
			sorting: [{ id: "reportId", desc: true }],
			columnVisibility: {
				sourceId: false,
				targetId: true,
				reason: true,
			},
		},
	});

	return <SortableTable table={table} title={"Your Report History"} />;
};

const ReportCreateForm = (): JSX.Element => {
	const { demoId, steamId, personMessageId } = Route.useSearch();
	const { sendFlash, sendError } = useUserFlashCtx();
	const [isCustom, setIsCustom] = useState(false);

	const defaultValues = {
		description: "",
		demoId: demoId ?? 0,
		demoTick: 0,
		personMessageId: personMessageId ?? 0n,
		targetId: steamId ?? "",
		reason: personMessageId ? BanReason.LANGUAGE : BanReason.CHEATING,
		reasonText: "",
	};

	const mutation = useMutation(reportCreate, {
		onSuccess: async (data) => {
			mdEditorRef.current?.setMarkdown("");
			await navigate({
				to: "/report/$reportId",
				params: { reportId: String(data.report?.report?.reportId) },
			});
			sendFlash("success", "Created report successfully");
		},
		onError: sendError,
	});

	const navigate = useNavigate();

	const form = useAppForm({
		onSubmit: ({ value }) => {
			mutation.mutate({
				demoId: BigInt(value.demoId ?? 0n),
				targetId: BigInt(value.targetId),
				demoTick: value.demoTick,
				reason: value.reason,
				reasonText: value.reasonText,
				description: value.description,
				personMessageId: BigInt(value.personMessageId),
			});
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
							name={"targetId"}
							children={(field) => {
								return <field.SteamIDField disabled={Boolean(steamId)} label={"SteamID"} />;
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
										items={enumValues(BanReason)}
										handleChange={(value) => {
											setIsCustom(value === BanReason.CUSTOM);
											field.handleChange(value);
										}}
										renderItem={(r) => {
											return (
												<MenuItem value={r} key={`reason-${r}`}>
													{BanReason[r]}
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
							name={"reasonText"}
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
					{Boolean(demoId) && (
						<>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"demoId"}
									children={(field) => {
										return <field.TextField disabled={Boolean(demoId)} label="Demo ID" />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6 }}>
								<form.AppField
									name={"demoTick"}
									children={(field) => {
										return (
											<field.TextField disabled={!demoId} label="Demo Tick" variant="outlined" />
										);
									}}
								/>
							</Grid>
						</>
					)}
					{personMessageId !== undefined && personMessageId > 0 && (
						<Grid size={{ md: 12 }}>
							<PlayerMessageContext playerMessageId={personMessageId} padding={5} />
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
