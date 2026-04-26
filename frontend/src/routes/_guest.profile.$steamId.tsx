import { type Timestamp, timestampDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import LinkIcon from "@mui/icons-material/Link";
import LocalLibraryIcon from "@mui/icons-material/LocalLibrary";
import Avatar from "@mui/material/Avatar";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { format } from "date-fns";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { SteamIDList } from "../component/SteamIDList.tsx";
import { profile } from "../rpc/person/v1/person-PersonService_connectquery.ts";
import { createExternalLinks } from "../util/history.ts";
import { avatarHashToURL } from "../util/strings.ts";
import { isValidSteamDate, renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_guest/profile/$steamId")({
	component: ProfilePage,
	head: () => ({
		meta: [{ name: "description", content: "Player Profile" }],
	}),
});

function ProfilePage() {
	const { data } = useQuery(profile);

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, md: 8 }}>
				<ContainerWithHeader title={"Profile"}>
					<Grid container spacing={2}>
						<Grid size={{ xs: 4 }}>
							<Avatar
								variant={"square"}
								src={avatarHashToURL(data?.profile?.player?.avatarHash)}
								alt={"Profile Avatar"}
								sx={{ width: "100%", height: "100%", minHeight: 240 }}
							/>
						</Grid>
						<Grid size={{ xs: 8 }}>
							<Stack spacing={2}>
								<Typography
									variant={"h3"}
									display="inline"
									style={{ wordBreak: "break-word", whiteSpace: "pre-line" }}
								>
									{data?.profile?.player?.name}
								</Typography>
								<Typography variant={"body1"}>
									Created: {renderTimestamp(data?.profile?.player?.timeCreated)}
								</Typography>
								{/*{!emptyOrNullString(data?.profile?.player?.loc_state_code) ||*/}
								{/*	(!emptyOrNullString(profile.player.loc_country_code) && (*/}
								{/*		<Typography variant={"body1"}>*/}
								{/*			{[profile.player.loc_state_code, profile.player.loc_country_code]*/}
								{/*				.filter((x) => x)*/}
								{/*				.join(",")}*/}
								{/*		</Typography>*/}
								{/*	))}*/}
								{isValidSteamDate(timestampDate(data?.profile?.player?.timeCreated as Timestamp)) && (
									<Typography variant={"body1"}>
										Created:{" "}
										{format(
											timestampDate(data?.profile?.player?.timeCreated as Timestamp),
											"yyyy-MM-dd",
										)}
									</Typography>
								)}
							</Stack>
						</Grid>
					</Grid>
				</ContainerWithHeader>
			</Grid>
			<Grid size={{ xs: 6, md: 2 }}>
				<ContainerWithHeader title={"Status"} iconLeft={<LocalLibraryIcon />} marginTop={0}>
					<Stack spacing={1} padding={1} justifyContent={"space-evenly"}>
						<Chip color={Number(data?.profile?.player?.vacBans) > 0 ? "error" : "success"} label={"VAC"} />
						<Chip
							color={Number(data?.profile?.player?.gameBans) > 0 ? "error" : "success"}
							label={"Game Ban"}
						/>
						{/*<Chip*/}
						{/*	color={profile.player.economy_ban !== "none" ? "error" : "success"}*/}
						{/*	label={"Economy Ban"}*/}
						{/*/>*/}
						{/*<Chip*/}
						{/*	color={data?.profile?.player?.community_banned ? "error" : "success"}*/}
						{/*	label={"Community Ban"}*/}
						{/*/>*/}
					</Stack>
				</ContainerWithHeader>
			</Grid>
			<Grid size={{ xs: 6, md: 2 }}>
				<SteamIDList steam_id={data?.profile?.player?.steamId.toString() ?? ""} />
			</Grid>
			{/*{isAuthenticated() &&
				(userProfile.steam_id === profile.player.steam_id || !profile.settings.stats_hidden) && (
					<>
						<Grid size={{ xs: 12 }}>
							{<PlayerStatsOverallContainer steam_id={profile.player.steam_id} />}
						</Grid>
						<Grid size={{ xs: 12 }}>
							<ContainerWithHeader title={"Player Overall Stats By Class"} iconLeft={<BarChartIcon />}>
								<PlayerClassStatsTable steam_id={profile.player.steam_id} />
							</ContainerWithHeader>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<ContainerWithHeader title={"Overall Player Weapon Stats"} iconLeft={<InsightsIcon />}>
								<PlayerWeaponsStatListContainer steamId={profile.player.steam_id} />
							</ContainerWithHeader>
						</Grid>
					</>
				)}*/}
			<Grid size={{ xs: 128 }}>
				<ContainerWithHeader title={"External Links"} iconLeft={<LinkIcon />}>
					<Grid container spacing={1} paddingLeft={1}>
						{createExternalLinks(String(data?.profile?.player?.steamId)).map((l) => {
							return (
								<Grid size={{ xs: 4 }} key={`btn-${l.url}`} padding={1}>
									<Button
										fullWidth
										color={"secondary"}
										variant={"contained"}
										component={Link}
										href={l.url}
										key={l.url}
									>
										{l.title}
									</Button>
								</Grid>
							);
						})}
					</Grid>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
