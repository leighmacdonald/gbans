import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import ChatIcon from "@mui/icons-material/Chat";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import FilterListIcon from "@mui/icons-material/FilterList";
import NoAccountsIcon from "@mui/icons-material/NoAccounts";
import PublicIcon from "@mui/icons-material/Public";
import ReportIcon from "@mui/icons-material/Report";
import VideocamIcon from "@mui/icons-material/Videocam";
import WifiFindIcon from "@mui/icons-material/WifiFind";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import ListItemIcon from "@mui/material/ListItemIcon";
import MenuItem from "@mui/material/MenuItem";
import Typography from "@mui/material/Typography";
import { useNavigate } from "@tanstack/react-router";
import type { ReactNode } from "react";
import SteamID from "steamid";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import { avatarHashToURL } from "../util/strings.ts";
import { emptyOrNullString } from "../util/types.ts";
import { MenuItemLink } from "./MenuItemLink.tsx";

interface PlayerLinksMenuProps {
	// Any form the steamid package accepts, e.g. "[U:1:68745073]" or a 64-bit id.
	steamId: string;
	personaName?: string;
	avatarHash?: string;
	// Called after a non-navigation item is activated, so the caller can close its menu.
	onAction: () => void;
	// When provided, a "filter to this player" entry is added at the top. It is
	// called with the id exactly as passed to this component, so callers can
	// filter on the form that appears in their own data.
	onFilterPlayer?: (steamId: string) => void;
}

export const PlayerLinksMenu = ({
	steamId,
	personaName,
	avatarHash,
	onAction,
	onFilterPlayer,
}: PlayerLinksMenuProps) => {
	const { hasPermission } = useAuth();
	const { sendFlash } = useUserFlashCtx();
	const navigate = useNavigate();
	const steamId64 = new SteamID(steamId).getSteamID64();

	const copySteamID = () => {
		void navigator.clipboard.writeText(steamId64).then(() => {
			sendFlash("success", `Copied to clipboard: ${steamId64}`);
		});
		onAction();
	};

	const openSteamProfile = () => {
		navigate({ href: `https://steamcommunity.com/profiles/${steamId64}` });
		onAction();
	};

	const items: ReactNode[] = [];
	if (onFilterPlayer) {
		items.push(
			<MenuItem key={10} onClick={() => onFilterPlayer(steamId)}>
				<ListItemIcon>
					<FilterListIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				{emptyOrNullString(personaName) ? "Filter to this player" : `Filter to ${personaName}`}
			</MenuItem>,
		);
	}
	items.push(
		<MenuItemLink to={`/profile/$steamId`} params={{ steamId: steamId64 }} key={20}>
			<ListItemIcon>
				<AccountCircleIcon fontSize="small" color={"primary"} />
			</ListItemIcon>
			Local Profile
		</MenuItemLink>,
		<MenuItem onClick={openSteamProfile} key={30}>
			<ListItemIcon>
				<PublicIcon fontSize="small" color={"primary"} />
			</ListItemIcon>
			Steam Profile
		</MenuItem>,
		<MenuItem onClick={copySteamID} key={40}>
			<ListItemIcon>
				<ContentCopyIcon fontSize="small" color={"primary"} />
			</ListItemIcon>
			Copy SteamID 64
		</MenuItem>,
	);
	if (hasPermission(Permission.CHATLOG_READ)) {
		items.push(
			<MenuItemLink to={`/chatlogs`} search={{ columnFilters: [{ id: "steamId", value: steamId64 }] }} key={50}>
				<ListItemIcon>
					<ChatIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				Chat Logs
			</MenuItemLink>,
		);
	}
	if (hasPermission(Permission.DEMO_READ)) {
		items.push(
			<MenuItemLink to={`/stv`} search={{ columnFilters: [{ id: "stats", value: steamId64 }] }} key={60}>
				<ListItemIcon>
					<VideocamIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				SourceTV History
			</MenuItemLink>,
		);
	}
	if (hasPermission(Permission.NETWORK_READ)) {
		items.push(
			<MenuItemLink
				to={"/admin/network/playersbyip"}
				search={{ columnFilters: [{ id: "steamId", value: steamId64 }] }}
				key={70}
			>
				<ListItemIcon>
					<WifiFindIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				Connection History
			</MenuItemLink>,
		);
	}
	if (hasPermission(Permission.BAN_CREATE)) {
		items.push(
			<MenuItemLink
				to={"/admin/bans"}
				search={{ columnFilters: [{ id: "targetId", value: steamId64 }] }}
				key={80}
			>
				<ListItemIcon>
					<NoAccountsIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				Ban History
			</MenuItemLink>,
		);
	}
	if (hasPermission(Permission.REPORT_ADMIN)) {
		items.push(
			<MenuItemLink
				to={"/admin/reports"}
				search={{ columnFilters: [{ id: "targetId", value: steamId64 }] }}
				key={90}
			>
				<ListItemIcon>
					<ReportIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				Report History
			</MenuItemLink>,
		);
	}

	return (
		<>
			<Box
				sx={{
					display: "flex",
					alignItems: "center",
					padding: 2,
					gap: 1,
					backgroundColor: "primary.main",
					color: "primary.contrastText",
				}}
			>
				{avatarHash ? <Avatar src={avatarHashToURL(avatarHash)} /> : null}
				<Typography
					sx={{
						fontWeight: 700,
					}}
				>
					{String(personaName ?? steamId)}
				</Typography>
			</Box>
			{items}
		</>
	);
};
