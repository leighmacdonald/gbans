import InsightsIcon from "@mui/icons-material/Insights";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from "@tanstack/react-table";
import { useState } from "react";
import { apiGetPlayersOverall } from "../api";
import type { PlayerWeaponStats } from "../schema/stats.ts";
import { type LazyResult, RowsPerPage } from "../util/table.ts";
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from "../util/text.tsx";
import { ContainerWithHeader } from "./ContainerWithHeader";
import { FmtWhenGt } from "./FmtWhenGT.tsx";
import { PaginatorLocal } from "./forum/PaginatorLocal.tsx";
import { PersonCell } from "./PersonCell.tsx";
import { DataTable } from "./table/DataTable.tsx";
import { TableCellSmall } from "./table/TableCellSmall.tsx";

export const PlayersOverallContainer = () => {
	const { data, isLoading } = useQuery({
		queryKey: ["statsWeaponOverall"],
		queryFn: async () => {
			return await apiGetPlayersOverall();
		},
	});

	return (
		<ContainerWithHeader title={"Top 1000 Players By Kills"} iconLeft={<InsightsIcon />}>
			<StatsKillsOverall stats={data ?? { data: [], count: 0 }} isLoading={isLoading} />
		</ContainerWithHeader>
	);
};

const columnHelper = createColumnHelper<PlayerWeaponStats>();

const StatsKillsOverall = ({ stats, isLoading }: { stats: LazyResult<PlayerWeaponStats>; isLoading: boolean }) => {
	const [pagination, setPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const columns = [
		columnHelper.accessor("rank", {
			header: "#",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{info.getValue()}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("personaname", {
			header: "Name",
			cell: (info) => (
				<TableCellSmall>
					<PersonCell
						steam_id={stats.data[info.row.index].steam_id}
						personaname={stats.data[info.row.index].personaname}
						avatar_hash={stats.data[info.row.index].avatar_hash}
					/>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("ka", {
			header: "KA",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("kills", {
			header: "K",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("assists", {
			header: "A",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("deaths", {
			header: "D",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("kad", {
			header: "KAD",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("shots", {
			header: "SHT",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("hits", {
			header: "HIT",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("accuracy", {
			header: "Acc%",
			cell: (info) => (
				<TableCellSmall>
					<Typography>
						{FmtWhenGt(stats.data[info.row.index].shots, () => defaultFloatFmtPct(info.getValue()))}
					</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("airshots", {
			header: "AS",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("backstabs", {
			header: "BS",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("headshots", {
			header: "HS",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("damage", {
			header: "DMG",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("dpm", {
			header: "DPM",
			cell: (info) => (
				<TableCellSmall>
					<Typography>
						{FmtWhenGt(stats.data[info.row.index].shots, () => defaultFloatFmt(info.getValue()))}
					</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("damage_taken", {
			header: "DT",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("dominations", {
			header: "DOM",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
		columnHelper.accessor("captures", {
			header: "CAP",
			cell: (info) => (
				<TableCellSmall>
					<Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
				</TableCellSmall>
			),
		}),
	];

	const table = useReactTable({
		data: stats.data,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
		state: {
			pagination,
		},
	});

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
				count={stats.count}
				rows={pagination.pageSize}
				page={pagination.pageIndex}
			/>
		</>
	);
};
