import { useQuery } from "@connectrpc/connect-query";
import NiceModal, { useModal } from "@ebay/nice-modal-react";
import BuildIcon from "@mui/icons-material/Build";
import LockIcon from "@mui/icons-material/Lock";
import MessageIcon from "@mui/icons-material/Message";
import PostAddIcon from "@mui/icons-material/PostAdd";
import PushPinIcon from "@mui/icons-material/PushPin";
import Avatar from "@mui/material/Avatar";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useCallback, useMemo, useState } from "react";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { ForumRowLink } from "../component/forum/ForumRowLink.tsx";
import { PaginatorLocal } from "../component/forum/PaginatorLocal.tsx";
import { VCenteredElement } from "../component/Heading.tsx";
import { ForumForumEditorModal } from "../component/modal/ForumForumEditorModal.tsx";
import { ForumThreadCreatorModal } from "../component/modal/ForumThreadCreatorModal.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { VCenterBox } from "../component/VCenterBox.tsx";
import { AppError } from "../error.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { Thread, ThreadWithSource } from "../rpc/forum/v1/forum_pb.ts";
import { forum, threads } from "../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { logErr } from "../util/errors.ts";
import { avatarHashToURL } from "../util/strings.ts";
import { RowsPerPage } from "../util/table.ts";
import { renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_auth/forums/$forumId")({
	component: ForumPage,
	// loader: async ({ context, abortController, params }) => {
	// 	const { forum_id } = params;
	// 	const forumQueryOpts = {
	// 		queryKey: forumQueryKey(forum_id),
	// 		queryFn: async () => {
	// 			return await apiForum(Number(forum_id), abortController.signal);
	// 		},
	// 	};
	// 	const forum = await context.queryClient.fetchQuery(forumQueryOpts);
	// 	const threadsQueryOpts: FetchQueryOptions<ForumThread[]> = {
	// 		queryKey: forumThreadsQueryKey(forum_id),
	// 		queryFn: async () => {
	// 			return (await apiGetThreads({ forum_id: Number(forum_id) }, abortController.signal)) ?? [];
	// 		},
	// 	};
	//
	// 	const threads = await context.queryClient.fetchQuery(threadsQueryOpts);
	// 	return { forum, threads };
	// },
	head: ({ match }) => ({
		meta: [
			// { name: "description", content: loaderData?.forum.description },
			match.context.title("Forum"),
		],
	}),
	errorComponent: ({ error }) => {
		if (error instanceof AppError) {
			return <ErrorDetails error={error} />;
		}
		return <div>Oops</div>;
	},
});

function ForumPage() {
	//const queryClient = useQueryClient();
	const { forumId } = Route.useParams();
	const { data: forumData, isLoading: isLoadingForum } = useQuery(forum, { forumId: Number(forumId) });
	const { data: threadsData, isLoading: isLoadingThreads } = useQuery(threads, { forumId: Number(forumId) });
	const modalCreate = useModal(ForumThreadCreatorModal);
	const { hasPermission } = useAuth();
	const { sendFlash } = useUserFlashCtx();
	const navigate = useNavigate();

	const [pagination, setPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const onNewThread = useCallback(async () => {
		try {
			const thread = (await modalCreate.show({ forum: forumData?.forum })) as Thread;
			await navigate({ to: `/forums/thread/${thread.forumThreadId}` });
			await modalCreate.hide();
		} catch (e) {
			sendFlash("error", `${e}`);
		}
	}, [modalCreate, navigate, sendFlash, forumData?.forum]);

	const onEditForum = useCallback(async () => {
		try {
			await NiceModal.show(ForumForumEditorModal, {
				forum: forumData?.forum,
			});
			// queryClient.setQueryData(forumQueryKey(forum_id), editedForum);
		} catch (e) {
			logErr(e);
		}
	}, [forumData?.forum]);

	const headerButtons = useMemo(() => {
		const buttons = [];

		if (hasPermission(Privilege.MODERATOR)) {
			buttons.push(
				<Button
					startIcon={<BuildIcon />}
					color={"warning"}
					variant={"contained"}
					size={"small"}
					key={"btn-edit-forum"}
					onClick={onEditForum}
				>
					Edit
				</Button>,
			);
		}
		buttons.push(
			<Button
				disabled={!hasPermission(Privilege.GUEST)}
				variant={"contained"}
				color={"success"}
				size={"small"}
				onClick={onNewThread}
				startIcon={<PostAddIcon />}
				key={"btn-new-post"}
			>
				New Post
			</Button>,
		);
		return [<ButtonGroup key={"forum-header-buttons"}>{buttons}</ButtonGroup>];
	}, [hasPermission, onEditForum, onNewThread]);

	if (isLoadingForum || isLoadingThreads || !forumData?.forum) {
		return;
	}

	return (
		<ContainerWithHeaderAndButtons title={forumData.forum.title} iconLeft={<MessageIcon />} buttons={headerButtons}>
			<Stack spacing={2}>
				{threadsData?.threads.map((t) => {
					return <ForumThreadRow thread={t} key={`ft-${t.thread?.forumThreadId}`} />;
				})}
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
					count={threadsData?.threads.length ?? 0}
					rows={pagination.pageSize}
					page={pagination.pageIndex}
				/>
			</Stack>
		</ContainerWithHeaderAndButtons>
	);
}

const ForumThreadRow = ({ thread }: { thread: ThreadWithSource }) => {
	return (
		<Grid
			container
			spacing={1}
			sx={{
				"&:hover": {
					backgroundColor: (theme) => theme.palette.background.default,
				},
			}}
		>
			<Grid size={{ xs: 12, md: 8 }}>
				<Stack direction={"row"} spacing={2}>
					<VCenteredElement
						icon={<Avatar alt={thread.personaName} src={avatarHashToURL(thread.avatarHash, "medium")} />}
					/>
					<Stack>
						<Stack direction={"row"} justifyContent="space-between">
							<ForumRowLink
								label={String(thread.thread?.title)}
								to={`/forums/thread/${thread.thread?.forumThreadId}`}
							/>
						</Stack>
						<Stack direction={"row"} spacing={1}>
							{thread.thread?.sticky && (
								<VCenterBox>
									<PushPinIcon fontSize={"small"} />
								</VCenterBox>
							)}
							{thread.thread?.locked && (
								<VCenterBox>
									<LockIcon fontSize={"small"} />
								</VCenterBox>
							)}
							<Typography
								variant={"body2"}
								component={RouterLink}
								to={`/profile/${thread.thread?.sourceId}`}
								sx={{
									color: (theme) => theme.palette.text.secondary,
									textDecoration: "none",
									"&:hover": { textDecoration: "underline" },
								}}
							>
								{thread.personaName}
							</Typography>
						</Stack>
					</Stack>
				</Stack>
			</Grid>
			<Grid size={{ xs: 6, md: 1 }}>
				<Grid container justifyContent="space-between">
					<Grid size={{ xs: 6 }}>
						<Typography variant={"body1"} align={"left"}>
							Replies:
						</Typography>
					</Grid>
					<Grid size={{ xs: 6 }} alignContent={"flex-end"}>
						<Typography variant={"body1"} align={"right"}>
							{thread.thread?.replies}
						</Typography>
					</Grid>
					<Grid size={{ xs: 6 }}>
						<Typography variant={"body2"}>Views:</Typography>
					</Grid>
					<Grid size={{ xs: 6 }} alignContent={"flex-end"}>
						<Typography variant={"body2"} align={"right"}>
							{thread.thread?.views}
						</Typography>
					</Grid>
				</Grid>
			</Grid>
			<Grid size={{ xs: 6, md: 3 }}>
				{thread.recentForumMessageId && thread.recentForumMessageId > 0 ? (
					<Stack direction={"row"} justifyContent={"end"} spacing={1}>
						<Stack>
							<Typography
								variant={"body2"}
								align={"right"}
								fontWeight={700}
								sx={{
									color: (theme) => theme.palette.text.primary,
									textDecoration: "none",
								}}
								component={RouterLink}
								to={`/forums/thread/${thread.thread?.forumThreadId}#${thread.recentForumMessageId}`}
							>
								{renderTimestamp(thread.recentCreatedOn)}
							</Typography>
							<Typography
								align={"right"}
								variant={"body2"}
								sx={{
									color: (theme) => theme.palette.text.secondary,
									textDecoration: "none",
								}}
								component={RouterLink}
								to={`/profile/${thread.recentSteamId}`}
							>
								{thread.recentPersonaName}
							</Typography>
						</Stack>
						<VCenterBox>
							<Avatar
								sx={{ height: "32px", width: "32px" }}
								alt={avatarHashToURL(thread.recentAvatarHash, "small")}
							/>
						</VCenterBox>
					</Stack>
				) : null}
			</Grid>
		</Grid>
	);
};
