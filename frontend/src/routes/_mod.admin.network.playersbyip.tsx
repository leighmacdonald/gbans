import FilterListIcon from "@mui/icons-material/FilterList";
import WifiFindIcon from "@mui/icons-material/WifiFind";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import TableCell from "@mui/material/TableCell";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { createColumnHelper, getCoreRowModel, useReactTable } from "@tanstack/react-table";
import { z } from "zod/v4";
import { apiGetConnections } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { Paginator } from "../component/forum/Paginator.tsx";
import { TextLink } from "../component/TextLink.tsx";
import { DataTable } from "../component/table/DataTable.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import type { PersonConnection } from "../schema/people.ts";
import { commonTableSearchSchema, type LazyResult, RowsPerPage } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";
import { emptyOrNullString } from "../util/types.ts";

const playersByIPSearchSchema = commonTableSearchSchema.extend({
	sortColumn: z.enum(["person_connection_id", "steam_id", "created_on", "ip_addr", "server_id"]).optional(),
	cidr: z.string().optional(),
});

export const Route = createFileRoute("/_mod/admin/network/playersbyip")({
	component: AdminNetworkPlayersByCIDR,
	head: () => ({
		meta: [{ name: "description", content: "Find players by IP address" }, { title: "Players By IP" }],
	}),
	validateSearch: (search) => playersByIPSearchSchema.parse(search),
});

function AdminNetworkPlayersByCIDR() {
	const defaultRows = RowsPerPage.TwentyFive;
	const navigate = useNavigate({ from: Route.fullPath });
	const { pageIndex, pageSize, sortOrder, sortColumn, cidr } = Route.useSearch();
	const { data: connections, isLoading } = useQuery({
		queryKey: ["playersByIP", { pageIndex, pageSize, sortOrder, sortColumn, cidr }],
		queryFn: async () => {
			if (emptyOrNullString(cidr)) {
				return { data: [], count: 0 };
			}
			return await apiGetConnections({
				limit: pageSize ?? defaultRows,
				offset: (pageIndex ?? 0) * (pageSize ?? defaultRows),
				order_by: sortColumn ?? "steam_id",
				desc: sortOrder === "desc",
				cidr: cidr ?? "",
			});
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({
				to: "/admin/network/playersbyip",
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onChange: z.object({
				cidr: z.string(),
			}),
		},
		defaultValues: {
			cidr: cidr ?? "",
		},
	});

	const clear = async () => {
		await navigate({
			to: "/admin/network/playersbyip",
			search: (prev) => ({ ...prev, source_id: undefined }),
		});
	};

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Filters"} iconLeft={<FilterListIcon />} marginTop={2}>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Grid container spacing={2}>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"cidr"}
									children={(field) => {
										return <field.TextField label={"CIDR/IP"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 12 }}>
								<form.AppForm>
									<ButtonGroup>
										<form.ClearButton onClick={clear} />
										<form.ResetButton />
										<form.SubmitButton />
									</ButtonGroup>
								</form.AppForm>
							</Grid>
						</Grid>
					</form>
				</ContainerWithHeader>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Find Players By IP/CIDR"} iconLeft={<WifiFindIcon />}>
					<PayersByIPTable connections={connections ?? { data: [], count: 0 }} isLoading={isLoading} />
					<Paginator
						page={pageIndex ?? 0}
						rows={pageSize ?? defaultRows}
						data={connections}
						path={"/admin/network/playersbyip"}
					/>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}

const columnHelper = createColumnHelper<PersonConnection>();

const PayersByIPTable = ({
	connections,
	isLoading,
}: {
	connections: LazyResult<PersonConnection>;
	isLoading: boolean;
}) => {
	const columns = [
		columnHelper.accessor("created_on", {
			size: 120,
			header: "Created",
			cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>,
		}),
		columnHelper.accessor("persona_name", {
			header: "Name",
			cell: (info) => (
				<TableCell>
					<Typography>{info.getValue()}</Typography>
				</TableCell>
			),
		}),
		columnHelper.accessor("steam_id", {
			size: 150,
			header: "Steam ID",
			cell: (info) => (
				<TableCell>
					<TextLink to={"/profile/$steamId"} params={{ steamId: info.getValue() }}>
						{info.getValue()}
					</TextLink>
				</TableCell>
			),
		}),
		columnHelper.accessor("ip_addr", {
			size: 120,
			header: "IP Address",
			cell: (info) => (
				<TableCell>
					<Typography>{info.getValue()}</Typography>
				</TableCell>
			),
		}),

		columnHelper.accessor("server_id", {
			header: "Server",
			size: 75,
			cell: (info) => (
				<TableCell>
					<Typography>{connections.data[info.row.index].server_name_short}</Typography>
				</TableCell>
			),
		}),
	];

	const table = useReactTable({
		data: connections.data,
		columns: columns,
		getCoreRowModel: getCoreRowModel(),
		manualPagination: true,
		autoResetPageIndex: true,
	});

	return <DataTable table={table} isLoading={isLoading} />;
};
