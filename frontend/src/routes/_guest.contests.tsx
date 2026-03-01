import InsightsIcon from "@mui/icons-material/Insights";
import Grid from "@mui/material/Grid";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import {
	createColumnHelper,
	getCoreRowModel,
	getPaginationRowModel,
	type OnChangeFn,
	type PaginationState,
	useReactTable,
} from "@tanstack/react-table";
import { useState } from "react";
import { apiContests } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader";
import { TextLink } from "../component/TextLink.tsx";
import { DataTable } from "../component/table/DataTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { Contest } from "../schema/contest.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { RowsPerPage } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_guest/contests")({
	component: Contests,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.contests_enabled);
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Contests" }, match.context.title("Contests")],
	}),
});

function Contests() {
	const [pagination, setPagination] = useState({
		pageIndex: 0, //initial page index
		pageSize: RowsPerPage.TwentyFive, //default page size
	});

	const { data: contests, isLoading } = useQuery({
		queryKey: ["contests"],
		queryFn: async () => {
			return await apiContests();
		},
	});

	// const onEnter = useCallback(async (contest_id: string) => {
	//     try {
	//         await NiceModal.show(ModalContestEntry, { contest_id });
	//     } catch (e) {
	//         logErr(e);
	//     }
	// }, []);

	return (
		<Grid container>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Contests"} iconLeft={<InsightsIcon />}>
					<ContestsTable
						contests={contests ?? []}
						isLoading={isLoading}
						pagination={pagination}
						setPagination={setPagination}
					/>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}

const columnHelper = createColumnHelper<Contest>();

const ContestsTable = ({
	contests,
	isLoading,
	pagination,
	setPagination,
}: {
	contests: Contest[];
	isLoading: boolean;
	pagination?: PaginationState;
	setPagination?: OnChangeFn<PaginationState>;
}) => {
	const columns = [
		columnHelper.accessor("title", {
			header: "Title",
			size: 700,
			cell: (info) => {
				return (
					<TableCellSmall>
						<TextLink
							to={`/contests/$contest_id`}
							params={{ contest_id: info.row.original.contest_id as string }}
						>
							{info.getValue()}
						</TextLink>
					</TableCellSmall>
				);
			},
		}),
		columnHelper.accessor("num_entries", {
			header: "Entries",
			size: 75,
			cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
		}),
		columnHelper.accessor("date_start", {
			header: "Stared On",
			size: 140,
			cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>,
		}),
		columnHelper.accessor("date_end", {
			header: "Ends On",
			size: 140,
			cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>,
		}),
	];

	const table = useReactTable({
		data: contests,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: false,
		autoResetPageIndex: true,
		onPaginationChange: setPagination,
		getPaginationRowModel: getPaginationRowModel(),
		state: { pagination },
	});

	return <DataTable table={table} isLoading={isLoading} />;
};
