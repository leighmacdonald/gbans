/** biome-ignore-all lint/correctness/noChildrenProp: form needs it */
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetAnticheatLogs, apiGetServers } from "../api";
import { PersonCell } from "../component/PersonCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import type { StacEntry } from "../schema/anticheat.ts";
import { stringToColour } from "../util/colours.ts";
import { renderDate, renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_mod/admin/anticheat")({
	component: AdminAnticheat,
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: apiGetServers,
		});
		return {
			servers: unsorted.sort((a, b) => {
				if (a.server_name > b.server_name) {
					return 1;
				}
				if (a.server_name < b.server_name) {
					return -1;
				}
				return 0;
			}),
		};
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Anti-Cheat Logs" }, match.context.title("Anti-Cheat Logs")],
	}),
});

const columnHelper = createMRTColumnHelper<StacEntry>();
const defaultOptions = createDefaultTableOptions<StacEntry>();

function AdminAnticheat() {
	const search = Route.useSearch();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["anticheat", search],
		queryFn: async () => {
			try {
				return await apiGetAnticheatLogs({
					server_id: 0,
					name: "",
					summary: "",
					steam_id: "",
					detection: "any",
					limit: 100000,
					offset: 0,
					order_by: "created_on",
					desc: true,
				});
			} catch {
				return [];
			}
		},
	});

	const theme = useTheme();

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("anticheat_id", {
				header: "ID",
				enableSorting: false,
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("server_id", {
				enableSorting: false,
				grow: false,
				enableColumnFilter: false,
				header: "Server",
				Cell: ({ row }) => {
					return (
						<Button
							variant={"text"}
							sx={{
								color: stringToColour(row.original.server_name, theme.palette.mode),
							}}
						>
							{row.original.server_name}
						</Button>
					);
				},
			}),
			columnHelper.accessor("name", {
				header: "Name",
				enableHiding: false,
				grow: true,
				Cell: ({ row }) => (
					<PersonCell
						showCopy={true}
						steam_id={row.original.steam_id}
						personaname={row.original.personaname}
						avatar_hash={row.original.avatar}
					/>
				),
			}),
			columnHelper.accessor("personaname", {
				enableHiding: true,
				grow: false,
				header: "Personaname",
			}),
			columnHelper.accessor("steam_id", {
				enableHiding: true,
				grow: false,
				header: "Steam ID",
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => (
					<TableCellString title={renderDateTime(cell.getValue())}>
						{renderDate(cell.getValue())}
					</TableCellString>
				),
			}),
			columnHelper.accessor("demo_id", {
				header: "Demo",
				grow: false,
				Cell: ({ cell }) => <Typography>{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("detection", {
				header: "Detection",
				filterVariant: "multi-select",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("triggered", {
				header: "Count",
				filterVariant: "range-slider",
				grow: false,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
			columnHelper.accessor("summary", {
				header: "Summary",
				grow: true,
				Cell: ({ cell }) => <TableCellString>{cell.getValue()}</TableCellString>,
			}),
		];
	}, [theme]);

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
			sorting: [{ id: "created_on", desc: true }],
			columnVisibility: {
				anticheat_id: false,
				server_id: true,
				name: true,
				personaname: false,
				target_id: false,
				steam_id: false,
				demo_id: false,
				reason: true,
				reason_text: true,
				created_on: false,
			},
		},
	});

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Entries"} />
			</Grid>
		</Grid>
	);
}
