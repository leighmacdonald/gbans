import CategoryIcon from "@mui/icons-material/Category";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import GroupAddIcon from "@mui/icons-material/GroupAdd";
import TimerIcon from "@mui/icons-material/Timer";
import WallpaperIcon from "@mui/icons-material/Wallpaper";
import Grid from "@mui/material/Grid";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createColumnHelper,
	getCoreRowModel,
	getPaginationRowModel,
	type TableOptions,
	useReactTable,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { getSpeedrun } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { PaginatorLocal } from "../component/forum/PaginatorLocal.tsx";
import { LoadingPlaceholder } from "../component/LoadingPlaceholder.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { DataTable } from "../component/table/DataTable.tsx";
import type { SpeedrunParticipant, SpeedrunPointCaptures } from "../schema/speedrun.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { RowsPerPage } from "../util/table.ts";
import { durationString, renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_guest/speedruns/id/$speedrunId")({
	component: SpeedrunDetail,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Speedrun details" }, match.context.title("Speedrun")],
	}),
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.speedruns_enabled);
	},
});

function SpeedrunDetail() {
	const { speedrunId } = Route.useParams();

	const { data: speedrun, isLoading } = useQuery({
		queryKey: ["speedrun", speedrunId],
		queryFn: async () => {
			return await getSpeedrun(Number(speedrunId));
		},
	});

	const sortedSpeedruns = useMemo(() => {
		return (
			speedrun?.players.sort((a, b) => {
				return a.duration > b.duration ? -1 : 1;
			}) ?? []
		);
	}, [speedrun]);

	return (
		<>
			{isLoading && <LoadingPlaceholder height={400} />}

			{!isLoading && speedrun && (
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
						<ContainerWithHeader title={"Players"} iconLeft={<GroupAddIcon />}>
							<SpeedrunPlayerTable
								captures={speedrun.point_captures}
								players={sortedSpeedruns}
								isLoading={isLoading}
							></SpeedrunPlayerTable>
						</ContainerWithHeader>
					</Grid>
				</Grid>
			)}
		</>
	);
}

const columnHelper = createColumnHelper<SpeedrunParticipant>();

const SpeedrunPlayerTable = ({
	captures,
	players,
	isLoading,
}: {
	captures: SpeedrunPointCaptures[];
	players: SpeedrunParticipant[];
	isLoading: boolean;
}) => {
	const [pagination, setPagination] = useState({
		pageIndex: 0,
		pageSize: RowsPerPage.TwentyFive,
	});
	const columns = [
		columnHelper.accessor("steam_id", {
			header: "Player",
			size: 10,
			cell: (info) => {
				return (
					<PersonCell
						steam_id={info.row.original.steam_id}
						personaname={info.row.original.persona_name}
						avatar_hash={info.row.original.avatar_hash}
					/>
				);
			},
		}),

		columnHelper.accessor("duration", {
			header: "Time",
			size: 60,
			cell: (info) => (
				<TableCell>
					<Typography align={"center"}>{durationString(info.getValue() / 1000)}</Typography>
				</TableCell>
			),
		}),
		columnHelper.accessor("kills", {
			header: "Kills",
			size: 60,
			cell: (info) => (
				<TableCell>
					<Typography align={"center"}>{info.getValue() ?? 0}</Typography>
				</TableCell>
			),
		}),
		columnHelper.accessor("destructions", {
			header: "Destructions",
			size: 60,
			cell: (info) => (
				<TableCell>
					<Typography align={"center"}>{info.getValue() ?? 0}</Typography>
				</TableCell>
			),
		}),
		columnHelper.display({
			header: "Captures",
			size: 60,
			cell: () => {
				return (
					<TableCell>
						<Typography align={"center"}>
							{captures.map((c) => c.players.filter((x) => x.steam_id)).length}
						</Typography>
					</TableCell>
				);
			},
		}),
	];

	const opts: TableOptions<SpeedrunParticipant> = {
		data: players,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: false,
		autoResetPageIndex: true,
		onPaginationChange: setPagination,
		getPaginationRowModel: getPaginationRowModel(),
		state: { pagination },
	};

	const table = useReactTable(opts);

	return (
		<>
			<DataTable table={table} isLoading={isLoading} />
			<PaginatorLocal
				onRowsChange={(rows) => {
					setPagination((prev) => {
						return { ...prev, pageSize: rows };
					});
				}}
				onPageChange={(page) => {
					setPagination((prev) => {
						return { ...prev, pageIndex: page };
					});
				}}
				count={players.length}
				rows={pagination.pageSize}
				page={pagination.pageIndex}
			/>
		</>
	);
};
