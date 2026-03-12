import Grid from "@mui/material/Grid";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetReports } from "../api";
import { PersonCell } from "../component/PersonCell.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { BanReason, BanReasons } from "../schema/bans.ts";
import { ReportStatus, type ReportWithAuthor, reportStatusString } from "../schema/report.ts";

const columnHelper = createMRTColumnHelper<ReportWithAuthor>();
const defaultOptions = createDefaultTableOptions<ReportWithAuthor>();

export const Route = createFileRoute("/_mod/admin/reports")({
	component: AdminReports,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Reports" }, match.context.title("Reports")],
	}),
});

function AdminReports() {
	const { data, isLoading, isError } = useQuery({
		queryKey: ["adminReports"],
		queryFn: async () => {
			return apiGetReports({ deleted: false });
		},
	});

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("report_id", {
				header: "ID",
				size: 30,
				Cell: ({ cell }) => (
					<TextLink
						color={"primary"}
						to={`/report/$reportId`}
						params={{ reportId: String(cell.getValue()) }}
						marginRight={2}
					>
						#{cell.getValue()}
					</TextLink>
				),
			}),
			columnHelper.accessor("report_status", {
				header: "Status",
				size: 150,
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(ReportStatus).map((status) => ({
					label: reportStatusString(status),
					value: status,
				})),
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.report_status);
				},
				Cell: ({ cell }) => {
					return (
						<Stack direction={"row"} spacing={1}>
							<Typography variant={"body1"}>{reportStatusString(cell.getValue())}</Typography>
						</Stack>
					);
				},
			}),
			columnHelper.accessor("source_id", {
				header: "Reporter",
				grow: true,
				enableColumnFilter: true,
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
						steam_id={row.original.author.steam_id}
						personaname={row.original.author.name}
						avatar_hash={row.original.author.avatarhash}
					/>
				),
			}),
			columnHelper.accessor("target_id", {
				header: "Subject",
				grow: true,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					const query = filterValue.toLowerCase();
					if (query === "") {
						return true;
					}
					const value = row.original.target_id.toLowerCase();
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
						steam_id={row.original.subject.steam_id}
						personaname={row.original.subject.name}
						avatar_hash={row.original.subject.avatarhash}
					/>
				),
			}),
			columnHelper.accessor("reason", {
				filterSelectOptions: Object.values(BanReason).map((reason) => ({
					label: BanReasons[reason],
					value: reason,
				})),
				filterVariant: "multi-select",
				header: "Reason",
				size: 150,
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
				filterVariant: "text",
				header: "Custom Reason",
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				size: 150,
				filterVariant: "date",
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				size: 150,
				filterVariant: "date",
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		enableFilters: true,
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
			columnVisibility: {
				source_id: false,
				target_id: true,
				reason: true,
				reason_text: false,
				created_on: false,
				report_status: true,
				updated_on: true,
				report_id: true,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"User Reports"} />
			</Grid>
		</Grid>
	);
}
