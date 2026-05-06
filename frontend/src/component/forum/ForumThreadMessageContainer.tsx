import { timestampDate } from "@bufbuild/protobuf/wkt";
import { useMutation } from "@connectrpc/connect-query";
import DeleteForeverIcon from "@mui/icons-material/DeleteForever";
import EditIcon from "@mui/icons-material/Edit";
import { Divider, IconButton } from "@mui/material";
import Box from "@mui/material/Box";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { isAfter } from "date-fns/fp";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useAuth } from "../../hooks/useAuth.ts";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Message } from "../../rpc/forum/v1/forum_pb.ts";
import { threadReplyEdit } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { Privilege } from "../../rpc/person/v1/privilege_pb.ts";
import { avatarHashToURL } from "../../util/strings.ts";
import { renderTimestamp } from "../../util/time.ts";
import { mdEditorRef } from "../form/field/MarkdownField.tsx";
import { MarkDownRenderer } from "../MarkdownRenderer.tsx";
import RouterLink from "../RouterLink.tsx";
import { ForumAvatar } from "./ForumAvatar.tsx";
import { ForumRowLink } from "./ForumRowLink.tsx";

export const ThreadMessageContainer = ({
	message,
	onDelete,
	onSave,
	assetURL,
}: {
	message: Message;
	onDelete: (message: Message) => Promise<void>;
	onSave: (message: Message) => Promise<void>;
	isFirstMessage: boolean;
	assetURL: string;
}) => {
	const [edit, setEdit] = useState(false);
	const { hasPermission, profile } = useAuth();
	const { sendError } = useUserFlashCtx();
	const theme = useTheme();

	const editable = useMemo(() => {
		return profile.steamId === message.sourceId || hasPermission(Privilege.MODERATOR);
	}, [hasPermission, message.sourceId, profile.steamId]);

	const mutation = useMutation(threadReplyEdit, {
		onSuccess: async (data) => {
			if (!data.message) {
				return;
			}
			mdEditorRef.current?.setMarkdown("");
			setEdit(false);
			await onSave(data.message);
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate({
				forumMessageId: message.forumMessageId,
				bodyMd: value.bodyMd ?? "",
			});
		},
		defaultValues: {
			bodyMd: message.bodyMd,
		},
	});

	return (
		<Paper elevation={1} id={`${message.forumMessageId}`}>
			<Grid container>
				<Grid size={{ xs: 2 }} padding={2} sx={{ backgroundColor: theme.palette.background.paper }}>
					<Stack alignItems={"center"}>
						<ForumAvatar
							alt={message.personaName}
							online={message.online}
							src={avatarHashToURL(message.avatarHash, "medium")}
						/>

						<ForumRowLink
							label={message.personaName}
							to={`/profile/${message.sourceId}`}
							align={"center"}
						/>
						<Typography variant={"subtitle1"} align={"center"}>
							{Privilege[message.permissionLevel]}
						</Typography>
					</Stack>
				</Grid>
				<Grid size={{ xs: 10 }}>
					{edit ? (
						<form
							onSubmit={async (e) => {
								e.preventDefault();
								e.stopPropagation();
								await form.handleSubmit();
							}}
						>
							<Stack padding={1}>
								<form.AppField
									name={"bodyMd"}
									validators={{
										onChange: z.string().min(4),
									}}
									children={(field) => {
										return <field.MarkdownField label={"Message (Markdown)"} />;
									}}
								/>
								<form.AppForm>
									<ButtonGroup>
										<form.ResetButton />
										<form.SubmitButton />
									</ButtonGroup>
								</form.AppForm>
							</Stack>
						</form>
					) : (
						<Box>
							<Grid container direction="row" borderBottom={(theme) => theme.palette.divider}>
								<Grid size={{ xs: 6 }}>
									<Stack direction={"row"}>
										<Typography variant={"body2"} padding={1}>
											{renderTimestamp(message.createdOn)}
										</Typography>
										{message.updatedOn &&
											message.createdOn &&
											isAfter(
												timestampDate(message.createdOn),
												timestampDate(message.updatedOn),
											) && (
												<Typography variant={"body2"} padding={1}>
													{`Edited: ${renderTimestamp(message.updatedOn)}`}
												</Typography>
											)}
									</Stack>
								</Grid>
								<Grid size={{ xs: 6 }}>
									<Stack direction="row" justifyContent="end">
										<IconButton
											color={"error"}
											onClick={async () => {
												await onDelete(message);
											}}
										>
											<DeleteForeverIcon />
										</IconButton>
										{editable && (
											<IconButton
												title={"Edit Post"}
												color={"secondary"}
												size={"small"}
												onClick={() => {
													setEdit(true);
												}}
											>
												<EditIcon />
											</IconButton>
										)}
										<Typography
											padding={1}
											component={RouterLink}
											variant={"body2"}
											to={`#${message.forumMessageId}`}
											textAlign={"right"}
											sx={{ color: (theme) => theme.palette.text.primary }}
										>
											{`#${message.forumMessageId}`}
										</Typography>
									</Stack>
								</Grid>
							</Grid>
							<Grid size={{ xs: 12 }} padding={1}>
								<MarkDownRenderer body_md={message.bodyMd} assetURL={assetURL} />

								{message.signature !== "" && (
									<>
										<Divider />
										<MarkDownRenderer body_md={message.signature} assetURL={assetURL} />
									</>
								)}
							</Grid>
						</Box>
					)}
				</Grid>
			</Grid>
		</Paper>
	);
};
