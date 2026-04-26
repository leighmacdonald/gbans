import { useMutation, useQuery } from "@connectrpc/connect-query";
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
import { type JSX, type SyntheticEvent, useState } from "react";
import { z } from "zod/v4";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { ReportWithAuthorValid } from "../rpc/ban/v1/report_pb.ts";
import { reportMessageCreate, reportMessages } from "../rpc/ban/v1/report-ReportService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { ContainerWithHeaderAndButtons } from "./ContainerWithHeaderAndButtons.tsx";
import { mdEditorRef } from "./form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "./MarkdownRenderer";
import { PlayerMessageContext } from "./PlayerMessageContext";
import { ReportMessageView } from "./ReportMessageView";
import { SourceBansList } from "./SourceBansList";
import { TabPanel } from "./TabPanel";
import { ChatTable } from "./table/ChatTable.tsx";
import { IPHistoryTable } from "./table/IPHistoryTable.tsx";

export const ReportViewComponent = ({
	report,
	assetURL,
}: {
	report: ReportWithAuthorValid;
	assetURL: string;
}): JSX.Element => {
	const theme = useTheme();
	const { sendFlash, sendError } = useUserFlashCtx();
	const [value, setValue] = useState<number>(0);
	const { hasPermission } = useAuth();

	const { data: messageData, isLoading: isLoadingMessages } = useQuery(reportMessages, {
		reportId: report.report.reportId,
	});

	const handleChange = (_: SyntheticEvent, newValue: number) => {
		setValue(newValue);
	};

	const createMessageMutation = useMutation(reportMessageCreate, {
		onSuccess: () => {
			// FIXME
			// queryClient.setQueryData(reportMessagesQueryOptions(report.report_id).queryKey, [
			// 	...(messageData?.messages ?? []),
			// 	message,
			// ]);
			mdEditorRef.current?.setMarkdown("");
			form.reset();
			sendFlash("success", "Created message successfully");
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			return await createMessageMutation.mutateAsync({ ...value, reportId: report.report.reportId });
		},
		defaultValues: {
			bodyMd: "",
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
									{hasPermission(Privilege.MODERATOR) && (
										<Tab
											sx={{ height: 20 }}
											label={`Chat Logs`}
											icon={<MessageIcon />}
											iconPosition={"start"}
										/>
									)}
									{hasPermission(Privilege.MODERATOR) && (
										<Tab label={`Connections`} icon={<LanIcon />} iconPosition={"start"} />
									)}
								</TabList>
							</Box>

							<TabPanel value={value} index={0}>
								{report && (
									<Box minHeight={300}>
										<MarkDownRenderer body_md={report.report.description} assetURL={assetURL} />
									</Box>
								)}
							</TabPanel>

							<TabPanel value={value} index={1}>
								<Box minHeight={300}>
									<ChatTable steamId={report.report.targetId} />
								</Box>
							</TabPanel>
							<TabPanel value={value} index={2}>
								<Box minHeight={300}>
									<IPHistoryTable steamId={report.report.targetId} />
								</Box>
							</TabPanel>
						</ContainerWithHeader>
						{report.report?.demoId > 0 && (
							<ContainerWithHeaderAndButtons
								title={`Demo Details: ${report.report.demoId}`}
								iconLeft={<VideocamIcon />}
								buttons={[
									<Button
										variant={"contained"}
										fullWidth
										key={"demo_download"}
										startIcon={<FileDownloadIcon />}
										component={Link}
										href={`/asset/${report.report.demoId}`}
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
										<Typography>Tick:&nbsp;{report.report.demoTick}</Typography>
									</Grid>
									<Grid size={{ xs: 2 }}>
										<Typography>ID:&nbsp;{report.report.demoId}</Typography>
									</Grid>
								</Grid>
							</ContainerWithHeaderAndButtons>
						)}

						{report.report.personMessageId > 0 && (
							<ContainerWithHeader title={"Message Context"} iconLeft={<QuickreplyIcon />}>
								<PlayerMessageContext playerMessageId={report.report.personMessageId} padding={4} />
							</ContainerWithHeader>
						)}

						{hasPermission(Privilege.MODERATOR) && (
							<SourceBansList steamId={report.report.sourceId} isReporter={true} />
						)}

						{hasPermission(Privilege.MODERATOR) && (
							<SourceBansList steamId={report.report.targetId} isReporter={false} />
						)}

						{!isLoadingMessages &&
							messageData?.messages &&
							messageData.messages.map((m) => (
								<ReportMessageView
									message={m}
									key={`report-msg-${m.reportMessageId}`}
									assetURL={assetURL}
								/>
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
											name={"bodyMd"}
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
