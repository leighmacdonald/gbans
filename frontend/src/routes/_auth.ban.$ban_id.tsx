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
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import { z } from "zod/v4";
import {
	apiCreateBanMessage,
	apiDeleteBanMessage,
	apiGetBanMessages,
	apiGetBanSteam,
	appealStateString,
	banTypeString,
} from "../api";
import { AppealMessageView } from "../component/AppealMessageView.tsx";
import { BanModPanel } from "../component/BanModPanel.tsx";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { MarkdownField, mdEditorRef } from "../component/form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { ProfileInfoBox } from "../component/ProfileInfoBox.tsx";
import { SourceBansList } from "../component/SourceBansList.tsx";
import { SteamIDList } from "../component/SteamIDList.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { AppError, ErrorCode } from "../error.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { logErr } from "../util/errors.ts";
import { renderDateTime, renderTimeDistance } from "../util/time.ts";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import { deleteAppealMessage, messages } from "../rpc/ban/v1/appeal-AppealService_connectquery.ts";
import { useQueryClient } from "@tanstack/react-query";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { AppealState } from "../rpc/ban/v1/ban_pb.ts";

export const Route = createFileRoute("/_auth/ban/$ban_id")({
	component: BanPage,

	loader: async ({ context, abortController, params }) => {
		const { ban_id } = params;
		const ban = await context.queryClient.fetchQuery({
			queryKey: ["ban", { ban_id }],
			queryFn: async () => {
				const ban = await apiGetBanSteam(Number(ban_id), true, abortController.signal);
				if (!ban) {
					throw new AppError(ErrorCode.NotFound);
				}
				return ban;
			},
		});
		return { ban };
	},
	errorComponent: (e) => {
		return <ErrorDetails error={e.error} />;
	},
	head: ({ loaderData, match }) => ({
		meta: [
			{ name: "description", content: match.context.appInfo.siteDescription },
			match.context.title(`Ban #${loaderData?.ban.ban_id}`),
		],
	}),
});

function BanPage() {
	const { permissionLevel, profile } = useAuth();
	const { ban } = Route.useLoaderData();
	const { sendFlash } = useUserFlashCtx();
	const { appInfo } = Route.useRouteContext();
	const queryClient = useQueryClient();

	const { data: banMessages } = useQuery(messages, { banId: ban.id });
	const banMutation = useMutation();
	const deleteMessageMutation = useMutation(deleteAppealMessage, {
		onSuccess: async (_, req) => {
			queryClient.setQueryData(
				["banMessages", { ban_id: ban.ban_id }],
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
			(ban?.appeal_state === AppealState.OPEN_UNSPECIFIED && ban?.target_id === profile.steamId)
		);
	}, [ban?.appeal_state, ban?.target_id, permissionLevel, profile.steamId]);

	const onDelete = useCallback(async (banMessageId: bigint) => {
		return await deleteMessageMutation.mutateAsync({ banMessageId });
	}, []);

	const mutation = useMutation({
		mutationKey: ["banSteam"],
		mutationFn: async (values: { body_md: string }) => {
			if (!ban) {
				return;
			}
			const ac = new AbortController();
			const msg = await apiCreateBanMessage(ban?.ban_id, values.body_md, ac.signal);

			queryClient.setQueryData(["banMessages", { ban_id: ban.ban_id }], [...(messages ?? []), msg]);
			sendFlash("success", "Created a message successfully");
			mdEditorRef.current?.setMarkdown("");
			form.reset();
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate(value);
		},
		defaultValues: {
			body_md: "",
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 8 }}>
				<Stack spacing={2}>
					{canPost && (messages ?? []).length === 0 && (
						<ContainerWithHeader title={`Ban Appeal #${ban.ban_id}`}>
							<Typography variant={"body2"} padding={2} textAlign={"center"}>
								You can start the appeal process by replying on this form.
							</Typography>
						</ContainerWithHeader>
					)}

					{permissionLevel() >= PermissionLevel.Moderator && (
						<SourceBansList steam_id={ban?.source_id} is_reporter={true} />
					)}

					{permissionLevel() >= PermissionLevel.Moderator && (
						<SourceBansList steam_id={ban?.target_id} is_reporter={false} />
					)}

					{(messages ?? []).map((m) => (
						<AppealMessageView
							onDelete={onDelete}
							message={m}
							key={`ban-appeal-msg-${m.ban_message_id}`}
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
											name={"body_md"}
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
								The ban appeal is closed: {appealStateString(ban.appeal_state)}
							</Typography>
						</Paper>
					)}
				</Stack>
			</Grid>
			<Grid size={{ xs: 4 }}>
				<Stack spacing={2}>
					<ProfileInfoBox steam_id={ban.target_id} />

					<ContainerWithHeader title={"Ban Details"} iconLeft={<InfoIcon />}>
						<List dense={true}>
							<ListItem>
								<ListItemText primary={"Reason"} secondary={BanReasons[ban.reason]} />
							</ListItem>
							<ListItem>
								<ListItemText primary={"Ban Type"} secondary={banTypeString(ban.ban_type)} />
							</ListItem>
							{ban.reason_text !== "" && (
								<ListItem>
									<ListItemText primary={"Reason (Custom)"} secondary={ban.reason_text} />
								</ListItem>
							)}

							<ListItem>
								<ListItemText primary={"Created At"} secondary={renderDateTime(ban.created_on)} />
							</ListItem>
							<ListItem>
								<ListItemText
									primary={"Expires At"}
									secondary={renderDateTime(ban.valid_until as Date)}
								/>
							</ListItem>
							<ListItem>
								<ListItemText
									primary={"Expires"}
									secondary={renderTimeDistance(ban.valid_until as Date)}
								/>
							</ListItem>
							{permissionLevel() >= PermissionLevel.Moderator && (
								<ListItem>
									<ListItemText primary={"Author"} secondary={ban.source_id.toString()} />
								</ListItem>
							)}
						</List>
					</ContainerWithHeader>

					<SteamIDList steam_id={ban?.target_id} />

					{permissionLevel() >= PermissionLevel.Moderator && ban.note !== "" && (
						<ContainerWithHeader title={"Mod Notes"} iconLeft={<DocumentScannerIcon />}>
							<MarkDownRenderer body_md={ban.note} assetURL={appInfo.assetUrl} />
						</ContainerWithHeader>
					)}

					{permissionLevel() >= PermissionLevel.Moderator && <BanModPanel ban_id={ban.ban_id} />}
				</Stack>
			</Grid>
		</Grid>
	);
}
