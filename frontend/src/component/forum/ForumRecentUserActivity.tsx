import PeopleIcon from "@mui/icons-material/People";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { apiForumActiveUsers } from "../../api/forum.ts";
import { ContainerWithHeader } from "../ContainerWithHeader.tsx";
import { LoadingPlaceholder } from "../LoadingPlaceholder.tsx";
import RouterLink from "../RouterLink.tsx";

export const ForumRecentUserActivity = () => {
	const { data: activity, isLoading } = useQuery({
		queryKey: ["forumActivity"],
		queryFn: async () => {
			return await apiForumActiveUsers();
		},
	});

	const theme = useTheme();

	return (
		<ContainerWithHeader title={`Users Online ${activity?.length ?? 0}`} iconLeft={<PeopleIcon />}>
			<Grid container>
				{isLoading ? (
					<LoadingPlaceholder />
				) : (
					activity?.map((a) => {
						return (
							<Grid size={{ xs: "auto" }} spacing={1} key={`activity-${a.steam_id}`}>
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
									to={`/profile/${a.steam_id}`}
								>
									{a.personaname}
								</Typography>
							</Grid>
						);
					})
				)}
			</Grid>
		</ContainerWithHeader>
	);
};
