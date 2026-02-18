import NewReleasesIcon from "@mui/icons-material/NewReleases";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { getChangelogs } from "../api/app.ts";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { tf2Fonts } from "../theme.ts";
import { renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_guest/changelog")({
	component: Changelogs,
});

function Changelogs() {
	const theme = useTheme();

	const { data: changelogs, isLoading } = useQuery({
		queryKey: ["changelogs"],
		queryFn: getChangelogs,
	});

	return (
		<Grid container spacing={2}>
			{!isLoading &&
				(changelogs ?? []).map((changelog) => (
					<Grid size={{ xs: 12 }}>
						<ContainerWithHeader
							title={
								<Stack direction={"row"}>
									<Typography
										padding={1}
										sx={{
											backgroundColor: theme.palette.primary.main,
											color: theme.palette.common.white,
											...tf2Fonts,
										}}
									>
										{changelog.name}
									</Typography>{" "}
									<Typography padding={1}>{renderDateTime(changelog.created_at)}</Typography>
								</Stack>
							}
							iconLeft={<NewReleasesIcon />}
						>
							<MarkDownRenderer body_md={changelog.body} />
						</ContainerWithHeader>
					</Grid>
				))}
		</Grid>
	);
}
