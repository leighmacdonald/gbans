/** biome-ignore-all lint/correctness/noChildrenProp: ts-form made me do it! */

import { CloudDownload } from "@mui/icons-material";
import FlagIcon from "@mui/icons-material/Flag";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import Link from "@mui/material/Link";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { apiGetDemos, apiGetServers } from "../api";
import { ButtonLink } from "../component/ButtonLink.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import type { DemoFile } from "../schema/demo.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { humanFileSize } from "../util/text.tsx";
export const Route = createFileRoute("/_guest/stv")({
	component: STV,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.demos_enabled);
	},
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: apiGetServers,
		});

		return {
			servers: unsorted.sort((a, b) =>
				a.server_name > b.server_name ? 1 : a.server_name < b.server_name ? -1 : 0,
			),
		};
	},
	head: ({ match }) => ({
		meta: [
			{
				name: "description",
				content: "Search and download SourceTV recordings",
			},
			match.context.title("SourceTV"),
		],
	}),
});

const columnHelper = createMRTColumnHelper<DemoFile>();
const defaultOptions = createDefaultTableOptions<DemoFile>();

function STV() {
	const { isAuthenticated } = useAuth();
	const theme = useTheme();

	const { data, isLoading, isError } = useQuery({
		queryKey: ["demos"],
		queryFn: apiGetDemos,
	});

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("demo_id", {
				header: "ID",
				grow: false,
				Cell: ({ cell }) => <Typography>#{cell.getValue()}</Typography>,
			}),
			columnHelper.accessor("server_id", {
				filterFn: (row, _, filterValue) => {
					return filterValue.length === 0 || filterValue.includes(row.original.server_id);
				},
				filterVariant: "multi-select",
				grow: false,
				enableSorting: true,
				enableColumnFilter: true,
				header: "Server",
				Cell: ({ row }) => {
					return (
						<Button
							variant="text"
							sx={{
								color: stringToColour(row.original.server_name_short, theme.palette.mode),
							}}
						>
							{row.original.server_name_short}
						</Button>
					);
				},
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				enableColumnFilter: false,
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} suffix />,
			}),
			columnHelper.accessor("map_name", {
				enableColumnFilter: true,
				header: "Map Name",
				grow: true,
				filterVariant: "multi-select",
				Cell: ({ cell }) => <Typography>{cell.getValue() as string}</Typography>,
			}),
			columnHelper.accessor("size", {
				header: "Size",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,
				Cell: ({ cell }) => <Typography>{humanFileSize(cell.getValue() as number)}</Typography>,
			}),
			columnHelper.accessor("stats", {
				header: "Players",
				grow: false,
				enableSorting: false,
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					return filterValue === "" || Object.keys(row.original.stats).includes(filterValue);
				},
				Cell: ({ cell }) => <Typography>{Object.keys(Object(cell.getValue())).length}</Typography>,
			}),
		];
	}, [theme.palette.mode]);

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
			sorting: [{ id: "created_on", desc: true }],
			columnVisibility: {
				demo_id: false,
				server_id: true,
				created_on: true,
			},
		},
		enableRowActions: true,
		renderTopToolbarCustomActions: () => {
			return <Typography variant="h3">Bans</Typography>;
		},
		renderRowActionMenuItems: ({ row }) => [
			<ButtonLink
				key={"report"}
				disabled={!isAuthenticated()}
				color={"error"}
				variant={"text"}
				to={"/report"}
				search={{ demo_id: row.original.demo_id }}
			>
				<FlagIcon />
			</ButtonLink>,
			<Button
				key={"dl-link"}
				color={"success"}
				component={Link}
				variant={"text"}
				href={`/asset/${row.original.asset_id}`}
			>
				<CloudDownload />
			</Button>,
		],
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"SourceTV Recordings"} />
			</Grid>
		</Grid>
	);
}
