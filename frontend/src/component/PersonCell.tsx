import AccountCircleIcon from "@mui/icons-material/AccountCircle";
import ChatIcon from "@mui/icons-material/Chat";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import NoAccountsIcon from "@mui/icons-material/NoAccounts";
import PublicIcon from "@mui/icons-material/Public";
import ReportIcon from "@mui/icons-material/Report";
import VideocamIcon from "@mui/icons-material/Videocam";
import WifiFindIcon from "@mui/icons-material/WifiFind";
import { IconButton, Menu, Typography, useTheme } from "@mui/material";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import ListItemIcon from "@mui/material/ListItemIcon";
import MenuItem from "@mui/material/MenuItem";
import Tooltip from "@mui/material/Tooltip";
import { useNavigate } from "@tanstack/react-router";
import React, { type MouseEventHandler, type PropsWithChildren, useCallback, useMemo, useState } from "react";
import SteamID from "steamid";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { Privilege } from "../rpc/person/v1/privilege_pb.ts";
import { avatarHashToURL } from "../util/strings.ts";
import { MenuItemLink } from "./MenuItemLink.tsx";
import { TextLink } from "./TextLink.tsx";

export type PersonCellProps = {
	steamId: string | bigint;
	personaName: string;
	avatarHash: string;
	onClick?: MouseEventHandler | undefined;
} & PropsWithChildren;

export const PersonCell = ({ steamId, avatarHash, personaName, onClick, children }: PersonCellProps) => {
	const { hasPermission } = useAuth();
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
	const open = Boolean(anchorEl);
	const { sendFlash } = useUserFlashCtx();
	const navigate = useNavigate();
	const theme = useTheme();

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
			const sid = new SteamID(String(steamId));
			await navigator.clipboard.writeText(sid.toString());
			handleClose();
			sendFlash("success", `Copied to clipboard: ${sid.toString()}`);
		},
		[steamId, sendFlash, handleClose],
	);

	const menu = useMemo(() => {
		let items = [
			<MenuItemLink to={`/profile/$steamId`} params={{ steamId: String(steamId) }} key={20}>
				<ListItemIcon>
					<AccountCircleIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				Local Profile
			</MenuItemLink>,

			<MenuItem
				onClick={async () => {
					await navigate({ href: `https://steamcommunity.com/profiles/${steamId}` });
				}}
				key={30}
			>
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
			<MenuItemLink to={`/chatlogs`} search={{ columnFilters: [{ id: "steam_id", value: steamId }] }} key={50}>
				<ListItemIcon>
					<ChatIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				Chat Logs
			</MenuItemLink>,
			<MenuItemLink to={`/stv`} search={{ columnFilters: [{ id: "stats", value: steamId }] }} key={60}>
				<ListItemIcon>
					<VideocamIcon fontSize="small" color={"primary"} />
				</ListItemIcon>
				SourceTV History
			</MenuItemLink>,
		];
		if (hasPermission(Privilege.MODERATOR)) {
			items = [
				...items,
				<MenuItemLink
					to={"/admin/network/playersbyip"}
					search={{ columnFilters: [{ id: "steam_id", value: steamId }] }}
					key={70}
				>
					<ListItemIcon>
						<WifiFindIcon fontSize="small" color={"primary"} />
					</ListItemIcon>
					Connection History
				</MenuItemLink>,

				<MenuItemLink
					to={"/admin/bans"}
					search={{ columnFilters: [{ id: "target_id", value: steamId }] }}
					key={80}
				>
					<ListItemIcon>
						<NoAccountsIcon fontSize="small" color={"primary"} />
					</ListItemIcon>
					Ban History
				</MenuItemLink>,
				<MenuItemLink
					to={"/admin/reports"}
					search={{ columnFilters: [{ id: "target_id", value: steamId }] }}
					key={90}
				>
					<ListItemIcon>
						<ReportIcon fontSize="small" color={"primary"} />
					</ListItemIcon>
					Report History
				</MenuItemLink>,
			];
		}
		return items;
	}, [copySteamID, hasPermission, steamId, navigate]);

	return (
		<>
			<Box display={"flex"} alignItems={"center"} gap={"0.2rem"} minWidth={200}>
				<Tooltip title="Player Links">
					<IconButton
						onClick={handleClick}
						size="small"
						sx={{
							ml: 0,
						}}
						aria-controls={open ? "account-menu" : undefined}
						aria-haspopup="true"
						aria-expanded={open ? "true" : undefined}
					>
						<Avatar
							alt={personaName}
							src={avatarHashToURL(avatarHash, "small")}
							variant={"rounded"}
							sizes=""
							sx={{ height: "32px", width: "32px" }}
							slotProps={{ img: { loading: "lazy" } }}
						>
							P
						</Avatar>
					</IconButton>
				</Tooltip>
				{children ?? (
					<TextLink
						style={{
							color:
								theme.palette.mode === "dark"
									? theme.palette.primary.light
									: theme.palette.primary.dark,
						}}
						to={"/profile/$steamId"}
						params={{ steamId: String(steamId) }}
						onClick={onClick ?? undefined}
					>
						{personaName !== "" ? personaName : String(steamId)}
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
					<Avatar src={avatarHashToURL(avatarHash)} />
					<Typography fontWeight={700}>{personaName ?? steamId}</Typography>
				</Box>
				{menu}
			</Menu>
		</>
	);
};
