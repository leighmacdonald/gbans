import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import ChatIcon from "@mui/icons-material/Chat";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import NoAccountsIcon from "@mui/icons-material/NoAccounts";
import PublicIcon from "@mui/icons-material/Public";
import VideocamIcon from "@mui/icons-material/Videocam";
import WifiFindIcon from "@mui/icons-material/WifiFind";
import { IconButton, Menu, Typography } from "@mui/material";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import ListItemIcon from "@mui/material/ListItemIcon";
import MenuItem from "@mui/material/MenuItem";
import Tooltip from "@mui/material/Tooltip";
import { useNavigate } from "@tanstack/react-router";
import { type MouseEventHandler, type PropsWithChildren, useCallback, useMemo, useState } from "react";
import SteamID from "steamid";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { PermissionLevel } from "../schema/people.ts";
import { avatarHashToURL } from "../util/text.tsx";
import { MenuItemLink } from "./MenuItemLink.tsx";
import { TextLink } from "./TextLink.tsx";

export type PersonCellProps = {
	steam_id: string;
	personaname: string;
	avatar_hash: string;
	onClick?: MouseEventHandler | undefined;
} & PropsWithChildren;

export const PersonCell = ({ steam_id, avatar_hash, personaname, onClick, children }: PersonCellProps) => {
	const { hasPermission } = useAuth();
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
	const open = Boolean(anchorEl);
	const { sendFlash } = useUserFlashCtx();
	const navigate = useNavigate();

	const handleClick = (event: React.MouseEvent<HTMLElement>) => {
		setAnchorEl(event.currentTarget);
	};

	const handleClose = useCallback(() => {
		setAnchorEl(null);
	}, []);

	const copySteamID = useCallback(
		async (event: React.MouseEvent<HTMLElement, MouseEvent>) => {
			event.preventDefault();
			event.stopPropagation();
			const sid = new SteamID(steam_id);
			await navigator.clipboard.writeText(sid.toString());
			handleClose();
			sendFlash("success", `Copied to clipboard: ${sid.toString()}`);
		},
		[steam_id, sendFlash, handleClose],
	);

	const menu = useMemo(() => {
		let items = [
			<MenuItemLink to={`/profile/$steamId`} params={{ steamId: steam_id }} key={20}>
				<ListItemIcon>
					<AccountCircleIcon fontSize="small" />
				</ListItemIcon>
				Local Profile
			</MenuItemLink>,

			<MenuItem
				onClick={() => {
					navigate({ href: `https://steamcommunity.com/profiles/${steam_id}` });
				}}
				key={30}
			>
				<ListItemIcon>
					<PublicIcon fontSize="small" />
				</ListItemIcon>
				Steam Profile
			</MenuItem>,
			<MenuItem onClick={copySteamID} key={40}>
				<ListItemIcon>
					<ContentCopyIcon fontSize="small" />
				</ListItemIcon>
				Copy SteamID 64
			</MenuItem>,
			<MenuItemLink to={`/chatlogs`} search={{ columnFilters: [{ id: "steam_id", value: steam_id }] }} key={50}>
				<ListItemIcon>
					<ChatIcon fontSize="small" />
				</ListItemIcon>
				Chat Logs
			</MenuItemLink>,
			<MenuItemLink to={`/stv`} search={{ columnFilters: [{ id: "stats", value: steam_id }] }} key={60}>
				<ListItemIcon>
					<VideocamIcon fontSize="small" />
				</ListItemIcon>
				SourceTV History
			</MenuItemLink>,
		];
		if (hasPermission(PermissionLevel.Moderator)) {
			items = [
				...items,
				<MenuItemLink
					to={"/admin/network/playersbyip"}
					search={{ columnFilters: [{ id: "steam_id", value: steam_id }] }}
					key={70}
				>
					<ListItemIcon>
						<WifiFindIcon fontSize="small" />
					</ListItemIcon>
					Connection History
				</MenuItemLink>,

				<MenuItemLink
					to={"/admin/bans"}
					search={{ columnFilters: [{ id: "target_id", value: steam_id }] }}
					key={80}
				>
					<ListItemIcon>
						<NoAccountsIcon fontSize="small" />
					</ListItemIcon>
					Ban History
				</MenuItemLink>,
				<MenuItemLink
					to={"/admin/reports"}
					search={{ columnFilters: [{ id: "target_id", value: steam_id }] }}
					key={90}
				>
					<ListItemIcon>
						<VideocamIcon fontSize="small" />
					</ListItemIcon>
					Report History
				</MenuItemLink>,
			];
		}
		return items;
	}, [copySteamID, hasPermission, steam_id, navigate]);

	return (
		<>
			<Box display={"flex"} alignItems={"center"} gap={"0.2rem"}>
				<Tooltip title="Player Links">
					<IconButton
						onClick={handleClick}
						size="small"
						sx={{
							ml: 2,
						}}
						aria-controls={open ? "account-menu" : undefined}
						aria-haspopup="true"
						aria-expanded={open ? "true" : undefined}
					>
						<Avatar
							alt={personaname}
							src={avatarHashToURL(avatar_hash, "small")}
							variant={"rounded"}
							sizes=""
							sx={{ height: "32px", width: "32px" }}
						>
							P
						</Avatar>
					</IconButton>
				</Tooltip>
				{children ?? (
					<TextLink to={"/profile/$steamId"} params={{ steamId: steam_id }} onClick={onClick ?? undefined}>
						{personaname !== "" ? personaname : steam_id}
					</TextLink>
				)}
			</Box>
			<Menu
				anchorEl={anchorEl}
				id="player-menu"
				open={open}
				onClose={handleClose}
				onClick={handleClose}
				slotProps={{
					paper: {
						elevation: 0,
						sx: {
							overflow: "visible",
							filter: "drop-shadow(0px 2px 8px rgba(0,0,0,0.32))",
							mt: 1.5,
							"& .MuiAvatar-root": {
								width: 32,
								height: 32,
								ml: -0.5,
								mr: 1,
							},
							"&::before": {
								content: '""',
								display: "block",
								position: "absolute",
								top: 0,
								right: 14,
								width: 10,
								height: 10,
								bgcolor: "background.paper",
								transform: "translateY(-50%) rotate(45deg)",
								zIndex: 0,
							},
						},
					},
				}}
				transformOrigin={{ horizontal: "left", vertical: "top" }}
				anchorOrigin={{ horizontal: "right", vertical: "bottom" }}
			>
				<Box
					sx={{ backgroundColor: "primary.main", color: "primary.contrastText" }}
					display={"flex"}
					alignItems={"center"}
					padding={2}
					gap={1}
				>
					<Avatar src={avatarHashToURL(avatar_hash)} />
					<Typography fontWeight={700}>{personaname ?? steam_id}</Typography>
				</Box>
				{menu}
			</Menu>
		</>
	);
};
