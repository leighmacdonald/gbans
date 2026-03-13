import CategoryIcon from "@mui/icons-material/Category";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import TimerIcon from "@mui/icons-material/Timer";
import WallpaperIcon from "@mui/icons-material/Wallpaper";
import Grid from "@mui/material/Grid";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { getSpeedrun } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import type { SpeedrunParticipant } from "../schema/speedrun.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { durationString, renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<SpeedrunParticipant>();
const defaultOptions = createDefaultTableOptions<SpeedrunParticipant>();

export const Route = createFileRoute("/_guest/speedruns/id/$speedrunId")({
	component: SpeedrunDetail,
	loader: async ({ context, params }) => {
		const { speedrunId } = params;
		const speedrun = await context.queryClient.fetchQuery({
			queryKey: ["speedrun", speedrunId],
			queryFn: async () => {
				return await getSpeedrun(Number(speedrunId));
			},
		});
		return { speedrun };
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Speedrun details" }, match.context.title("Speedrun")],
	}),
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.speedruns_enabled);
	},
});

function SpeedrunDetail() {
	const { speedrun } = Route.useLoaderData();

	const columns = useMemo(
		() => [
			columnHelper.accessor("steam_id", {
				header: "Player",
				size: 10,
				Cell: ({ row }) => {
					return (
						<PersonCell
							steam_id={row.original.steam_id}
							personaname={row.original.persona_name}
							avatar_hash={row.original.avatar_hash}
						/>
					);
				},
			}),

			columnHelper.accessor("duration", {
				header: "Time",
				size: 60,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography align={"center"}>{durationString(cell.getValue() / 1000)}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("kills", {
				header: "Kills",
				size: 60,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography align={"center"}>{cell.getValue() ?? 0}</Typography>
					</TableCell>
				),
			}),
			columnHelper.accessor("destructions", {
				header: "Destructions",
				size: 60,
				Cell: ({ cell }) => (
					<TableCell>
						<Typography align={"center"}>{cell.getValue() ?? 0}</Typography>
					</TableCell>
				),
			}),
			columnHelper.display({
				header: "Captures",
				size: 60,
				Cell: () => {
					return (
						<TableCell>
							<Typography align={"center"}>
								{speedrun.point_captures.map((c) => c.players.filter((x) => x.steam_id)).length}
							</Typography>
						</TableCell>
					);
				},
			}),
		],
		[speedrun],
	);
	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: speedrun.players,
		enableFilters: true,
		enableRowActions: true,
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "duration", desc: true }],
		},
	});

	return (
		<>
			{
				<Grid container spacing={2}>
					<Grid size={{ xs: 2 }}>
						<ContainerWithHeader title={"Rank (Initial)"} iconLeft={<EmojiEventsIcon />}>
							<Typography textAlign={"center"} fontSize={"large"} fontWeight={"bold"}>
								{speedrun.rank} ({speedrun.initial_rank})
							</Typography>
						</ContainerWithHeader>
					</Grid>
					<Grid size={{ xs: 2 }}>
						<ContainerWithHeader title={"Time"} iconLeft={<TimerIcon />}>
							<Typography textAlign={"center"} fontSize={"large"} fontWeight={"bold"}>
								{durationString((speedrun.duration ?? 0) / 1000)}
							</Typography>
						</ContainerWithHeader>
					</Grid>
					<Grid size={{ xs: 4 }}>
						<ContainerWithHeader title={"Map"} iconLeft={<WallpaperIcon />}>
							<Typography textAlign={"center"} fontSize={"large"} fontWeight={"bold"}>
								{speedrun.map_detail.map_name}
							</Typography>
						</ContainerWithHeader>
					</Grid>
					<Grid size={{ xs: 2 }}>
						<ContainerWithHeader title={"Category"} iconLeft={<CategoryIcon />}>
							<Typography textAlign={"center"} fontSize={"large"} fontWeight={"bold"}>
								{speedrun.category}
							</Typography>
						</ContainerWithHeader>
					</Grid>
					<Grid size={{ xs: 2 }}>
						<ContainerWithHeader title={"Submitted"} iconLeft={<CategoryIcon />}>
							<Typography textAlign={"center"} fontSize={"large"} fontWeight={"bold"}>
								{renderDateTime(speedrun.created_on)}
							</Typography>
						</ContainerWithHeader>
					</Grid>
					<Grid size={{ xs: 12 }}>
						<SortableTable table={table} title={"Players"} />
					</Grid>
				</Grid>
			}
		</>
	);
}
