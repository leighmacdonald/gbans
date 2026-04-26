import { useQuery } from "@connectrpc/connect-query";
import PaymentIcon from "@mui/icons-material/Payment";
import SearchIcon from "@mui/icons-material/Search";
import SettingsInputComponentIcon from "@mui/icons-material/SettingsInputComponent";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { createFileRoute, Navigate } from "@tanstack/react-router";
import { z } from "zod/v4";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { ImageBox } from "../component/ImageBox.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { patronCampaigns } from "../rpc/patreon/v1/patreon-PatreonService_connectquery.ts";
import { ensureFeatureEnabled } from "../util/features.ts";

const patreonSearchSchema = z.object({
	redirect: z.string().catch("/"),
});

export const Route = createFileRoute("/_guest/patreon")({
	component: Patreon,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.patreonEnabled);
	},

	validateSearch: (search) => patreonSearchSchema.parse(search),
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Patreon Campaigns" }, match.context.title("Patreon")],
	}),
});

function Patreon() {
	const { isAuthenticated, profile } = useAuth();
	const { appInfo } = Route.useRouteContext();
	const theme = useTheme();

	const { data, isLoading } = useQuery(patronCampaigns);

	const followCallback = async () => {
		// const result = await queryClient.fetchQuery({
		// 	queryKey: ["callback"],
		// 	queryFn: ({ signal }) => apiGetPatreonLogin(signal),
		// });
		// window.open(result.url, "_self");
	};

	if (!appInfo.patreonEnabled) {
		return <Navigate to={"/"} />;
	}

	if (isLoading) {
		return;
	}

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeaderAndButtons
					title={`Patreon Campaign: ${data?.campaigns[0].attributes?.creationName}`}
					iconLeft={<PaymentIcon />}
					buttons={
						profile.patreonId
							? []
							: [
									<Button
										key={"connect"}
										variant={"contained"}
										color={"success"}
										disabled={!isAuthenticated() || profile.patreonId !== ""}
										onClick={followCallback}
										startIcon={<SettingsInputComponentIcon />}
									>
										Connect Patreon
									</Button>,
								]
					}
				>
					<Grid container>
						<Grid size={{ xs: 12 }}>
							<Stack spacing={1}>
								<Paper>
									<ImageBox
										height={"100%"}
										width={"100%"}
										alt={"Campaign background"}
										src={String(data?.campaigns[0].attributes?.imageUrl)}
									/>
								</Paper>

								<MarkDownRenderer
									body_md={String(data?.campaigns[0].attributes?.summary)}
									assetURL={appInfo.assetUrl}
								/>

								<MarkDownRenderer
									body_md={String(data?.campaigns[0].attributes?.thanksMsg)}
									assetURL={appInfo.assetUrl}
								/>
							</Stack>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<Box display="flex" justifyContent="center" alignItems="center" padding={2}>
								<Paper
									elevation={1}
									sx={{
										backgroundColor: theme.palette.primary.main,
										color: theme.palette.common.white,
										borderRadius: 0.5,
									}}
								>
									<Typography
										variant={"subtitle1"}
										textAlign={"center"}
										padding={2}
										textTransform={"uppercase"}
									>
										Patrons
									</Typography>
									<Typography
										variant={"h1"}
										textAlign={"center"}
										padding={2}
										sx={{ backgroundColor: theme.palette.primary.light }}
									>
										{String(data?.campaigns[0].attributes?.patronCount)}
									</Typography>
								</Paper>
							</Box>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<Box textAlign={"center"}>
								<Button
									component={Link}
									variant={"contained"}
									color={"success"}
									startIcon={<SearchIcon />}
									href={`${String(data?.campaigns[0].attributes?.url)}/membership`}
								>
									View Membership Tiers
								</Button>
							</Box>
						</Grid>
					</Grid>
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
}
