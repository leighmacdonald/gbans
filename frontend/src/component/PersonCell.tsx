import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import Avatar from "@mui/material/Avatar";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import type { MouseEventHandler } from "react";
import SteamID from "steamid";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { avatarHashToURL } from "../util/text.tsx";
import { ButtonLink } from "./ButtonLink.tsx";

export interface PersonCellProps {
	steam_id: string;
	personaname: string;
	avatar_hash: string;
	onClick?: MouseEventHandler | undefined;
	showCopy?: boolean;
}

export const PersonCell = ({ steam_id, avatar_hash, personaname, onClick, showCopy = false }: PersonCellProps) => {
	const theme = useTheme();
	const { sendFlash } = useUserFlashCtx();

	return (
		<Stack
			minWidth={200}
			direction={"row"}
			alignItems={"center"}
			overflow={"hidden"}
			paddingLeft={1}
			paddingRight={1}
		>
			{showCopy && (
				<Tooltip title={"Copy Steamid"}>
					<IconButton
						color={"primary"}
						onClick={async (event) => {
							event.preventDefault();
							event.stopPropagation();
							const sid = new SteamID(steam_id);
							await navigator.clipboard.writeText(sid.toString());
							sendFlash("success", `Copied to clipboard: ${sid.toString()}`);
						}}
					>
						<ContentCopyIcon />
					</IconButton>
				</Tooltip>
			)}
			<ButtonLink
				to={"/profile/$steamId"}
				params={{ steamId: steam_id }}
				onClick={onClick ?? undefined}
				variant={"text"}
				sx={{
					backgroundColor: theme.palette.background.default,
					color: theme.palette.primary.contrastText,
					"&:hover": {
						cursor: "pointer",
						textDecoration: "underline",
						// color: theme.palette.getContrastText(theme.palette.background.default),
						backgroundColor: theme.palette.background.default,
					},
				}}
				fullWidth
				startIcon={
					<Avatar
						alt={personaname}
						src={avatarHashToURL(avatar_hash, "small")}
						variant={"rounded"}
						sx={{ height: "32px", width: "32px" }}
					/>
				}
			>
				<Typography fontWeight={"bold"} variant={"body1"} sx={{ width: "100%" }}>
					{personaname !== "" ? personaname : steam_id}
				</Typography>
			</ButtonLink>
		</Stack>
	);
};
