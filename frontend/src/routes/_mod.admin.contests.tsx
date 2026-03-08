import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import EditIcon from "@mui/icons-material/Edit";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import Button from "@mui/material/Button";
import IconButton from "@mui/material/IconButton";
import { createFileRoute } from "@tanstack/react-router";
import {
	createColumnHelper,
	getCoreRowModel,
	getPaginationRowModel,
	type OnChangeFn,
	type PaginationState,
	useReactTable,
} from "@tanstack/react-table";
import { useCallback, useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiContests } from "../api";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { PaginatorLocal } from "../component/forum/PaginatorLocal.tsx";
import { ContestEditor } from "../component/modal/ContestEditor.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { DataTable } from "../component/table/DataTable.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { Contest } from "../schema/contest.ts";
import { type PermissionLevelEnum, permissionLevelString } from "../schema/people.ts";
import { logErr } from "../util/errors.ts";
import { commonTableSearchSchema, initPagination } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";

const contestsSearchSchema = commonTableSearchSchema.extend({
	sortColumn: z.enum(["contest_id", "deleted"]).optional(),
	deleted: z.boolean().catch(false),
});

export const Route = createFileRoute("/_mod/admin/contests")({
	component: AdminContests,
	validateSearch: (search) => contestsSearchSchema.parse(search),
	loader: async ({ context }) => {
		const contests = await context.queryClient.fetchQuery({
			queryKey: ["adminContests"],
			queryFn: async () => {
				return await apiContests();
			},
		});
		return { contests };
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Contests" }, match.context.title("Contests")],
	}),
});

function AdminContests() {
	const search = Route.useSearch();
	const { contests } = Route.useLoaderData();
	const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));

	const onEditContest = useCallback(async (contest?: Contest) => {
		try {
			await NiceModal.show(ContestEditor, { contest });
		} catch (e) {
			logErr(e);
		}
	}, []);

	// const onDeleteContest = useCallback(
	//     async (contest_id: string) => {
	//         try {
	//             await apiContestDelete(contest_id);
	//             await modal.hide();
	//         } catch (e) {
	//             logErr(e);
	//             throw e;
	//         }
	//     },
	//     [modal]
	// );

	return (
		<ContainerWithHeaderAndButtons
			title={"User Submission Contests"}
			iconLeft={<EmojiEventsIcon />}
			buttons={[
				<Button
					key={"add-button"}
					startIcon={<AddIcon />}
					variant={"contained"}
					onClick={async () => {
						await onEditContest();
					}}
					color={"success"}
				>
					New Contest
				</Button>,
			]}
		>
			<ContestTable
				contests={contests ?? []}
				isLoading={false}
				onEdit={onEditContest}
				pagination={pagination}
				setPagination={setPagination}
			/>
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
				count={contests?.length ?? 0}
				rows={pagination.pageSize}
				page={pagination.pageIndex}
			/>
		</ContainerWithHeaderAndButtons>
	);
}

const columnHelper = createColumnHelper<Contest>();

const ContestTable = ({
	contests,
	isLoading,
	onEdit,
	pagination,
	setPagination,
}: {
	contests: Contest[];
	isLoading: boolean;
	onEdit: (person: Contest) => Promise<void>;
	pagination: PaginationState;
	setPagination: OnChangeFn<PaginationState>;
}) => {
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("title", {
				header: "Title",
				size: 200,
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("public", {
				header: "Public",
				size: 30,
				cell: (info) => <BoolCell enabled={info.getValue()} />,
			}),

			columnHelper.accessor("hide_submissions", {
				meta: { tooltip: "Are submissions hidden from public" },
				header: "Hide Sub.",
				size: 70,
				cell: (info) => <BoolCell enabled={info.getValue()} />,
			}),
			columnHelper.accessor("voting", {
				meta: { tooltip: "Is voting enabled on submissions" },
				header: "Voting",
				size: 70,
				cell: (info) => <BoolCell enabled={info.getValue()} />,
			}),
			columnHelper.accessor("down_votes", {
				meta: {
					tooltip: "Is down voting enabled. Required voting to be enabled",
				},
				header: "Down Votes",
				size: 30,
				cell: (info) => <BoolCell enabled={info.getValue()} />,
			}),
			columnHelper.accessor("max_submissions", {
				meta: { tooltip: "Max number of submissions a single user can make" },
				header: "Max Subs.",
				size: 100,
				cell: (info) => <TableCellString>{info.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("min_permission_level", {
				meta: { tooltip: "Minimum permission level required to participate" },
				header: "Min. Perms",
				size: 100,
				cell: (info) => (
					<TableCellString>{permissionLevelString(info.getValue() as PermissionLevelEnum)}</TableCellString>
				),
			}),

			columnHelper.accessor("date_start", {
				meta: { tooltip: "Start date" },
				header: "Starts",
				size: 150,
				cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.accessor("date_end", {
				meta: { tooltip: "End date" },
				header: "Ends",
				size: 150,
				cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				size: 150,
				cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>,
			}),
			columnHelper.display({
				id: "actions",
				size: 30,
				cell: (info) => {
					return (
						<IconButton color={"warning"} onClick={() => onEdit(info.row.original)}>
							<EditIcon />
						</IconButton>
					);
				},
			}),
		];
	}, [onEdit]);
	const table = useReactTable({
		data: contests,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		getPaginationRowModel: getPaginationRowModel(),
		onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
		state: {
			pagination,
		},
	});

	return <DataTable table={table} isLoading={isLoading} />;
};
