import DescriptionIcon from "@mui/icons-material/Description";
import FileDownloadIcon from "@mui/icons-material/FileDownload";
import LanIcon from "@mui/icons-material/Lan";
import MessageIcon from "@mui/icons-material/Message";
import QuickreplyIcon from "@mui/icons-material/Quickreply";
import ReportIcon from "@mui/icons-material/Report";
import VideocamIcon from "@mui/icons-material/Videocam";
import TabContext from "@mui/lab/TabContext";
import TabList from "@mui/lab/TabList";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Tab from "@mui/material/Tab";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouteContext } from "@tanstack/react-router";
import { type JSX, type SyntheticEvent, useState } from "react";
import { z } from "zod/v4";
import { apiCreateReportMessage, apiGetConnections, apiGetMessages } from "../api";
import { useAppForm } from "../contexts/formContext.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { reportMessagesQueryOptions } from "../queries/reportMessages.ts";
import { PermissionLevel } from "../schema/people.ts";
import type { Report } from "../schema/report.ts";
import { RowsPerPage } from "../util/table.ts";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { ContainerWithHeaderAndButtons } from "./ContainerWithHeaderAndButtons.tsx";
import { mdEditorRef } from "./form/field/MarkdownField.tsx";
import { PaginatorLocal } from "./forum/PaginatorLocal.tsx";
import { MarkDownRenderer } from "./MarkdownRenderer";
import { PlayerMessageContext } from "./PlayerMessageContext";
import { ReportMessageView } from "./ReportMessageView";
import { SourceBansList } from "./SourceBansList";
import { TabPanel } from "./TabPanel";
import { ChatTable } from "./table/ChatTable.tsx";
import { IPHistoryTable } from "./table/IPHistoryTable.tsx";

export const ReportViewComponent = ({ report }: { report: Report }): JSX.Element => {
	const theme = useTheme();
	const queryClient = useQueryClient();
	const { sendFlash, sendError } = useUserFlashCtx();
	const [value, setValue] = useState<number>(0);
	const { hasPermission } = useRouteContext({
		from: "/_auth/report/$reportId",
	});

	const [chatPagination, setChatPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const [connectionPagination, setConnectionPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const { data: connections, isLoading: isLoadingConnections } = useQuery({
		queryKey: ["reportConnectionHist", { steamId: report.target_id }],
		queryFn: async () => {
			return await apiGetConnections({
				limit: 1000,
				offset: 0,
				order_by: "person_connection_id",
				desc: true,
				source_id: report.target_id,
			});
		},
	});

	const { data: chat, isLoading: isLoadingChat } = useQuery({
		queryKey: ["reportChat", { steamId: report.target_id }],
		queryFn: async () => {
			return await apiGetMessages({
				personaname: "",
				query: "",
				source_id: report.target_id,
				limit: 2500,
				offset: 0,
				order_by: "person_message_id",
				desc: true,
				flagged_only: false,
			});
		},
	});

	const { data: messages, isLoading: isLoadingMessages } = useQuery(reportMessagesQueryOptions(report.report_id));

	const handleChange = (_: SyntheticEvent, newValue: number) => {
		setValue(newValue);
	};

	const createMessageMutation = useMutation({
		mutationFn: async ({ body_md }: { body_md: string }) => {
			return await apiCreateReportMessage(report.report_id, body_md);
		},
		onSuccess: (message) => {
			queryClient.setQueryData(reportMessagesQueryOptions(report.report_id).queryKey, [
				...(messages ?? []),
				message,
			]);
			mdEditorRef.current?.setMarkdown("");
			form.reset();
			sendFlash("success", "Created message successfully");
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			createMessageMutation.mutate(value);
		},
		defaultValues: {
			body_md: "",
		},
	});

	return (
		<Grid container>
			<Grid size={{ xs: 12 }}>
				<TabContext value={value}>
					<Stack spacing={2}>
						<ContainerWithHeader title={"Report Overview"} iconLeft={<ReportIcon />}>
							<Box
								sx={{
									borderBottom: 1,
									borderColor: "divider",
									backgroundColor: theme.palette.background.paper,
								}}
							>
								<TabList
									variant={"fullWidth"}
									onChange={handleChange}
									aria-label="ReportCreatePage detail tabs"
								>
									<Tab label="Description" icon={<DescriptionIcon />} iconPosition={"start"} />
									{hasPermission(PermissionLevel.Moderator) && (
										<Tab
											sx={{ height: 20 }}
											label={`Chat Logs`}
											icon={<MessageIcon />}
											iconPosition={"start"}
										/>
									)}
									{hasPermission(PermissionLevel.Moderator) && (
										<Tab label={`Connections`} icon={<LanIcon />} iconPosition={"start"} />
									)}
								</TabList>
							</Box>

							<TabPanel value={value} index={0}>
								{report && (
									<Box minHeight={300}>
										<MarkDownRenderer body_md={report.description} />
									</Box>
								)}
							</TabPanel>

							<TabPanel value={value} index={1}>
								<Box minHeight={300}>
									<ChatTable
										messages={chat ?? []}
										isLoading={isLoadingChat}
										manualPaging={false}
										pagination={chatPagination}
										setPagination={setChatPagination}
									/>
									<PaginatorLocal
										onRowsChange={(rows) => {
											setChatPagination((prev) => {
												return { ...prev, pageSize: rows };
											});
										}}
										onPageChange={(page) => {
											setChatPagination((prev) => {
												return { ...prev, pageIndex: page };
											});
										}}
										count={chat?.length ?? 0}
										rows={chatPagination.pageSize}
										page={chatPagination.pageIndex}
									/>
								</Box>
							</TabPanel>
							<TabPanel value={value} index={2}>
								<Box minHeight={300}>
									<IPHistoryTable
										connections={connections ?? { data: [], count: 0 }}
										isLoading={isLoadingConnections}
										manualPaging={false}
										pagination={connectionPagination}
										setPagination={setConnectionPagination}
									/>
									<PaginatorLocal
										onRowsChange={(rows) => {
											setConnectionPagination((prev) => {
												return { ...prev, pageSize: rows };
											});
										}}
										onPageChange={(page) => {
											setConnectionPagination((prev) => {
												return { ...prev, pageIndex: page };
											});
										}}
										count={connections?.data?.length ?? 0}
										rows={connectionPagination.pageSize}
										page={connectionPagination.pageIndex}
									/>
								</Box>
							</TabPanel>
						</ContainerWithHeader>
						{report.demo_id > 0 && (
							<ContainerWithHeaderAndButtons
								title={`Demo Details: ${report.demo_name}`}
								iconLeft={<VideocamIcon />}
								buttons={[
									<Button
										variant={"contained"}
										fullWidth
										key={"demo_download"}
										startIcon={<FileDownloadIcon />}
										component={Link}
										href={`/asset/${report.demo_id}`}
										color={"success"}
									>
										Download
									</Button>,
								]}
							>
								<Grid container padding={2}>
									{/*<Grid size={{ xs: 4 }}>*/}
									{/*    <Typography>Map:&nbsp;{report.demo.map_name}</Typography>*/}
									{/*</Grid>*/}
									{/*<Grid size={{ xs: 4 }}>*/}
									{/*    <Typography>Server:&nbsp;{report.demo.server_name_short}</Typography>*/}
									{/*</Grid>*/}
									<Grid size={{ xs: 2 }}>
										<Typography>Tick:&nbsp;{report.demo_tick}</Typography>
									</Grid>
									<Grid size={{ xs: 2 }}>
										<Typography>ID:&nbsp;{report.demo_id}</Typography>
									</Grid>
								</Grid>
							</ContainerWithHeaderAndButtons>
						)}

						{report.person_message_id > 0 && (
							<ContainerWithHeader title={"Message Context"} iconLeft={<QuickreplyIcon />}>
								<PlayerMessageContext playerMessageId={report.person_message_id} padding={4} />
							</ContainerWithHeader>
						)}

						{hasPermission(PermissionLevel.Moderator) && (
							<SourceBansList steam_id={report.source_id} is_reporter={true} />
						)}

						{hasPermission(PermissionLevel.Moderator) && (
							<SourceBansList steam_id={report.target_id} is_reporter={false} />
						)}

						{!isLoadingMessages &&
							messages &&
							messages.map((m) => (
								<ReportMessageView message={m} key={`report-msg-${m.report_message_id}`} />
							))}
						<Paper elevation={1}>
							<form
								onSubmit={async (e) => {
									e.preventDefault();
									e.stopPropagation();
									await form.handleSubmit();
								}}
							>
								<Grid container spacing={2} padding={1}>
									<Grid size={{ xs: 12 }}>
										<form.AppField
											name={"body_md"}
											validators={{
												onChange: z.string().min(2),
											}}
											children={(field) => {
												return <field.MarkdownField label={"Message"} />;
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
						</Paper>
					</Stack>
				</TabContext>
			</Grid>
		</Grid>
	);
};
