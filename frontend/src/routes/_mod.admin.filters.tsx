import InfoIcon from "@mui/icons-material/Info";
import Grid from "@mui/material/Grid";
import TableCell from "@mui/material/TableCell";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiGetFilters, apiGetWarningState } from "../api/filters.ts";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellSmall } from "../component/table/TableCellSmall.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { type Filter, filterActionString, type UserWarning } from "../schema/filters.ts";
import { renderDateTime } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Filter>();
const defaultOptions = createDefaultTableOptions<Filter>();

export const Route = createFileRoute("/_mod/admin/filters")({
	component: AdminFilters,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Filtered Words" }, match.context.title("Filtered Words")],
	}),
});

function AdminFilters() {
	const { data, isLoading, isError } = useQuery({
		queryKey: ["filters"],
		queryFn: async () => {
			return await apiGetFilters();
		},
	});

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("pattern", {
				header: "Pattern",
				grow: true,
				enableColumnFilter: false,
				Cell: ({ cell }) => cell.getValue(),
			}),

			columnHelper.accessor("is_regex", {
				header: "Rx",
				filterVariant: "checkbox",
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),

			columnHelper.accessor("action", {
				header: "Action",
				enableColumnFilter: false,
				meta: { tooltip: "What action to take?" },
				grow: false,
				Cell: ({ cell }) => {
					return (
						<TableCellString>
							{typeof cell.getValue() === "undefined" ? "" : filterActionString(cell.getValue())}
						</TableCellString>
					);
				},
			}),

			columnHelper.accessor("duration", {
				header: "Duration",
				enableColumnFilter: false,
				grow: false,
				meta: { tooltip: "Duration of the punishment when triggered" },
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("weight", {
				grow: false,
				filterVariant: "range",
				header: "Weight",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("trigger_count", {
				header: "Trig #",
				enableColumnFilter: false,
				grow: false,
				meta: { tooltip: "Number of times the filter has been triggered" },
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
		];
	}, []);

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
				<SortableTable table={table} title={"Word Filters"} />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<WarningStateTable />
			</Grid>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"How it works"} iconLeft={<InfoIcon />}>
					<Typography variant={"body1"}>
						The way the warning tracking works is that each time a user triggers a match, it gets an entry
						in the table based on the weight of the match. The individual match weight is determined by the
						word filter defined above. Once the sum of their triggers exceeds the max weight the user will
						have action taken against them automatically. Matched entries are ephemeral and are removed over
						time based on the configured timeout value.
					</Typography>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}

const columnHelperWarn = createMRTColumnHelper<UserWarning>();
const defaultOptionsWarn = createDefaultTableOptions<UserWarning>();

export const WarningStateTable = () => {
	const { data, isLoading, isError } = useQuery({
		queryKey: ["filterWarnings"],
		queryFn: async () => {
			return await apiGetWarningState();
		},
	});
	const renderFilter = useCallback((f: Filter) => {
		const pat = f.is_regex ? (f.pattern as string) : (f.pattern as string);

		return (
			<>
				<Typography variant={"h6"}>Matched {f.is_regex ? "Regex" : "Text"}</Typography>
				<Typography variant={"body1"}>{pat}</Typography>
				<Typography variant={"body1"}>Weight: {f.weight}</Typography>
				<Typography variant={"body1"}>Action: {filterActionString(f.action)}</Typography>
			</>
		);
	}, []);

	const columns = useMemo(
		() => [
			columnHelperWarn.accessor("steam_id", {
				header: "Pattern",
				Cell: ({ row }) => (
					<TableCellSmall>
						<PersonCell
							steam_id={row.original.steam_id}
							personaname={row.original.personaname}
							avatar_hash={row.original.avatar}
						/>
					</TableCellSmall>
				),
			}),
			columnHelperWarn.accessor("created_on", {
				header: "Created",
				Cell: ({ cell }) => <TableCellString>{renderDateTime(cell.getValue())}</TableCellString>,
			}),
			columnHelperWarn.accessor("matched_filter.action", {
				header: "Action",
				Cell: ({ cell }) => (
					<TableCellSmall>
						<Typography>
							{typeof cell.getValue() === "undefined" ? "" : filterActionString(cell.getValue())}
						</Typography>
					</TableCellSmall>
				),
			}),
			columnHelperWarn.accessor("matched", {
				header: "Duration",
				Cell: ({ row, cell }) => (
					<TableCell>
						<Tooltip title={renderFilter(row.original as unknown as Filter)}>
							<Typography>{cell.getValue()}</Typography>
						</Tooltip>
					</TableCell>
				),
			}),
			columnHelperWarn.accessor("current_total", {
				header: "Weight",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelperWarn.accessor("message", {
				header: "Triggered",
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
		],
		[renderFilter],
	);

	const table = useMaterialReactTable({
		...defaultOptionsWarn,
		columns,
		data: data ? data.current : [],
		enableFilters: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptionsWarn.initialState,
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

	return <SortableTable table={table} title={`Current Warning State (Max Weight: ${data?.max_weight ?? "..."})`} />;
};
