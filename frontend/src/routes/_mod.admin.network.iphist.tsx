import FilterListIcon from "@mui/icons-material/FilterList";
import SensorOccupiedIcon from "@mui/icons-material/SensorOccupied";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { z } from "zod/v4";
import { apiGetConnections } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader";
import { Paginator } from "../component/forum/Paginator.tsx";
import { IPHistoryTable } from "../component/table/IPHistoryTable.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { commonTableSearchSchema, RowsPerPage } from "../util/table.ts";
import { emptyOrNullString } from "../util/types.ts";

const ipHistorySearchSchema = commonTableSearchSchema.extend({
	sortColumn: z.enum(["person_connection_id", "steam_id", "created_on", "server_id"]).optional(),
	steam_id: z.string().optional().default(""),
});

export const Route = createFileRoute("/_mod/admin/network/iphist")({
	component: AdminNetworkPlayerIPHistory,
	validateSearch: (search) => ipHistorySearchSchema.parse(search),
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Player IP History" }, match.context.title("Player IP History")],
	}),
});

const schema = z.object({
	steam_id: z.string(),
});

function AdminNetworkPlayerIPHistory() {
	const navigate = useNavigate({ from: Route.fullPath });
	const { pageIndex, pageSize, sortOrder, sortColumn, steam_id } = Route.useSearch();
	const defaultValues: z.input<typeof schema> = {
		steam_id: steam_id ?? "",
	};

	const { data: connections, isLoading } = useQuery({
		queryKey: ["connectionHist", { pageIndex, pageSize, sortOrder, sortColumn, steam_id }],
		queryFn: async () => {
			if (emptyOrNullString(steam_id)) {
				return { data: [], count: 0 };
			}
			return await apiGetConnections({
				limit: pageSize,
				offset: (pageIndex ?? 0) * (pageSize ?? RowsPerPage.TwentyFive),
				order_by: sortColumn ?? "steam_id",
				desc: (sortOrder ?? "desc") === "desc",
				sid64: steam_id,
			});
		},
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({
				to: "/admin/network/iphist",
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onSubmit: schema,
		},
		defaultValues,
	});

	const clear = async () => {
		await navigate({
			to: "/admin/network/iphist",
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
									name={"steam_id"}
									children={(field) => {
										return <field.SteamIDField />;
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
				<ContainerWithHeader title="Player IP History" iconLeft={<SensorOccupiedIcon />}>
					<IPHistoryTable connections={connections ?? { data: [], count: 0 }} isLoading={isLoading} />
					<Paginator
						page={pageIndex ?? 0}
						rows={pageSize ?? RowsPerPage.TwentyFive}
						data={connections}
						path={"/admin/network/iphist"}
					/>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
