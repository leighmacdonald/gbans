import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import Menu from "@mui/material/Menu";
import { useTheme } from "@mui/material/styles";
import Tooltip from "@mui/material/Tooltip";
import React, { type MouseEventHandler, type PropsWithChildren, useCallback, useState } from "react";
import { avatarHashToURL } from "../util/strings.ts";
import { emptyOrNullString } from "../util/types.ts";
import { PlayerLinksMenu } from "./PlayerLinksMenu.tsx";
import { TextLink } from "./TextLink.tsx";

export type PersonCellProps = {
	steamId: string;
	personaName?: string;
	avatarHash?: string;
	onClick?: MouseEventHandler | undefined;
} & PropsWithChildren;

const PersonCellInner = ({ steamId, avatarHash, personaName, onClick, children }: PersonCellProps) => {
	const theme = useTheme();
	const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
	const open = Boolean(anchorEl);

	const handleClick = (event: React.MouseEvent<HTMLElement>) => {
		setAnchorEl(event.currentTarget);
	};

	const handleClose = useCallback(() => {
		setAnchorEl(null);
	}, []);

	return (
		<>
			<Box
				sx={{
					display: "flex",
					alignItems: "center",
					gap: "0.2rem",
					minWidth: 200,
				}}
			>
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
						{!emptyOrNullString(personaName) ? personaName : String(steamId)}
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
				<PlayerLinksMenu
					steamId={String(steamId)}
					personaName={personaName}
					avatarHash={avatarHash}
					onAction={handleClose}
				/>
			</Menu>
		</>
	);
};

export const PersonCell = React.memo(
	PersonCellInner,
	(prev, next) =>
		prev.steamId === next.steamId &&
		prev.personaName === next.personaName &&
		prev.avatarHash === next.avatarHash &&
		prev.onClick === next.onClick &&
		prev.children === next.children,
);
