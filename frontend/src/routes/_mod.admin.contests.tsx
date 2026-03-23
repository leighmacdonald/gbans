import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiContests } from "../api";
import { ContestEditor } from "../component/modal/ContestEditor.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { Contest } from "../schema/contest.ts";
import { type PermissionLevelEnum, permissionLevelString } from "../schema/people.ts";
import { logErr } from "../util/errors.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Contest>();
const defaultOptions = createDefaultTableOptions<Contest>();

export const Route = createFileRoute("/_mod/admin/contests")({
	component: AdminContests,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Contests" }, match.context.title("Contests")],
	}),
});

function AdminContests() {
	const { data, isError, isLoading } = useQuery({
		queryKey: ["adminContests"],
		queryFn: async ({ signal }) => {
			return await apiContests(signal);
		},
	});

	const onEditContest = useCallback(async (contest?: Contest) => {
		try {
			await NiceModal.show(ContestEditor, { contest });
		} catch (e) {
			logErr(e);
		}
	}, []);

	const columns = useMemo(
		() => [
			columnHelper.accessor("title", {
				header: "Title",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("public", {
				header: "Public",
				grow: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),

			columnHelper.accessor("hide_submissions", {
				meta: { tooltip: "Are submissions hidden from public" },
				header: "Hide Sub.",
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("voting", {
				meta: { tooltip: "Is voting enabled on submissions" },
				header: "Voting",
				filterVariant: "checkbox",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("down_votes", {
				meta: {
					tooltip: "Is down voting enabled. Required voting to be enabled",
				},
				header: "Down Votes",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),
			columnHelper.accessor("max_submissions", {
				meta: { tooltip: "Max number of submissions a single user can make" },
				header: "Max Subs.",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("min_permission_level", {
				meta: { tooltip: "Minimum permission level required to participate" },
				header: "Min. Perms",
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => (
					<TableCellString>{permissionLevelString(cell.getValue() as PermissionLevelEnum)}</TableCellString>
				),
			}),
			columnHelper.accessor("date_start", {
				meta: { tooltip: "Start date" },
				header: "Starts",
				filterVariant: "date",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.accessor("date_end", {
				meta: { tooltip: "End date" },
				filterVariant: "date",
				header: "Ends",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				filterVariant: "date",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue() as Date)}</TableCellString>,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableHiding: true,
		enableFacetedValues: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "ban_id", desc: true }],
			columnVisibility: {
				title: true,
				max_submissions: false,
				down_votes: false,
				hide_submissions: false,
				voting: true,
				min_permission_level: false,
				date_start: true,
				date_end: true,
				created_on: false,
				updated_on: false,
			},
		},
		enableRowActions: true,
		renderRowActionMenuItems: ({ row }) => [
			<IconButton color={"warning"} onClick={() => onEditContest(row.original)} key={row.id}>
				<EditIcon />
			</IconButton>,
		],
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"User Submission Contests"}
					buttons={[
						<IconButton
							key={"add-button"}
							sx={{ color: "primary.contrastText" }}
							onClick={async () => {
								await onEditContest();
							}}
						>
							<AddIcon />
						</IconButton>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
