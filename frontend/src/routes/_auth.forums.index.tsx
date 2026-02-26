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
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo } from "react";
import { apiGetForumOverview } from "../api/forum.ts";
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
import { Title } from "../component/Title";
import { VCenterBox } from "../component/VCenterBox.tsx";
import { useAppInfoCtx } from "../contexts/AppInfoCtx.ts";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { ForumCategory } from "../schema/forum.ts";
import { PermissionLevel } from "../schema/people.ts";
import { logErr } from "../util/errors.ts";
import { avatarHashToURL, humanCount } from "../util/text.tsx";
import { renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_auth/forums/")({
	component: ForumOverview,
});

const CategoryBlock = ({ category }: { category: ForumCategory }) => {
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
		return hasPermission(PermissionLevel.Moderator)
			? [
					<Button
						size={"small"}
						variant={"contained"}
						color={"warning"}
						key={`cat-edit-${category.forum_category_id}`}
						startIcon={<ConstructionIcon />}
						onClick={onEdit}
					>
						Edit
					</Button>,
				]
			: [];
	}, [category.forum_category_id, hasPermission, onEdit]);

	return (
		<ContainerWithHeaderAndButtons title={category.title} iconLeft={<CategoryIcon />} buttons={buttons}>
			<Title>Forums</Title>
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
							key={`forum-${f.forum_id}`}
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
												<ForumRowLink label={f.title} to={`/forums/${f.forum_id}`} />
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
											{humanCount(f.count_threads)}
										</Typography>
									</Stack>
									<Stack>
										<Typography variant={"body2"}>Messages</Typography>
										<Typography variant={"body1"} align={"center"}>
											{humanCount(f.count_messages)}
										</Typography>
									</Stack>
								</Stack>
							</Grid>
							<Grid size={{ xs: 5 }}>
								{f.recent_forum_thread_id && f.recent_forum_thread_id > 0 ? (
									<Stack direction={"row"} spacing={2}>
										<VCenteredElement
											icon={
												<Avatar
													alt={f.recent_personaname}
													src={avatarHashToURL(f.recent_avatarhash, "medium")}
												/>
											}
										/>
										<Stack>
											<ForumRowLink
												variant={"body1"}
												label={f.recent_forum_title ?? ""}
												to={`/forums/thread/${f.recent_forum_thread_id}`}
											/>

											<Stack direction={"row"} spacing={1}>
												<AccessTimeIcon />
												<VCenterBox>
													<Typography variant={"body2"}>
														{renderDateTime(f.recent_created_on ?? new Date())}
													</Typography>
												</VCenterBox>
												<Person2 />
												<VCenterBox>
													<Typography
														sx={{
															color: (theme) => theme.palette.text.secondary,
														}}
														component={RouterLink}
														to={`/profile/${f.recent_source_id}`}
														variant={"body2"}
													>
														{f.recent_personaname}
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
	const { appInfo } = useAppInfoCtx();
	const { hasPermission } = useAuth();
	const { data: overview, isLoading } = useQuery({
		queryKey: ["forumOverview"],
		queryFn: async () => {
			return await apiGetForumOverview();
		},
	});

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
				categories: overview?.categories ?? [],
			});
			sendFlash("success", "Created new forum successfully");
		} catch (e) {
			logErr(e);
		}
	}, [overview?.categories, sendFlash]);

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Typography variant={"h2"}>{appInfo.site_name} community</Typography>
			</Grid>
			<Grid size={{ xs: 12, md: 9 }}>
				<Stack spacing={2}>
					{isLoading ? (
						<LoadingPlaceholder />
					) : (
						overview?.categories
							.filter((c) => c.forums.length > 0)
							.map((cat) => {
								return <CategoryBlock category={cat} key={`category-${cat.forum_category_id}`} />;
							})
					)}
				</Stack>
			</Grid>
			<Grid size={{ xs: 12, md: 3 }}>
				<Stack spacing={2}>
					<ForumRecentMessageActivity />
					<ForumRecentUserActivity />
					{hasPermission(PermissionLevel.Moderator) && (
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
