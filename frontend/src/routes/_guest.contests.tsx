import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiContests } from "../api";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { Contest } from "../schema/contest.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Contest>();
const defaultOptions = createDefaultTableOptions<Contest>();

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
	const { data, isLoading, isError } = useQuery({
		queryKey: ["contests"],
		queryFn: async () => {
			return await apiContests();
		},
	});

	const columns = useMemo(
		() => [
			columnHelper.accessor("title", {
				header: "Title",
				enableSorting: false,
				grow: true,
				Cell: ({ row, cell }) => {
					return (
						<TableCellSmall>
							<TextLink
								to={`/contests/$contest_id`}
								params={{ contest_id: row.original.contest_id as string }}
							>
								{cell.getValue()}
							</TextLink>
						</TableCellSmall>
					);
				},
			}),
			columnHelper.accessor("num_entries", {
				header: "Entries",
				enableSorting: false,
				grow: false,
				filterVariant: "range-slider",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("date_start", {
				header: "Started On",
				filterVariant: "date",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
			columnHelper.accessor("date_end", {
				header: "Ends On",
				filterVariant: "date",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "ends_on", desc: true }],
		},
	});

	return <SortableTable table={table} title={"Contests"} />;
}
