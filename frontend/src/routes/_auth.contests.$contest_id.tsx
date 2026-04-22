import NiceModal from "@ebay/nice-modal-react";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import InfoIcon from "@mui/icons-material/Info";
import PageviewIcon from "@mui/icons-material/Pageview";
import PublishIcon from "@mui/icons-material/Publish";
import ThumbDownIcon from "@mui/icons-material/ThumbDown";
import ThumbUpIcon from "@mui/icons-material/ThumbUp";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { format } from "date-fns";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import { isAfter } from "date-fns/fp";
import { useCallback, useMemo } from "react";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { InfoBar } from "../component/InfoBar.tsx";
import { LoadingSpinner } from "../component/LoadingSpinner.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { AssetViewer } from "../component/modal/AssetViewer.tsx";
import { ContestEntryDeleteModal } from "../component/modal/ContestEntryDeleteModal.tsx";
import { ContestEntryModal } from "../component/modal/ContestEntryModal.tsx";
import { PageNotFound } from "../component/PageNotFound.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { VCenterBox } from "../component/VCenterBox.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { logErr } from "../util/errors.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { humanFileSize } from "../util/text.tsx";
import { useQuery, useSuspenseQuery } from "@connectrpc/connect-query";
import { contest, entries } from "../rpc/contest/v1/contest-Service_connectquery.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import type { Asset } from "../rpc/asset/v1/asset_pb.ts";

export const Route = createFileRoute("/_auth/contests/$contest_id")({
	component: Contest,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.contestsEnabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Contests" }, match.context.title("Contests")],
	}),
});

function Contest() {
	const { contest_id } = Route.useParams();

	const { appInfo } = Route.useRouteContext();
	const { hasPermission, profile } = useAuth();
	const { sendFlash } = useUserFlashCtx();

	const { data } = useSuspenseQuery(contest, { contestId: contest_id });

	const onEnter = useCallback(async (contest_id: string) => {
		try {
			await NiceModal.show(ContestEntryModal, { contest_id });
		} catch (e) {
			logErr(e);
		}
	}, []);

	const { data: contestEntries, isLoading: entriesLoading } = useQuery(entries, {});

	const showEntries = useMemo(() => {
		return (data.contest && !data.contest?.hideSubmissions) || hasPermission(Privilege.MODERATOR);
	}, [contest, hasPermission]);

	const vote = useCallback(
		async (contest_entry_id: string, up_vote: boolean) => {
			if (!data.contest?.contestId) {
				return;
			}
			try {
				const ac = new AbortController();
				await apiContestEntryVote(contest?.contest_id, contest_entry_id, up_vote, ac.signal);
				updateEntries();
			} catch (e) {
				logErr(e);
			}
		},
		[contest?.contest_id, updateEntries],
	);

	const onViewAsset = useCallback(async (asset: Asset) => {
		await NiceModal.show(AssetViewer, asset);
	}, []);

	const onDeleteEntry = useCallback(
		async (contest_entry_id: string) => {
			try {
				await NiceModal.show(ContestEntryDeleteModal, {
					contest_entry_id,
				});
				setEntries((prevState) => {
					return prevState.filter((v) => v.contest_entry_id !== contest_entry_id);
				});
				sendFlash("success", `Entry deleted successfully`);
			} catch (e) {
				sendFlash("error", `Failed to delete entry: ${e}`);
			}
		},
		[sendFlash],
	);

	if (!contest_id) {
		return <PageNotFound />;
	}

	return (
		<Grid container spacing={3}>
			<Grid size={{ xs: 8 }}>
				<ContainerWithHeader title={`Contest: ${data.contest?.title}`} iconLeft={<EmojiEventsIcon />}>
					<Grid container>
						<Grid size={{ xs: 12 }} minHeight={400}>
							<Typography variant={"body1"} padding={2}>
								{data.contest?.description}
							</Typography>
						</Grid>
					</Grid>
				</ContainerWithHeader>
			</Grid>
			<Grid size={{ xs: 4 }}>
				<ContainerWithHeader title={`Contest Details`} iconLeft={<InfoIcon />}>
					<Stack spacing={2}>
						<InfoBar
							title={"Starting Date"}
							value={format(data.contest?.dateStart, "dd/MM/yy H:m")}
							align={"right"}
						/>

						<InfoBar
							title={"Ending Date"}
							value={format(data.contest?.dateEnd, "dd/MM/yy H:m")}
							align={"right"}
						/>

						<InfoBar
							title={"Remaining"}
							value={
								isAfter(data.contest?.dateEnd, new Date())
									? "Expired"
									: formatDistanceToNowStrict(data.contest?.dateEnd)
							}
							align={"right"}
						/>

						<InfoBar title={"Max Entries Per User"} value={data.contest?.maxSubmissions} align={"right"} />

						<InfoBar title={"Total Entries"} value={contestEntries?.entries.length} align={"right"} />
						<Button
							fullWidth
							variant={"contained"}
							color={"success"}
							disabled={isAfter(data.contest?.dateEnd, new Date())}
							startIcon={<PublishIcon />}
							onClick={async () => {
								await onEnter(data.contest?.contestId as string);
							}}
						>
							Submit Entry
						</Button>
					</Stack>
				</ContainerWithHeader>
			</Grid>
			{entriesLoading ? (
				<LoadingSpinner />
			) : (
				<>
					{!showEntries && (
						<Grid size={{ xs: 12 }}>
							<Paper>
								<Typography variant={"subtitle1"} align={"center"} padding={4}>
									Entries from other contestants are hidden.
								</Typography>
							</Paper>
						</Grid>
					)}
					<Grid size={{ xs: 12 }}>
						<Stack spacing={2}>
							{contestEntries?.entries
								.filter((e) => showEntries || e.steamId === profile.steamId)
								.map((entry) => {
									return (
										<Stack key={entry.contestEntryId}>
											<Paper elevation={2}>
												<Grid container>
													<Grid size={{ xs: 8 }} padding={2}>
														<Typography variant={"subtitle1"}>Description</Typography>
														<MarkDownRenderer
															assetURL={appInfo.assetUrl}
															body_md={
																entry.description !== ""
																	? entry.description
																	: "No description provided"
															}
														/>
													</Grid>
													<Grid size={{ xs: 4 }} padding={2}>
														<PersonCell
															steam_id={entry.steamId}
															personaname={entry.personaName}
															avatar_hash={entry.avatarHash}
														/>
														<Typography variant={"subtitle1"}>File Details</Typography>
														<Typography variant={"body2"}>{entry.asset?.name}</Typography>
														<Typography variant={"body2"}>
															{entry.asset?.mimeType}
														</Typography>
														<Typography variant={"body2"}>
															{humanFileSize(Number(entry.asset?.size))}
														</Typography>
														<ButtonGroup fullWidth>
															<Button
																disabled={
																	!(
																		hasPermission(Privilege.MODERATOR) ||
																		profile.steamId === entry.steamId
																	)
																}
																color={"error"}
																variant={"contained"}
																onClick={async () => {
																	await onDeleteEntry(entry.contestEntryId);
																}}
															>
																Delete
															</Button>

															{mediaType(entry.asset.mimeType) !== MediaTypes.other ? (
																<Button
																	startIcon={<PageviewIcon />}
																	fullWidth
																	variant={"contained"}
																	color={"success"}
																	onClick={async () => {
																		await onViewAsset(entry.asset);
																	}}
																>
																	View
																</Button>
															) : (
																<Button>Download</Button>
															)}
														</ButtonGroup>
													</Grid>
												</Grid>
											</Paper>
											<Stack direction={"row"} padding={1} spacing={2}>
												<ButtonGroup
													disabled={
														!data.contest?.voting ||
														isAfter(data.contest?.dateEnd, new Date())
													}
												>
													<Button
														size={"small"}
														variant={"contained"}
														startIcon={<ThumbUpIcon />}
														color={"success"}
														onClick={async () => {
															await vote(entry.contestEntryId, true);
														}}
													>
														{entry.votesUp}
													</Button>
													<Button
														size={"small"}
														variant={"contained"}
														startIcon={<ThumbDownIcon />}
														color={"error"}
														disabled={!data.contest?.downVotes}
														onClick={async () => {
															await vote(entry.contestEntryId, false);
														}}
													>
														{entry.votesDown}
													</Button>
												</ButtonGroup>
												<VCenterBox>
													<Typography variant={"caption"}>
														{`Updated: ${format(entry.updatedOn, "dd/MM/yy H:m")}`}
													</Typography>
												</VCenterBox>
											</Stack>
										</Stack>
									);
								})}
						</Stack>
					</Grid>
				</>
			)}
		</Grid>
	);
}
