import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import DocumentScannerIcon from "@mui/icons-material/DocumentScanner";
import InfoIcon from "@mui/icons-material/Info";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemText from "@mui/material/ListItemText";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import { AppealMessageView } from "../component/AppealMessageView.tsx";
import { BanModPanel } from "../component/BanModPanel.tsx";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { MarkdownField } from "../component/form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { ProfileInfoBox } from "../component/ProfileInfoBox.tsx";
import { SourceBansList } from "../component/SourceBansList.tsx";
import { SteamIDList } from "../component/SteamIDList.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { deleteAppealMessage, messages, reply } from "../rpc/ban/v1/appeal-AppealService_connectquery.ts";
import { AppealState, BanReason, BanType } from "../rpc/ban/v1/ban_pb.ts";
import { get } from "../rpc/ban/v1/ban-BanService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { renderTimeDistance, renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_auth/ban/$banId")({
	component: BanPage,
	head: ({ match }) => ({
		meta: [{ name: "description", content: match.context.appInfo.siteDescription }, match.context.title(`Ban`)],
	}),
});

function BanPage() {
	const { permissionLevel, profile } = useAuth();
	const { banId } = Route.useParams();
	const { sendFlash } = useUserFlashCtx();
	const { appInfo } = Route.useRouteContext();
	const queryClient = useQueryClient();

	const { data: banData, isLoading: isLoadingBan } = useQuery(get, { banId: Number(banId) });
	const { data: banMessages, isLoading: isLoadingMessages } = useQuery(messages, { banId: Number(banId) });
	// const banMutation = useMutation();

	const deleteMessageMutation = useMutation(deleteAppealMessage, {
		onSuccess: async (_, req) => {
			queryClient.setQueryData(
				["banMessages", { banId }],
				banMessages?.messages?.filter((m) => {
					return m.banMessageId !== req.banMessageId;
				}),
			);
			sendFlash("success", "Deleted message successfully");
		},
		onError: async (error: Error) => {
			sendFlash("error", error.message);
		},
	});

	const canPost = useMemo(() => {
		return (
			permissionLevel() >= Privilege.MODERATOR ||
			(banData?.ban?.appealState === AppealState.OPEN_UNSPECIFIED && banData.ban?.targetId === profile.steamId)
		);
	}, [banData?.ban, permissionLevel, profile.steamId]);

	const onDelete = useCallback(
		async (banMessageId: bigint) => {
			return await deleteMessageMutation.mutateAsync({ banMessageId });
		},
		[deleteMessageMutation.mutateAsync],
	);

	const replyMutation = useMutation(reply);
	// 	mutationFn: async (values: { body_md: string }) => {
	// 		if (!ban) {
	// 			return;
	// 		}
	// 		const ac = new AbortController();
	// 		const msg = await apiCreateBanMessage(ban?.ban_id, values.body_md, ac.signal);
	//
	// 		queryClient.setQueryData(["banMessages", { ban_id: ban.ban_id }], [...(banMessages?.messages ?? []), msg]);
	// 		sendFlash("success", "Created a message successfully");
	// 		mdEditorRef.current?.setMarkdown("");
	// 		form.reset();
	// 	},
	// });

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			replyMutation.mutate({ banId: Number(banId), bodyMd: value.bodyMd });
		},
		defaultValues: {
			bodyMd: "",
		},
	});

	if (isLoadingMessages || isLoadingBan || !banData?.ban) {
		return;
	}

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 8 }}>
				<Stack spacing={2}>
					{canPost && (banMessages?.messages ?? []).length === 0 && (
						<ContainerWithHeader title={`Ban Appeal #${banData.ban.banId}`}>
							<Typography variant={"body2"} padding={2} textAlign={"center"}>
								You can start the appeal process by replying on this form.
							</Typography>
						</ContainerWithHeader>
					)}

					{permissionLevel() >= Privilege.MODERATOR && (
						<SourceBansList steamId={banData.ban.sourceId} isReporter={true} />
					)}

					{permissionLevel() >= Privilege.MODERATOR && (
						<SourceBansList steamId={banData.ban.targetId} isReporter={false} />
					)}

					{(banMessages?.messages ?? []).map((m) => (
						<AppealMessageView
							onDelete={onDelete}
							message={m}
							key={`ban-appeal-msg-${m.banMessageId}`}
							assetURL={appInfo.assetUrl}
						/>
					))}
					{canPost && (
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
											children={(props) => {
												return (
													<MarkdownField
														{...props}
														value={props.state.value}
														label={"Message"}
													/>
												);
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
					)}
					{!canPost && (
						<Paper elevation={1}>
							<Typography variant={"body2"} padding={2} textAlign={"center"}>
								The ban appeal is closed: {AppealState[banData.ban?.appealState]}
							</Typography>
						</Paper>
					)}
				</Stack>
			</Grid>
			<Grid size={{ xs: 4 }}>
				<Stack spacing={2}>
					<ProfileInfoBox steamId={banData.ban.targetId} />

					<ContainerWithHeader title={"Ban Details"} iconLeft={<InfoIcon />}>
						<List dense={true}>
							<ListItem>
								<ListItemText primary={"Reason"} secondary={BanReason[banData.ban.reason]} />
							</ListItem>
							<ListItem>
								<ListItemText primary={"Ban Type"} secondary={BanType[banData.ban.banType]} />
							</ListItem>
							{banData.ban.reasonText !== "" && (
								<ListItem>
									<ListItemText primary={"Reason (Custom)"} secondary={banData.ban.reasonText} />
								</ListItem>
							)}

							<ListItem>
								<ListItemText
									primary={"Created At"}
									secondary={renderTimestamp(banData.ban.createdOn)}
								/>
							</ListItem>
							<ListItem>
								<ListItemText
									primary={"Expires At"}
									secondary={renderTimestamp(banData.ban.validUntil)}
								/>
							</ListItem>
							<ListItem>
								<ListItemText
									primary={"Expires"}
									secondary={renderTimeDistance(
										banData.ban.validUntil ? timestampDate(banData.ban.validUntil) : new Date(),
									)}
								/>
							</ListItem>
							{permissionLevel() >= Privilege.MODERATOR && (
								<ListItem>
									<ListItemText primary={"Author"} secondary={banData.ban.sourceId.toString()} />
								</ListItem>
							)}
						</List>
					</ContainerWithHeader>

					<SteamIDList steam_id={banData.ban.targetId} />

					{permissionLevel() >= Privilege.MODERATOR && banData.ban.note !== "" && (
						<ContainerWithHeader title={"Mod Notes"} iconLeft={<DocumentScannerIcon />}>
							<MarkDownRenderer body_md={banData.ban.note} assetURL={appInfo.assetUrl} />
						</ContainerWithHeader>
					)}

					{permissionLevel() >= Privilege.MODERATOR && <BanModPanel banId={banData.ban.banId} />}
				</Stack>
			</Grid>
		</Grid>
	);
}
