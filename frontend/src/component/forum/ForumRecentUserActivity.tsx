import { useQuery } from "@connectrpc/connect-query";
import PeopleIcon from "@mui/icons-material/People";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { activeUsers } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { ContainerWithHeader } from "../ContainerWithHeader.tsx";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";
import RouterLink from "../RouterLink.tsx";

export const ForumRecentUserActivity = () => {
	const { data, isLoading } = useQuery(activeUsers);

	const theme = useTheme();

	return (
		<ContainerWithHeader title={`Users Online ${data?.userActivity?.length ?? 0}`} iconLeft={<PeopleIcon />}>
			<Grid container>
				{isLoading ? (
					<LoadingPlaceholder />
				) : (
					data?.userActivity?.map((a) => {
						return (
							<Grid size={{ xs: "auto" }} spacing={1} key={`activity-${a.steamId}`}>
								<Typography
									sx={{
										display: "inline",
										textDecoration: "none",
										"&:hover": {
											textDecoration: "underline",
										},
									}}
									variant={"body2"}
									color={theme.palette.text.secondary}
									component={RouterLink}
									to={`/profile/${a.steamId}`}
								>
									{a.personaName}
								</Typography>
							</Grid>
						);
					})
				)}
			</Grid>
		</ContainerWithHeader>
	);
};
