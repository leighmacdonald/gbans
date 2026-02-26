import NiceModal, { useModal } from "@ebay/nice-modal-react";
import AccessTimeIcon from "@mui/icons-material/AccessTime";
import ConstructionIcon from "@mui/icons-material/Construction";
import LockIcon from "@mui/icons-material/Lock";
import Person2Icon from "@mui/icons-material/Person2";
import { IconButton } from "@mui/material";
import Box from "@mui/material/Box";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import { apiCreateThreadReply, apiDeleteMessage, apiGetThread, apiGetThreadMessages } from "../api/forum.ts";
import { mdEditorRef } from "../component/form/field/MarkdownField.tsx";
import { ThreadMessageContainer } from "../component/forum/ForumThreadMessageContainer.tsx";
import { PaginatorLocal } from "../component/forum/PaginatorLocal.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { ConfirmationModal } from "../component/modal/ConfirmationModal.tsx";
import { ForumThreadEditorModal } from "../component/modal/ForumThreadEditorModal.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { VCenterBox } from "../component/VCenterBox.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { ForumMessage, ForumThread } from "../schema/forum.ts";
import { PermissionLevel } from "../schema/people.ts";
import { logErr } from "../util/errors.ts";
import { useScrollToLocation } from "../util/history.ts";
import { commonTableSearchSchema, RowsPerPage } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";
import { LoginPage } from "./_guest.login.index.tsx";

const forumThreadSearchSchema = commonTableSearchSchema;

export const Route = createFileRoute("/_auth/forums/thread/$forum_thread_id")({
	component: ForumThreadPage,
	validateSearch: (search) => forumThreadSearchSchema.parse(search),
	loader: async ({ context, params }) => {
		const thread = await context.queryClient.fetchQuery({
			queryKey: ["forumThread", { forum_thread_id: Number(params.forum_thread_id) }],
			queryFn: async () => {
				return await apiGetThread(Number(params.forum_thread_id));
			},
		});

		return { thread, appInfo: context.appInfo };
	},
	head: ({ loaderData }) => ({
		meta: [
			{ name: "description", content: loaderData?.thread?.title },
			{ title: `Thread - ${loaderData?.appInfo.site_name}` },
		],
	}),
});

function ForumThreadPage() {
	const { hasPermission, permissionLevel } = useAuth();
	const { forum_thread_id } = Route.useParams();
	const { thread, appInfo } = Route.useLoaderData();
	const { pageIndex } = Route.useSearch();
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const confirmModal = useModal(ConfirmationModal);
	const navigate = useNavigate();
	const theme = useTheme();

	const { data: messages, isLoading: isLoadingMessages } = useQuery({
		queryKey: ["threadMessages", { forum_thread_id }],
		queryFn: async () => {
			return await apiGetThreadMessages({
				forum_thread_id: Number(forum_thread_id),
			});
		},
	});

	const [pagination, setPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const onSave = async (message: ForumMessage) => {
		queryClient.setQueryData(
			["threadMessages", { forum_thread_id }],
			messages?.map((m) => {
				return message.forum_message_id === m.forum_message_id ? message : m;
			}),
		);
	};

	useScrollToLocation();

	const firstPostID = useMemo(() => {
		if (Number(pageIndex) > 1 || !messages) {
			return -1;
		}
		if (messages.length > 0) {
			return messages[0].forum_message_id;
		}
		return -1;
	}, [messages, pageIndex]);

	const onEditThread = useCallback(async () => {
		try {
			const newThread = (await NiceModal.show(ForumThreadEditorModal, {
				thread: thread,
			})) as ForumThread;

			if (newThread.forum_thread_id > 0) {
				queryClient.setQueryData(["forumThread", { forum_thread_id: Number(forum_thread_id) }], newThread);
			} else {
				await navigate({ to: "/forums" });
			}
		} catch (e) {
			logErr(e);
		}
	}, [forum_thread_id, navigate, queryClient, thread]);

	const deleteMessageMutation = useMutation({
		mutationFn: async ({ message }: { message: ForumMessage }) => {
			await apiDeleteMessage(message.forum_message_id);
		},
		onSuccess: async (_, variables) => {
			const newMessages = (messages ?? []).filter(
				(m) => m.forum_message_id !== variables.message.forum_message_id,
			);
			queryClient.setQueryData(["threadMessages", { forum_thread_id }], newMessages);
			sendFlash("success", `Messages deleted successfully: #${variables.message.forum_message_id}`);
			if (firstPostID === variables.message.forum_message_id) {
				await navigate({ to: "/forums" });
			}
		},
		onError: sendError,
	});

	const onMessageDeleted = useCallback(
		async (message: ForumMessage) => {
			const isFirstMessage = firstPostID === message.forum_message_id;
			const confirmed = await confirmModal.show({
				title: "Delete Post?",
				children: (
					<Box>
						{isFirstMessage && (
							<Typography variant={"body1"} fontWeight={700} color={theme.palette.error.dark}>
								Please be aware that by deleting the first post in the thread, this will result in the
								deletion of the <i>entire thread</i>.
							</Typography>
						)}
						<Typography variant={"body1"}>This action cannot be undone.</Typography>
					</Box>
				),
			});

			if (!confirmed) {
				return;
			}

			deleteMessageMutation.mutate({ message });
		},
		[confirmModal, deleteMessageMutation, firstPostID, theme.palette.error.dark],
	);

	const createMessageMutation = useMutation({
		mutationFn: async ({ body_md }: { body_md: string }) => {
			return await apiCreateThreadReply(Number(forum_thread_id), body_md);
		},
		onSuccess: (message) => {
			const newMessages = [...(messages ?? []), message];
			queryClient.setQueryData(["threadMessages", { forum_thread_id }], newMessages);
			mdEditorRef.current?.setMarkdown("");
			form.reset();
			sendFlash("success", `New message created (#${message.forum_message_id})`);
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

	const replyContainer = useMemo(() => {
		if (permissionLevel() === PermissionLevel.Guest) {
			return <LoginPage />;
		} else if (thread?.forum_thread_id && !thread?.locked) {
			return (
				<Paper>
					<Box padding={2}>
						<form
							onSubmit={async (e) => {
								e.preventDefault();
								e.stopPropagation();
								await form.handleSubmit();
							}}
						>
							<Grid container spacing={2} justifyItems={"flex-end"}>
								<Grid size={{ xs: 12 }}>
									<form.AppField
										name={"body_md"}
										children={(field) => {
											return <field.MarkdownField label={"Message"} minHeight={400} />;
										}}
									/>
								</Grid>
								<Grid size={{ xs: 4 }}>
									<form.AppForm>
										<ButtonGroup>
											<form.ResetButton />
											<form.SubmitButton />
										</ButtonGroup>
									</form.AppForm>
								</Grid>
							</Grid>
						</form>
					</Box>
				</Paper>
			);
		} else {
			return null;
		}
	}, [permissionLevel, thread?.forum_thread_id, thread?.locked, form]);

	return (
		<Stack spacing={1}>
			<Stack direction={"row"}>
				{hasPermission(PermissionLevel.Moderator) && (
					<IconButton color={"warning"} onClick={onEditThread}>
						<ConstructionIcon fontSize={"small"} />
					</IconButton>
				)}
				<Typography variant={"h3"}>{thread?.title}</Typography>
			</Stack>
			<Stack direction={"row"} spacing={1}>
				<Person2Icon />
				<VCenterBox>
					<Typography
						variant={"body2"}
						component={RouterLink}
						sx={{ color: (theme) => theme.palette.text.primary }}
						to={`/profile/${thread?.source_id}`}
					>
						{thread?.personaname ?? ""}
					</Typography>
				</VCenterBox>
				<AccessTimeIcon />
				<VCenterBox>
					<Typography variant={"body2"}>{renderDateTime(thread?.created_on ?? new Date())}</Typography>
				</VCenterBox>
			</Stack>
			{isLoadingMessages ? (
				<LoadingPlaceholder />
			) : (
				(messages ?? []).map((m) => (
					<ThreadMessageContainer
						assetURL={appInfo.asset_url}
						onSave={onSave}
						message={m}
						key={`thread-message-id-${m.forum_message_id}`}
						onDelete={onMessageDeleted}
						isFirstMessage={firstPostID === m.forum_message_id}
					/>
				))
			)}

			<PaginatorLocal
				onRowsChange={(rows) => {
					setPagination((prev) => {
						return { ...prev, pageSize: rows };
					});
				}}
				onPageChange={(page) => {
					setPagination((prev) => {
						return { ...prev, pageIndex: page };
					});
				}}
				count={(messages ?? []).length}
				rows={pagination.pageSize}
				page={pagination.pageIndex}
			/>
			{thread?.locked && (
				<Paper>
					<Typography variant={"h4"} textAlign={"center"} padding={1}>
						<LockIcon /> Thread Locked
					</Typography>
				</Paper>
			)}
			{replyContainer}
		</Stack>
	);
}
