import AdsClickIcon from "@mui/icons-material/AdsClick";
import AttachMoneyIcon from "@mui/icons-material/AttachMoney";
import ChatIcon from "@mui/icons-material/Chat";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import EventIcon from "@mui/icons-material/Event";
import GavelIcon from "@mui/icons-material/Gavel";
import LeaderboardIcon from "@mui/icons-material/Leaderboard";
import MarkUnreadChatAltIcon from "@mui/icons-material/MarkUnreadChatAlt";
import PieChartIcon from "@mui/icons-material/PieChart";
import StorageIcon from "@mui/icons-material/Storage";
import SupportIcon from "@mui/icons-material/Support";
import VideocamIcon from "@mui/icons-material/Videocam";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { ButtonLink } from "../component/ButtonLink.tsx";
import { NewsView } from "../component/NewsView";
import RouterLink from "../component/RouterLink.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { blu, red, tf2Fonts } from "../theme.ts";

export const Route = createFileRoute("/_guest/")({
	component: Index,
	head: ({ match }) => ({
		meta: match.context.meta({ title: "Home" }),
	}),
});

function Index() {
	const { appInfo } = Route.useRouteContext();
	const { profile, hasPermission } = useAuth();
	const showPlay = profile && profile.banId === 0 && appInfo.serversEnabled;
	const showAppeal = profile && profile.banId !== 0 && appInfo.reportsEnabled;
	const banner = appInfo.bannerImage === "" ? "/banner-default.webp" : appInfo.bannerImage;

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Paper
					sx={{
						padding: 2.5,
						overflow: "hidden",
						position: "relative",
						backgroundImage: `url(${banner})`,
						backgroundSize: "cover",
						backgroundPosition: "center 20%",
					}}
				>
					<Box
						sx={{
							position: "absolute",
							inset: 0,
							background: "linear-gradient(to bottom, rgba(0, 0, 0, 0.55), rgba(0, 0, 0, 0.35))",
						}}
					/>
					<Box
						sx={{
							height: 4,
							marginX: -2.5,
							marginTop: -2.5,
							marginBottom: 2,
							position: "relative",
							background: `linear-gradient(90deg, ${blu} 0%, ${blu} 50%, ${red} 50%, ${red} 100%)`,
						}}
					/>
					<Stack
						direction={{ xs: "column", md: "row" }}
						spacing={2}
						sx={{ justifyContent: "space-between", alignItems: { md: "center" }, position: "relative" }}
					>
						<Box>
							<Typography variant="overline" sx={{ letterSpacing: 1.5, color: "#fff", opacity: 0.85 }}>
								Welcome To
							</Typography>
							<Typography
								variant="h3"
								sx={{ ...tf2Fonts, fontWeight: 700, lineHeight: 1.1, color: "#fff" }}
							>
								{appInfo.siteName}
							</Typography>
							{appInfo.siteDescription && (
								<Typography
									variant="body2"
									sx={{ marginTop: 1, maxWidth: 640, color: "#fff", opacity: 0.85 }}
								>
									{appInfo.siteDescription}
								</Typography>
							)}
						</Box>
						<Stack direction="row" spacing={1} sx={{ flexShrink: 0 }}>
							{showPlay && (
								<Button
									component={RouterLink}
									startIcon={<StorageIcon />}
									color={"success"}
									variant={"contained"}
									size={"large"}
									to={"/servers"}
								>
									Play Now!
								</Button>
							)}
							{showAppeal && (
								<Button
									component={RouterLink}
									startIcon={<SupportIcon />}
									color={"success"}
									variant={"contained"}
									size={"large"}
									to={`/ban/${profile.banId}`}
								>
									Appeal Ban
								</Button>
							)}
						</Stack>
					</Stack>
				</Paper>
			</Grid>
			<Grid size={{ xs: 12, sm: 12, md: 9 }}>
				<NewsView itemsPerPage={3} assetURL={appInfo.assetUrl} />
			</Grid>
			<Grid size={{ xs: 12, sm: 12, md: 3 }}>
				<Paper sx={{ padding: 2 }}>
					<Typography variant="overline" color="textSecondary" sx={{ letterSpacing: 1.5 }}>
						Quick Links
					</Typography>
					<Stack spacing={1.5} sx={{ marginTop: 1.5 }}>
						{appInfo.wikiEnabled && (
							<>
								<Button
									component={RouterLink}
									startIcon={<GavelIcon />}
									fullWidth
									color={"primary"}
									variant={"contained"}
									to={`/wiki/Rules`}
								>
									Rules
								</Button>

								<Button
									component={RouterLink}
									startIcon={<EventIcon />}
									fullWidth
									color={"primary"}
									variant={"contained"}
									to={"/wiki/Events"}
								>
									Events
								</Button>
							</>
						)}
						{appInfo.patreonEnabled && (
							<Button
								component={RouterLink}
								startIcon={<AttachMoneyIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/patreon`}
							>
								Donate
							</Button>
						)}
						{appInfo.contestsEnabled && (
							<Button
								component={RouterLink}
								startIcon={<EmojiEventsIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/contests`}
							>
								Contests
							</Button>
						)}
						{appInfo.mgeEnabled && (
							<Button
								component={RouterLink}
								startIcon={<AdsClickIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/mge`}
							>
								MGE Rankings
							</Button>
						)}
						{appInfo.chatlogsEnabled && (
							<Button
								component={RouterLink}
								startIcon={<ChatIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/chatlogs`}
							>
								Chat Logs
							</Button>
						)}
						{appInfo.demosEnabled && (
							<Button
								component={RouterLink}
								startIcon={<VideocamIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/stv`}
							>
								SourceTV
							</Button>
						)}
						{appInfo.statsEnabled && (
							<ButtonLink
								startIcon={<PieChartIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/matches`}
							>
								Recent Matches
							</ButtonLink>
						)}
						{appInfo.statsEnabled && (
							<ButtonLink
								startIcon={<LeaderboardIcon />}
								fullWidth
								color={"primary"}
								variant={"contained"}
								to={`/stats`}
							>
								Global Stats
							</ButtonLink>
						)}
						{appInfo.discordEnabled && appInfo.linkId !== "" && (
							<Button
								component={Link}
								startIcon={<MarkUnreadChatAltIcon />}
								fullWidth
								sx={{ backgroundColor: "#5865F2" }}
								variant={"contained"}
								href={`https://discord.gg/${appInfo.linkId}`}
							>
								Join Discord
							</Button>
						)}
					</Stack>
				</Paper>
			</Grid>
		</Grid>
	);
}
