import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import CalendarMonthIcon from "@mui/icons-material/CalendarMonth";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import ErrorIcon from "@mui/icons-material/Error";
import ExpandLessIcon from "@mui/icons-material/ExpandLess";
import ExpandMoreIcon from "@mui/icons-material/ExpandMore";
import GroupIcon from "@mui/icons-material/Group";
import LinkIcon from "@mui/icons-material/Link";
import OpenInNewIcon from "@mui/icons-material/OpenInNew";
import SportsEsportsIcon from "@mui/icons-material/SportsEsports";
import VerifiedUserIcon from "@mui/icons-material/VerifiedUser";
import Alert from "@mui/material/Alert";
import Avatar from "@mui/material/Avatar";
import Button from "@mui/material/Button";
import Collapse from "@mui/material/Collapse";
import Grid from "@mui/material/Grid";
import List from "@mui/material/List";
import ListItem from "@mui/material/ListItem";
import ListItemButton from "@mui/material/ListItemButton";
import ListItemIcon from "@mui/material/ListItemIcon";
import ListItemText from "@mui/material/ListItemText";
import Skeleton from "@mui/material/Skeleton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { format } from "date-fns";
import { formatDistanceToNowStrict } from "date-fns/formatDistanceToNowStrict";
import prettyMilliseconds from "pretty-ms";
import { useMemo, useState } from "react";
import { ButtonLink } from "../component/ButtonLink.tsx";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { SteamIDList } from "../component/SteamIDList.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { profile } from "../rpc/person/v1/person-PersonService_connectquery.ts";
import type { PlayerMatchHistory } from "../rpc/stats/v1/stats_pb.ts";
import { matchesWithPlayer } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { createExternalLinks } from "../util/history.ts";
import { avatarHashToURL } from "../util/strings.ts";
import { isValidSteamDate } from "../util/time.ts";

export const Route = createFileRoute("/_guest/profile/$steamId")({
	component: ProfilePage,
	head: () => ({
		meta: [{ name: "description", content: "Player Profile" }],
	}),
});

const recentMatchLimit = 5;
const friendPreviewLimit = 8;

function StandingRow({ clean, title, detail }: { clean: boolean; title: string; detail: string }) {
	return (
		<ListItem>
			<ListItemIcon sx={{ minWidth: 40 }}>
				{clean ? <VerifiedUserIcon color={"success"} /> : <ErrorIcon color={"error"} />}
			</ListItemIcon>
			<ListItemText primary={title} secondary={detail} />
		</ListItem>
	);
}

function relativeTime(when: Timestamp | undefined): string | null {
	if (!when) {
		return null;
	}
	try {
		return formatDistanceToNowStrict(timestampDate(when), { addSuffix: true });
	} catch {
		return null;
	}
}

function MatchRow({ match }: { match: PlayerMatchHistory }) {
	const playedAgo = relativeTime(match.startTime);
	return (
		<ListItemButton
			component={TextLink}
			to={"/match/$matchId"}
			params={{ matchId: match.matchId }}
			sx={{ borderRadius: 1 }}
		>
			<ListItemIcon sx={{ minWidth: 40 }}>
				<EmojiEventsIcon color={match.isWinner ? "warning" : "disabled"} />
			</ListItemIcon>
			<ListItemText
				primary={`${match.mapName} · ${match.scoreRed}–${match.scoreBlu}`}
				secondary={[match.serverName, prettyMilliseconds(Number(match.duration)), playedAgo]
					.filter(Boolean)
					.join(" · ")}
			/>
		</ListItemButton>
	);
}

function RecentMatches({ steamId }: { steamId: string }) {
	const { data, isLoading, isError } = useQuery(matchesWithPlayer, { steamId });

	const recent = useMemo(() => (data?.matches ?? []).slice(0, recentMatchLimit), [data]);
	const total = data?.matches?.length ?? 0;

	if (isError) {
		return null;
	}

	return (
		<ContainerWithHeader title={`Recent Matches${total > 0 ? ` (${total})` : ""}`} iconLeft={<SportsEsportsIcon />}>
			{isLoading ? (
				<List disablePadding>
					{Array.from({ length: recentMatchLimit }).map((_, i) => (
						// biome-ignore lint/suspicious/noArrayIndexKey: placeholder rows
						<ListItem key={i}>
							<Skeleton variant={"circular"} width={24} height={24} sx={{ mr: 2 }} />
							<Skeleton variant={"text"} width={"70%"} />
						</ListItem>
					))}
				</List>
			) : recent.length === 0 ? (
				<Typography variant={"body2"} color={"textSecondary"} sx={{ padding: 2 }}>
					No recorded matches yet.
				</Typography>
			) : (
				<>
					<List disablePadding>
						{recent.map((match) => (
							<MatchRow key={match.matchId} match={match} />
						))}
					</List>
					<ButtonLink
						to={"/matches/$steamId"}
						params={{ steamId }}
						variant={"text"}
						sx={{ alignSelf: "flex-start", ml: 1, mb: 1 }}
					>
						View Full History
					</ButtonLink>
				</>
			)}
		</ContainerWithHeader>
	);
}

function FriendsCard({ steamIds }: { steamIds: string[] }) {
	const [expanded, setExpanded] = useState(false);

	if (steamIds.length === 0) {
		return null;
	}

	const visible = expanded ? steamIds : steamIds.slice(0, friendPreviewLimit);

	return (
		<ContainerWithHeader title={`Friends (${steamIds.length})`} iconLeft={<GroupIcon />}>
			<List dense disablePadding>
				{visible.map((sid) => (
					<ListItem key={sid} disablePadding>
						<PersonCell steamId={sid} />
					</ListItem>
				))}
			</List>
			{steamIds.length > friendPreviewLimit && (
				<>
					<Collapse in={expanded} timeout={"auto"} unmountOnExit>
						<List dense disablePadding>
							{steamIds.slice(friendPreviewLimit).map((sid) => (
								<ListItem key={sid} disablePadding>
									<PersonCell steamId={sid} />
								</ListItem>
							))}
						</List>
					</Collapse>
					<Button
						variant={"text"}
						size={"small"}
						startIcon={expanded ? <ExpandLessIcon /> : <ExpandMoreIcon />}
						onClick={() => setExpanded((v) => !v)}
						sx={{ alignSelf: "flex-start", ml: 1, mb: 1 }}
					>
						{expanded ? "Show Less" : `Show All ${steamIds.length}`}
					</Button>
				</>
			)}
		</ContainerWithHeader>
	);
}

function ProfilePage() {
	const { steamId } = Route.useParams();
	const { data } = useQuery(profile, { steamId });

	const friendIds = useMemo(() => {
		const friends = data?.profile?.friends ?? [];
		return friends
			.filter((f) => !f.removedOn)
			.map((f) => String(f.steamId))
			.filter((id, i, all) => id !== "" && all.indexOf(id) === i);
	}, [data]);

	if (!data?.profile?.player) {
		return <LoadingPlaceholder />;
	}

	const player = data.profile.player;
	const statsVisible = !data.profile.settings?.statsHidden;
	const vacBans = Number(player.vacBans);
	const gameBans = Number(player.gameBans);
	const banId = Number(player.banId);
	const createdOn =
		player.timeCreated && isValidSteamDate(timestampDate(player.timeCreated))
			? timestampDate(player.timeCreated)
			: null;

	return (
		<Grid container spacing={3}>
			<Grid size={{ xs: 12, md: 8 }}>
				<Stack spacing={3}>
					<ContainerWithHeader title={"Profile"} padding={3}>
						<Stack direction={{ xs: "column", sm: "row" }} spacing={3} alignItems={{ sm: "center" }}>
							<Avatar
								src={avatarHashToURL(player.avatarHash)}
								alt={player.name}
								sx={{ width: 120, height: 120, alignSelf: { xs: "center", sm: "flex-start" } }}
							/>
							<Stack spacing={1} sx={{ minWidth: 0 }}>
								<Typography variant={"h4"} sx={{ wordBreak: "break-word" }}>
									{player.name}
								</Typography>
								{createdOn && (
									<Stack direction={"row"} spacing={1} alignItems={"center"}>
										<CalendarMonthIcon fontSize={"small"} color={"action"} />
										<Typography variant={"body2"} color={"textSecondary"}>
											Member since {format(createdOn, "yyyy-MM-dd")} ·{" "}
											{formatDistanceToNowStrict(createdOn, { addSuffix: true })}
										</Typography>
									</Stack>
								)}
							</Stack>
						</Stack>
						{banId > 0 && (
							<Alert
								severity={"warning"}
								action={
									<ButtonLink
										to={`/ban/$banId`}
										params={{ banId: String(banId) }}
										size={"small"}
										color={"inherit"}
									>
										View Ban
									</ButtonLink>
								}
							>
								This player is currently banned.
							</Alert>
						)}
					</ContainerWithHeader>
					{statsVisible && <RecentMatches steamId={String(player.steamId)} />}
					<ContainerWithHeader title={"External Links"} iconLeft={<LinkIcon />} padding={2}>
						<Grid container spacing={1}>
							{createExternalLinks(String(player.steamId)).map((l) => {
								return (
									<Grid size={{ xs: 6, sm: 4 }} key={l.url}>
										<Button
											fullWidth
											variant={"outlined"}
											startIcon={<OpenInNewIcon fontSize={"small"} />}
											href={l.url}
											target={"_blank"}
											rel={"noopener noreferrer"}
										>
											{l.title}
										</Button>
									</Grid>
								);
							})}
						</Grid>
					</ContainerWithHeader>
				</Stack>
			</Grid>
			<Grid size={{ xs: 12, md: 4 }}>
				<Stack spacing={3}>
					<ContainerWithHeader title={"Account Standing"} iconLeft={<VerifiedUserIcon />}>
						<List disablePadding>
							<StandingRow
								clean={vacBans === 0}
								title={"VAC Bans"}
								detail={vacBans === 0 ? "Clean record" : `${vacBans} recorded`}
							/>
							<StandingRow
								clean={gameBans === 0}
								title={"Game Bans"}
								detail={gameBans === 0 ? "Clean record" : `${gameBans} recorded`}
							/>
						</List>
					</ContainerWithHeader>
					<SteamIDList steamId={player.steamId} />
					<FriendsCard steamIds={friendIds} />
				</Stack>
			</Grid>
		</Grid>
	);
}
