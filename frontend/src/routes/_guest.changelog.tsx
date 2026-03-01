import NewReleasesIcon from "@mui/icons-material/NewReleases";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { getChangelogs } from "../api/app.ts";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { tf2Fonts } from "../theme.ts";
import { renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_guest/changelog")({
	component: Changelogs,
	loader: async ({ context }) => {
		const changelogs = await context.queryClient.fetchQuery({
			queryKey: ["changelogs"],
			queryFn: getChangelogs,
		});

		return { changelogs };
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Git Changelogs" }, match.context.title("Changelog")],
	}),
});

function Changelogs() {
	const theme = useTheme();
	const { appInfo } = Route.useRouteContext();
	const { changelogs } = Route.useLoaderData();
	return (
		<Grid container spacing={2}>
			{(changelogs ?? []).map((changelog) => (
				<Grid size={{ xs: 12 }} key={changelog.id}>
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
						<MarkDownRenderer body_md={changelog.body} assetURL={appInfo.asset_url} />
					</ContainerWithHeader>
				</Grid>
			))}
		</Grid>
	);
}
