import Grid from "@mui/material/Grid";
import { createFileRoute } from "@tanstack/react-router";
import { HealersOverallContainer } from "../component/HealersOverallContainer.tsx";
import { MapUsageContainer } from "../component/MapUsageContainer.tsx";
import { PlayersOverallContainer } from "../component/PlayersOverallContainer.tsx";
import { WeaponsStatListContainer } from "../component/WeaponsStatListContainer.tsx";

export const Route = createFileRoute("/_auth/stats/")({
	component: Stats,
	loader: ({ context }) => ({
		appInfo: context.appInfo,
	}),
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Stats" }, match.context.title("Stats")],
	}),
});

function Stats() {
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<PlayersOverallContainer />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<HealersOverallContainer />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<WeaponsStatListContainer />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<MapUsageContainer />
			</Grid>
		</Grid>
	);
}
