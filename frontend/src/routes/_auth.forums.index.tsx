import { useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import { Person2 } from "@mui/icons-material";
import AccessTimeIcon from "@mui/icons-material/AccessTime";
import CategoryIcon from "@mui/icons-material/Category";
import ChatIcon from "@mui/icons-material/Chat";
import ConstructionIcon from "@mui/icons-material/Construction";
import Avatar from "@mui/material/Avatar";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { ForumRecentMessageActivity } from "../component/forum/ForumRecentmessageActivity.tsx";
import { ForumRecentUserActivity } from "../component/forum/ForumRecentUserActivity.tsx";
import { ForumRowLink } from "../component/forum/ForumRowLink.tsx";
import { VCenteredElement } from "../component/Heading.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { ForumCategoryEditorModal } from "../component/modal/ForumCategoryEditorModal.tsx";
import { ForumForumEditorModal } from "../component/modal/ForumForumEditorModal.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { VCenterBox } from "../component/VCenterBox.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { Category } from "../rpc/forum/v1/forum_pb.ts";
import { overview } from "../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { logErr } from "../util/errors.ts";
import { avatarHashToURL, humanCount } from "../util/text.tsx";
import { renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_auth/forums/")({
	component: ForumOverview,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Forums" }, match.context.title("Forums")],
	}),
});

const CategoryBlock = ({ category }: { category: Category }) => {
	const { hasPermission } = useAuth();

	const onEdit = useCallback(async () => {
		try {
			await NiceModal.show(ForumCategoryEditorModal, {
				category,
			});
		} catch (e) {
			logErr(e);
		}
	}, [category]);

	const buttons = useMemo(() => {
		return hasPermission(Privilege.MODERATOR)
			? [
					<Button
						size={"small"}
						variant={"contained"}
						color={"warning"}
						key={`cat-edit-${category.forumCategoryId}`}
						startIcon={<ConstructionIcon />}
						onClick={onEdit}
					>
						Edit
					</Button>,
				]
			: [];
	}, [category.forumCategoryId, hasPermission, onEdit]);

	return (
		<ContainerWithHeaderAndButtons title={category.title} iconLeft={<CategoryIcon />} buttons={buttons}>
			<Stack
				spacing={1}
				sx={{
					overflow: "hidden",
					textOverflow: "ellipsis",
					whiteSpace: "nowrap",
					width: "100%",
				}}
			>
				{category.description !== "" && <Typography>{category.description}</Typography>}
				{category.forums.map((f) => {
					return (
						<Grid
							container
							key={`forum-${f.forumId}`}
							spacing={1}
							sx={{
								"&:hover": {
									backgroundColor: (theme) => theme.palette.background.default,
								},
							}}
						>
							<Grid size={{ xs: 5 }} margin={0}>
								<VCenterBox justify={"left"}>
									<Stack direction={"row"} spacing={1}>
										<VCenteredElement icon={<ChatIcon />} />

										<Stack>
											<VCenterBox>
												<ForumRowLink label={f.title} to={`/forums/${f.forumId}`} />
											</VCenterBox>
											<VCenterBox>
												<Typography variant={"body2"}>{f.description}</Typography>
											</VCenterBox>
										</Stack>
									</Stack>
								</VCenterBox>
							</Grid>
							<Grid size={{ xs: 2 }}>
								<Stack direction={"row"} spacing={1}>
									<Stack>
										<Typography variant={"body2"} align={"left"}>
											Threads
										</Typography>
										<Typography variant={"body1"} align={"center"}>
											{humanCount(f.countThreads)}
										</Typography>
									</Stack>
									<Stack>
										<Typography variant={"body2"}>Messages</Typography>
										<Typography variant={"body1"} align={"center"}>
											{humanCount(f.countMessages)}
										</Typography>
									</Stack>
								</Stack>
							</Grid>
							<Grid size={{ xs: 5 }}>
								{f.recentForumThreadId && f.recentForumThreadId > 0 ? (
									<Stack direction={"row"} spacing={2}>
										<VCenteredElement
											icon={
												<Avatar
													alt={f.recentPersonaName}
													src={avatarHashToURL(f.recentAvatarHash, "medium")}
												/>
											}
										/>
										<Stack>
											<ForumRowLink
												variant={"body1"}
												label={f.recentForumTitle ?? ""}
												to={`/forums/thread/${f.recentForumThreadId}`}
											/>

											<Stack direction={"row"} spacing={1}>
												<AccessTimeIcon />
												<VCenterBox>
													<Typography variant={"body2"}>
														{renderTimestamp(f.recentCreatedOn)}
													</Typography>
												</VCenterBox>
												<Person2 />
												<VCenterBox>
													<Typography
														sx={{
															color: (theme) => theme.palette.text.secondary,
														}}
														component={RouterLink}
														to={`/profile/${f.recentSourceId}`}
														variant={"body2"}
													>
														{f.recentPersonaName}
													</Typography>
												</VCenterBox>
											</Stack>
										</Stack>
									</Stack>
								) : null}
							</Grid>
						</Grid>
					);
				})}
			</Stack>
		</ContainerWithHeaderAndButtons>
	);
};

function ForumOverview() {
	const { sendFlash } = useUserFlashCtx();
	const { appInfo } = Route.useRouteContext();
	const { hasPermission } = useAuth();
	const { data, isLoading } = useQuery(overview);

	const onNewCategory = useCallback(async () => {
		try {
			await NiceModal.show(ForumCategoryEditorModal, {});
			sendFlash("success", "Created new category successfully");
		} catch (e) {
			logErr(e);
		}
	}, [sendFlash]);

	const onNewForum = useCallback(async () => {
		try {
			await NiceModal.show(ForumForumEditorModal, {
				categories: data?.categories ?? [],
			});
			sendFlash("success", "Created new forum successfully");
		} catch (e) {
			logErr(e);
		}
	}, [data?.categories, sendFlash]);

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Typography variant={"h2"}>{appInfo.siteName} community</Typography>
			</Grid>
			<Grid size={{ xs: 12, md: 9 }}>
				<Stack spacing={2}>
					{isLoading ? (
						<LoadingPlaceholder />
					) : (
						data?.categories
							.filter((c) => c.forums.length > 0)
							.map((cat) => {
								return <CategoryBlock category={cat} key={`category-${cat.forumCategoryId}`} />;
							})
					)}
				</Stack>
			</Grid>
			<Grid size={{ xs: 12, md: 3 }}>
				<Stack spacing={2}>
					<ForumRecentMessageActivity />
					<ForumRecentUserActivity />
					{hasPermission(Privilege.MODERATOR) && (
						<ContainerWithHeader title={"Mod Tools"} iconLeft={<ConstructionIcon />}>
							<Button onClick={onNewCategory} variant={"contained"} color={"success"}>
								New Category
							</Button>
							<Button onClick={onNewForum} variant={"contained"} color={"success"}>
								New Forum
							</Button>
						</ContainerWithHeader>
					)}
				</Stack>
			</Grid>
		</Grid>
	);
}
