import { useMutation, useQuery } from "@connectrpc/connect-query";
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
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { ImageBox } from "../component/ImageBox.tsx";
import { MarkDownRenderer } from "../component/MarkdownRenderer.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { patronCampaigns, patronLogin } from "../rpc/patreon/v1/patreon-PatreonService_connectquery.ts";
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
		meta: match.context.meta({ title: "Patreon", description: "Support the community on Patreon." }),
	}),
});

function Patreon() {
	const { isAuthenticated, profile } = useAuth();
	const { appInfo } = Route.useRouteContext();
	const { sendError } = useUserFlashCtx();
	const theme = useTheme();

	const { data, isLoading, isError, error } = useQuery(patronCampaigns);

	const loginMutation = useMutation(patronLogin, {
		onSuccess: (resp) => {
			window.open(resp.url, "_self");
		},
		onError: sendError,
	});

	const followCallback = async () => {
		loginMutation.mutate({});
	};

	const connectButton = (key: string) => (
		<Button
			key={key}
			variant={"contained"}
			color={"success"}
			disabled={!isAuthenticated() || profile.patreonId !== "" || loginMutation.isPending}
			onClick={followCallback}
			startIcon={<SettingsInputComponentIcon />}
		>
			Connect Patreon
		</Button>
	);

	if (!appInfo.patreonEnabled) {
		return <Navigate to={"/"} />;
	}

	if (isError) {
		return <ErrorDetails error={error} />;
	}

	if (isLoading) {
		return;
	}

	const campaign = data?.campaigns?.[0];
	const attributes = campaign?.attributes;

	if (!campaign || !attributes) {
		return (
			<Grid container spacing={2}>
				<Grid size={{ xs: 12 }}>
					<ContainerWithHeaderAndButtons
						title={"Patreon"}
						iconLeft={<PaymentIcon />}
						buttons={profile.patreonId ? [] : [connectButton("connect")]}
					>
						<Typography variant={"body1"} sx={{ padding: 2 }}>
							Campaign information is not available yet. Please check back later.
						</Typography>
					</ContainerWithHeaderAndButtons>
				</Grid>
			</Grid>
		);
	}

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeaderAndButtons
					title={`Patreon Campaign: ${attributes.creationName}`}
					iconLeft={<PaymentIcon />}
					buttons={profile.patreonId ? [] : [connectButton("connect")]}
				>
					<Grid container>
						<Grid size={{ xs: 12 }}>
							<Stack spacing={1}>
								<Paper>
									<ImageBox
										height={"100%"}
										width={"100%"}
										alt={"Campaign background"}
										src={String(attributes.imageUrl)}
									/>
								</Paper>

								<MarkDownRenderer bodyMd={String(attributes.summary)} assetURL={appInfo.assetUrl} />

								<MarkDownRenderer bodyMd={String(attributes.thanksMsg)} assetURL={appInfo.assetUrl} />
							</Stack>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<Box
								sx={{
									display: "flex",
									justifyContent: "center",
									alignItems: "center",
									padding: 2,
								}}
							>
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
										sx={{
											textAlign: "center",
											padding: 2,
											textTransform: "uppercase",
										}}
									>
										Patrons
									</Typography>
									<Typography
										variant={"h1"}
										sx={{
											textAlign: "center",
											padding: 2,
											backgroundColor: theme.palette.primary.light,
										}}
									>
										{String(attributes.patronCount)}
									</Typography>
								</Paper>
							</Box>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<Box
								sx={{
									textAlign: "center",
								}}
							>
								<Button
									component={Link}
									variant={"contained"}
									color={"success"}
									startIcon={<SearchIcon />}
									href={`${String(attributes.url)}/membership`}
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
