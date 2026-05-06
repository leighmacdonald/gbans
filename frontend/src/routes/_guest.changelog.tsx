import { useQuery } from "@connectrpc/connect-query";
import NewReleasesIcon from "@mui/icons-material/NewReleases";
import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { changelog } from "../rpc/config/v1/config-ConfigService_connectquery.ts";
import { tf2Fonts } from "../theme.ts";
import { renderTimestamp } from "../util/time.ts";

export const Route = createFileRoute("/_guest/changelog")({
	component: Changelogs,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Git Changelogs" }, match.context.title("Changelog")],
	}),
});

function Changelogs() {
	const theme = useTheme();
	const { appInfo } = Route.useRouteContext();
	const { data, isLoading } = useQuery(changelog);

	if (isLoading) {
		return;
	}

	return (
		<Grid container spacing={2}>
			{(data?.changelog ?? []).map((changelog) => (
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
								<Typography padding={1}>{renderTimestamp(changelog.createdAt)}</Typography>
							</Stack>
						}
						iconLeft={<NewReleasesIcon />}
					>
						<MarkDownRenderer body_md={changelog.body} assetURL={appInfo.assetUrl} />
					</ContainerWithHeader>
				</Grid>
			))}
		</Grid>
	);
}
