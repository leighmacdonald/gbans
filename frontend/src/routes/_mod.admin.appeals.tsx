import Grid from "@mui/material/Grid";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetAppeals, appealStateString } from "../api";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { AppealState, BanReason, BanReasons, type BanRecord } from "../schema/bans.ts";

const columnHelper = createMRTColumnHelper<BanRecord>();
const defaultOptions = createDefaultTableOptions<BanRecord>();

export const Route = createFileRoute("/_mod/admin/appeals")({
	component: AdminAppeals,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Appeals" }, match.context.title("Appeals")],
	}),
});

function AdminAppeals() {
	const { data, isLoading, isError } = useQuery({
		queryKey: ["appeals"],
		queryFn: async () => {
			return (await apiGetAppeals({})) ?? [];
		},
	});

	const columns = useMemo(
		() => [
			columnHelper.accessor("ban_id", {
				header: "ID",
				size: 75,
				grow: false,
				Cell: ({ cell }) => (
					<TextLink
						color={"primary"}
						to={`/ban/$ban_id`}
						params={{ ban_id: String(cell.getValue()) }}
						marginRight={2}
					>
						#{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("appeal_state", {
				header: "Status",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(AppealState).map((reason) => ({
					label: appealStateString(reason),
					value: reason,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(AppealState.Any) ||
						filterValue.includes(row.original.appeal_state)
					);
				},
				Cell: ({ cell }) => {
					return <Typography variant={"body1"}>{appealStateString(cell.getValue())}</Typography>;
				},
			}),
			columnHelper.accessor("source_id", {
				header: "Author",
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.source_id.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.source_id.includes(query) || row.original.source_id === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.source_id}
						personaname={row.original.source_personaname}
						avatar_hash={row.original.source_avatarhash}
					/>
				),
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				enableColumnFilter: true,
				grow: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.target_personaname.toLowerCase();
					if (value.includes(query)) {
						return true;
					}
					if (row.original.target_id.includes(query) || row.original.target_id === query) {
						return true;
					}

					return false;
				},
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.target_id}
						personaname={row.original.target_personaname}
						avatar_hash={row.original.target_avatarhash}
					/>
				),
			}),
			columnHelper.accessor("reason", {
				filterVariant: "multi-select",
				header: "Reason",
				size: 150,
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				filterFn: (row, _, filterValue) => {
					return (
						filterValue.length === 0 ||
						filterValue.includes(BanReason.Any) ||
						filterValue.includes(row.original.reason)
					);
				},
				Cell: ({ cell }) => <Typography>{BanReasons[cell.getValue()]}</Typography>,
			}),
			columnHelper.accessor("reason_text", {
				header: "Custom",
				filterVariant: "text",
				grow: true,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				filterVariant: "date",
				size: 120,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
			columnHelper.accessor("updated_on", {
				header: "Last Active",
				enableColumnFilter: false,
				size: 120,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
		],
		[],
	);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				reason_text: true,
				created_on: false,
				updated_on: true,
			},
		},
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Ban Appeals"} />
			</Grid>
		</Grid>
	);
}
