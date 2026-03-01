import AttachMoneyIcon from "@mui/icons-material/AttachMoney";
import ChatIcon from "@mui/icons-material/Chat";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import EventIcon from "@mui/icons-material/Event";
import GavelIcon from "@mui/icons-material/Gavel";
import MarkUnreadChatAltIcon from "@mui/icons-material/MarkUnreadChatAlt";
import PieChartIcon from "@mui/icons-material/PieChart";
import StorageIcon from "@mui/icons-material/Storage";
import SupportIcon from "@mui/icons-material/Support";
import VideocamIcon from "@mui/icons-material/Videocam";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Stack from "@mui/material/Stack";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { NewsView } from "../component/NewsView";
import RouterLink from "../component/RouterLink.tsx";
import { useAuth } from "../hooks/useAuth.ts";

export const Route = createFileRoute("/_guest/")({
	component: Index,
	head: ({ match }) => ({
		meta: [
			{ name: "og:description", content: match.context.appInfo.site_description },
			{ name: "og:title", content: `Home - ${match.context.appInfo.site_name}` },
			match.context.title("Home"),
		],
	}),
});

function Index() {
	const navigate = useNavigate();
	const { appInfo } = Route.useRouteContext();
	const { profile } = useAuth();

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12, sm: 12, md: 9 }}>
				<NewsView itemsPerPage={3} assetURL={appInfo.asset_url} />
			</Grid>
			<Grid size={{ xs: 12, sm: 12, md: 3 }}>
				<Stack spacing={3}>
					{profile && profile.ban_id === 0 && appInfo.servers_enabled && (
						<Button
							startIcon={<StorageIcon />}
							fullWidth
							color={"success"}
							variant={"contained"}
							onClick={async () => {
								await navigate({ to: "/servers" });
							}}
						>
							Play Now!
						</Button>
					)}
					{profile && profile.ban_id !== 0 && appInfo.reports_enabled && (
						<Button
							startIcon={<SupportIcon />}
							fullWidth
							color={"success"}
							variant={"contained"}
							onClick={async () => {
								await navigate({
									to: `/ban/${profile.ban_id}`,
								});
							}}
						>
							Appeal Ban
						</Button>
					)}
					{appInfo.wiki_enabled && (
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
					{appInfo.patreon_enabled && (
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
					{appInfo.contests_enabled && (
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
					{appInfo.chatlogs_enabled && (
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
					{appInfo.demos_enabled && (
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
					{appInfo.stats_enabled && (
						<Button
							component={RouterLink}
							startIcon={<PieChartIcon />}
							fullWidth
							color={"primary"}
							variant={"contained"}
							to={`/stats`}
						>
							Stats (Beta)
						</Button>
					)}
					{appInfo.discord_enabled && appInfo.link_id !== "" && (
						<Button
							component={Link}
							startIcon={<MarkUnreadChatAltIcon />}
							fullWidth
							sx={{ backgroundColor: "#5865F2" }}
							variant={"contained"}
							href={`https://discord.gg/${appInfo.link_id}`}
						>
							Join Discord
						</Button>
					)}
				</Stack>
			</Grid>
		</Grid>
	);
}
