/** biome-ignore-all lint/correctness/noChildrenProp: <explanation> */
import { ChevronLeft, CloudDownload } from "@mui/icons-material";
import FilterListIcon from "@mui/icons-material/FilterList";
import FlagIcon from "@mui/icons-material/Flag";
import VideocamIcon from "@mui/icons-material/Videocam";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import FormControl from "@mui/material/FormControl";
import Grid from "@mui/material/Grid";
import InputLabel from "@mui/material/InputLabel";
import Link from "@mui/material/Link";
import MenuItem from "@mui/material/MenuItem";
import Select from "@mui/material/Select";
import { useTheme } from "@mui/material/styles";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import {
	type ColumnFiltersState,
	createColumnHelper,
	type SortingState,
} from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiGetDemos, apiGetServers } from "../api";
import { ButtonLink } from "../component/ButtonLink.tsx";
import { ContainerWithHeader } from "../component/ContainerWithHeader";
import { FullTable } from "../component/table/FullTable.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import type { DemoFile } from "../schema/demo.ts";
import type { ServerSimple } from "../schema/server.ts";
import { stringToColour } from "../util/colours.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import {
	commonTableSearchSchema,
	initColumnFilter,
	initPagination,
} from "../util/table.ts";
import { humanFileSize } from "../util/text.tsx";
import { renderDate, renderDateTime } from "../util/time.ts";

const demosSchema = commonTableSearchSchema.extend({
	sortColumn: z
		.enum(["demo_id", "server_id", "created_on", "map_name"])
		.optional(),
	map_name: z.string().optional(),
	server_id: z.number().optional(),
	stats: z.string().optional(),
});

export const Route = createFileRoute("/_guest/stv")({
	component: STV,
	validateSearch: (search) => demosSchema.parse(search),
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.demos_enabled);
	},
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
		meta: [
			{
				name: "description",
				content: "Search and download SourceTV recordings",
			},
			match.context.title("SourceTV"),
		],
	}),
});

const schema = z.object({
	map_name: z.string(),
	server_id: z.number(),
	stats: z.string(),
});

function STV() {
	const navigate = useNavigate({ from: Route.fullPath });
	const search = Route.useSearch();
	const { servers } = Route.useLoaderData();
	const { profile, isAuthenticated } = useAuth();
	const [pagination, setPagination] = useState(
		initPagination(search.pageIndex, search.pageSize),
	);
	const [sorting] = useState<SortingState>([{ id: "demo_id", desc: true }]);
	const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(
		initColumnFilter(search),
	);
	const theme = useTheme();

	const defaultValues: z.infer<typeof schema> = {
		map_name: search.map_name ?? "",
		server_id: search.server_id ?? 0,
		stats: search.stats ?? "",
	};

	const { data: demos, isLoading } = useQuery({
		queryKey: ["demos"],
		queryFn: apiGetDemos,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			setColumnFilters(initColumnFilter(value));
			await navigate({ search: (prev) => ({ ...prev, ...value }) });
		},
		validators: {
			onChange: schema,
		},
		defaultValues,
	});

	const clear = async () => {
		setColumnFilters([]);
		form.reset();
		await navigate({
			search: (prev) => ({
				...prev,
				map_name: undefined,
				server_id: undefined,
				stats: undefined,
			}),
		});
	};

	const columnHelper = createColumnHelper<DemoFile>();

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("demo_id", {
				header: "ID",
				size: 40,
				cell: (info) => <Typography>#{info.getValue()}</Typography>,
			}),
			columnHelper.accessor("server_id", {
				filterFn: (row, _, filterValue) => {
					return filterValue === 0 || row.original.server_id === filterValue;
				},
				size: 75,
				enableSorting: true,
				enableColumnFilter: true,
				header: "Server",
				cell: (info) => {
					return (
						<Button
							sx={{
								color: stringToColour(
									info.row.original.server_name_short,
									theme.palette.mode,
								),
							}}
							onClick={async () => {
								await navigate({
									search: (prev) => ({
										...prev,
										server_id: info.row.original.server_id,
									}),
								});
								await form.handleSubmit();
							}}
						>
							{info.row.original.server_name_short}
						</Button>
					);
				},
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				size: 140,
				cell: (info) => (
					<Typography>{renderDate(info.getValue() as Date)}</Typography>
				),
			}),
			columnHelper.accessor("map_name", {
				enableColumnFilter: true,
				header: "Map Name",
				size: 450,
				cell: (info) => <Typography>{info.getValue() as string}</Typography>,
			}),
			columnHelper.accessor("size", {
				header: "Size",
				size: 60,
				cell: (info) => (
					<Typography>{humanFileSize(info.getValue() as number)}</Typography>
				),
			}),
			columnHelper.accessor("stats", {
				enableColumnFilter: true,
				filterFn: (row, _, filterValue) => {
					return (
						filterValue === "" ||
						Object.keys(row.original.stats).includes(filterValue)
					);
				},
				header: "Players",
				size: 60,
				cell: (info) => (
					<Typography>{Object.keys(Object(info.getValue())).length}</Typography>
				),
			}),

			columnHelper.display({
				id: "report",
				size: 60,
				cell: (info) => (
					<ButtonLink
						disabled={!isAuthenticated()}
						color={"error"}
						variant={"text"}
						to={"/report"}
						search={{ demo_id: info.row.original.demo_id }}
					>
						<FlagIcon />
					</ButtonLink>
				),
			}),
			columnHelper.display({
				id: "download",
				size: 60,
				cell: (info) => (
					<Button
						color={"success"}
						component={Link}
						variant={"text"}
						href={`/asset/${info.row.original.asset_id}`}
					>
						<CloudDownload />
					</Button>
				),
			}),
		];
	}, [
		columnHelper,
		form.handleSubmit,
		isAuthenticated,
		navigate,
		theme.palette.mode,
	]);

	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader
					title={"Filters"}
					iconLeft={<FilterListIcon />}
					marginTop={2}
				>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Grid container spacing={2}>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"server_id"}
									children={({ state, handleChange, handleBlur }) => {
										return (
											<FormControl fullWidth>
												<InputLabel id="server-select-label">
													Servers
												</InputLabel>
												<Select
													fullWidth
													value={state.value}
													variant={"outlined"}
													label="Servers"
													onChange={(e) => {
														handleChange(Number(e.target.value));
													}}
													onBlur={handleBlur}
												>
													<MenuItem value={0}>All</MenuItem>
													{servers.map((s: ServerSimple) => (
														<MenuItem value={s.server_id} key={s.server_id}>
															{s.server_name}
														</MenuItem>
													))}
												</Select>
											</FormControl>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"map_name"}
									children={(field) => {
										return <field.TextField label={"Map Name"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"stats"}
									children={(field) => {
										return <field.TextField label={"Steam ID"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }} padding={2}>
								<Button
									fullWidth
									disabled={!isAuthenticated()}
									startIcon={<ChevronLeft />}
									variant={"contained"}
									onClick={async () => {
										await navigate({
											search: (prev) => ({ ...prev, stats: profile.steam_id }),
										});
										await form.handleSubmit();
									}}
								>
									My SteamID
								</Button>
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
				<ContainerWithHeader
					title={"SourceTV Recordings"}
					iconLeft={<VideocamIcon />}
				>
					<FullTable
						columnFilters={columnFilters}
						pagination={pagination}
						setPagination={setPagination}
						data={demos ?? []}
						isLoading={isLoading}
						columns={columns}
						sorting={sorting}
						toOptions={{ to: Route.fullPath }}
					/>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
