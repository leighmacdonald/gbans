import PregnantWomanIcon from "@mui/icons-material/PregnantWoman";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { format, fromUnixTime } from "date-fns";
import { apiGetProfile } from "../api";
import { ErrorCode } from "../error.tsx";
import { avatarHashToURL } from "../util/text.tsx";
import { isValidSteamDate, renderDateTime } from "../util/time.ts";
import { emptyOrNullString } from "../util/types.ts";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { ErrorDetails } from "./ErrorDetails.tsx";
import { LoadingPlaceholder } from "./LoadingPlaceholder";

export const ProfileInfoBox = ({ steam_id }: { steam_id: string }) => {
	const { data: profile, isLoading } = useQuery({
		queryKey: ["profile", { steam_id }],
		queryFn: async () => await apiGetProfile(steam_id),
	});

	if (isLoading) {
		return <LoadingPlaceholder />;
	}

	if (!profile) {
		return <ErrorDetails error={ErrorCode.Unknown} />;
	}

	return (
		<ContainerWithHeader title={"Profile"} iconLeft={<PregnantWomanIcon />} marginTop={0}>
			<Grid container spacing={1}>
				<Grid size={{ xs: 12 }}>
					<Avatar
						variant={"square"}
						src={avatarHashToURL(profile.player.avatar_hash)}
						alt={"Profile Avatar"}
						sx={{ width: "100%", height: "100%" }}
					/>
				</Grid>
				<Grid size={{ xs: 12 }}>
					<Box>
						<Typography
							variant={"h3"}
							display="inline"
							style={{ wordBreak: "break-word", whiteSpace: "pre-line" }}
						>
							{profile.player.persona_name}
						</Typography>
					</Box>
				</Grid>

				<Grid size={{ xs: 12 }}>
					<Typography variant={"body1"}>First Seen: {renderDateTime(profile.player.created_on)}</Typography>
				</Grid>

				{!emptyOrNullString(profile.player.loc_state_code) ||
					(!emptyOrNullString(profile.player.loc_country_code) && (
						<Grid size={{ xs: 12 }}>
							<Typography variant={"body1"}>
								{[profile.player.loc_state_code, profile.player.loc_country_code]
									.filter((x) => x)
									.join(",")}
							</Typography>
						</Grid>
					))}

				{isValidSteamDate(fromUnixTime(profile.player.time_created)) && (
					<Grid size={{ xs: 12 }}>
						<Typography variant={"body1"}>
							Created: {format(fromUnixTime(profile.player.time_created), "yyyy-MM-dd")}
						</Typography>
					</Grid>
				)}
			</Grid>
		</ContainerWithHeader>
	);
};
