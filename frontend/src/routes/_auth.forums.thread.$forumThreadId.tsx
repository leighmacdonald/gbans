import { useMutation, useQuery } from "@connectrpc/connect-query";
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
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Navigate, useNavigate } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
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
import type { Message, Thread } from "../rpc/forum/v1/forum_pb.ts";
import {
	thread,
	threadMessageDelete,
	threadMessages,
	threadReplyCreate,
} from "../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { logErr } from "../util/errors.ts";
import { useScrollToLocation } from "../util/history.ts";
import { commonTableSearchSchema, RowsPerPage } from "../util/table.ts";
import { renderTimestamp } from "../util/time.ts";

const forumThreadSearchSchema = commonTableSearchSchema;

export const Route = createFileRoute("/_auth/forums/thread/$forumThreadId")({
	component: ForumThreadPage,
	validateSearch: (search) => forumThreadSearchSchema.parse(search),
	head: ({ match }) => ({
		meta: [match.context.title("Thread")],
	}),
});

function ForumThreadPage() {
	const { hasPermission, permissionLevel } = useAuth();
	const { forumThreadId } = Route.useParams();
	const { appInfo } = Route.useRouteContext();
	const { pageIndex } = Route.useSearch();
	const { sendFlash, sendError } = useUserFlashCtx();
	const queryClient = useQueryClient();
	const confirmModal = useModal(ConfirmationModal);
	const navigate = useNavigate();
	const theme = useTheme();

	const { data: threadData, isLoading: isLoadingThread } = useQuery(thread, { forumThreadId: Number(forumThreadId) });

	const { data: messagesData, isLoading: isLoadingMessages } = useQuery(threadMessages, {
		forumThreadId: Number(forumThreadId),
	});

	const [pagination, setPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const onSave = async (message: Message) => {
		queryClient.setQueryData(
			["threadMessages", { forum_thread_id: forumThreadId }],
			messagesData?.messages?.map((m) => {
				return message.forumMessageId === m.forumMessageId ? message : m;
			}),
		);
	};

	useScrollToLocation();

	const firstPostID = useMemo(() => {
		if (Number(pageIndex) > 1 || !messagesData?.messages) {
			return -1;
		}
		if (messagesData.messages.length > 0) {
			return messagesData.messages[0].forumMessageId;
		}
		return -1;
	}, [messagesData, pageIndex]);

	const onEditThread = useCallback(async () => {
		try {
			const newThread = (await NiceModal.show(ForumThreadEditorModal, {
				thread: threadData?.thread,
			})) as Thread;

			if (newThread.forumThreadId > 0) {
				queryClient.setQueryData(["forumThread", { forum_thread_id: Number(forumThreadId) }], newThread);
			} else {
				await navigate({ to: "/forums" });
			}
		} catch (e) {
			logErr(e);
		}
	}, [forumThreadId, navigate, queryClient, threadData?.thread]);

	const deleteMessageMutation = useMutation(threadMessageDelete, {
		onSuccess: async (_, variables) => {
			const newMessages = (messagesData?.messages ?? []).filter(
				(m) => m.forumMessageId !== variables.forumMessageId,
			);
			queryClient.setQueryData(["threadMessages", { forum_thread_id: forumThreadId }], newMessages);
			sendFlash("success", `Messages deleted successfully: #${variables.forumMessageId}`);
			if (firstPostID === variables.forumMessageId) {
				await navigate({ to: "/forums" });
			}
		},
		onError: sendError,
	});

	const onMessageDeleted = useCallback(
		async (message: Message) => {
			const isFirstMessage = firstPostID === message.forumMessageId;
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

			deleteMessageMutation.mutate({ forumMessageId: message.forumMessageId });
		},
		[confirmModal, deleteMessageMutation, firstPostID, theme.palette.error.dark],
	);

	const createMessageMutation = useMutation(threadReplyCreate, {
		onSuccess: (message) => {
			const newMessages = [...(messagesData?.messages ?? []), message];
			queryClient.setQueryData(["threadMessages", { forum_thread_id: forumThreadId }], newMessages);
			mdEditorRef.current?.setMarkdown("");
			form.reset();
			sendFlash("success", `New message created (#${message.message?.forumMessageId})`);
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			createMessageMutation.mutate(value);
		},
		defaultValues: {
			bodyMd: "",
		},
	});

	const replyContainer = useMemo(() => {
		if (permissionLevel() === Privilege.GUEST) {
			return (
				<Navigate
					to={"/login"}
					search={{
						redirect: window.location.pathname + window.location.search,
					}}
				/>
			);
		} else if (threadData?.thread?.forumThreadId && !threadData.thread.locked) {
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
										name={"bodyMd"}
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
	}, [permissionLevel, threadData?.thread?.forumThreadId, threadData?.thread?.locked, form]);

	if (isLoadingThread || isLoadingThread) {
		return;
	}

	return (
		<Stack spacing={1}>
			<Stack direction={"row"}>
				{hasPermission(Privilege.MODERATOR) && (
					<IconButton color={"warning"} onClick={onEditThread}>
						<ConstructionIcon fontSize={"small"} />
					</IconButton>
				)}
				<Typography variant={"h3"}>{threadData?.thread?.title}</Typography>
			</Stack>
			<Stack direction={"row"} spacing={1}>
				<Person2Icon />
				<VCenterBox>
					<Typography
						variant={"body2"}
						component={RouterLink}
						sx={{ color: (theme) => theme.palette.text.primary }}
						to={`/profile/${threadData?.thread?.sourceId}`}
					>
						{/*{thread?.personaname ?? "FIXME"}*/}
						{"FIXME"}
					</Typography>
				</VCenterBox>
				<AccessTimeIcon />
				<VCenterBox>
					<Typography variant={"body2"}>{renderTimestamp(threadData?.thread?.createdOn)}</Typography>
				</VCenterBox>
			</Stack>
			{isLoadingMessages ? (
				<LoadingPlaceholder />
			) : (
				(messagesData?.messages ?? []).map((m) => (
					<ThreadMessageContainer
						assetURL={appInfo.assetUrl}
						onSave={onSave}
						message={m}
						key={`thread-message-id-${m.forumMessageId}`}
						onDelete={onMessageDeleted}
						isFirstMessage={firstPostID === m.forumMessageId}
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
				count={(messagesData?.messages ?? []).length}
				rows={pagination.pageSize}
				page={pagination.pageIndex}
			/>
			{threadData?.thread?.locked && (
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
