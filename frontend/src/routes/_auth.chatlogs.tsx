import ChatIcon from "@mui/icons-material/Chat";
import FilterAltIcon from "@mui/icons-material/FilterAlt";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useLoaderData, useNavigate } from "@tanstack/react-router";
import { z } from "zod/v4";
import { apiGetMessages, apiGetServers } from "../api";
import { ContainerWithHeader } from "../component/ContainerWithHeader.tsx";
import { Paginator } from "../component/forum/Paginator.tsx";
import { ChatTable } from "../component/table/ChatTable.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { PermissionLevel } from "../schema/people.ts";
import type { ServerSimple } from "../schema/server.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { commonTableSearchSchema, RowsPerPage } from "../util/table.ts";

const searchSchema = commonTableSearchSchema.extend({
	sort_column: z
		.enum([
			"person_message_id",
			"steam_id",
			"persona_name",
			"server_name",
			"server_id",
			"team",
			"created_on",
			"pattern",
			"auto_filter_flagged",
		])
		.optional(),
	server_id: z.number().optional(),
	persona_name: z.string().optional(),
	body: z.string().optional(),
	steam_id: z.string().optional(),
	flagged_only: z.boolean().optional(),
	auto_refresh: z.number().optional(),
});

export const Route = createFileRoute("/_auth/chatlogs")({
	component: ChatLogs,
	validateSearch: (search) => searchSchema.parse(search),
	head: () => ({
		meta: [{ name: "description", content: "Browse in-game chat logs" }, { title: "Chat Logs" }],
	}),
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogs_enabled);
	},
	loader: async ({ context }) => {
		const unsorted = await context.queryClient.ensureQueryData({
			queryKey: ["serversSimple"],
			queryFn: apiGetServers,
		});
		return {
			appInfo: context.appInfo,
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
});

const schema = z.object({
	server_id: z.number(),
	persona_name: z.string(),
	body: z.string(),
	steam_id: z.string(),
	flagged_only: z.boolean(),
	auto_refresh: z.number(),
});

function ChatLogs() {
	const defaultRows = RowsPerPage.TwentyFive;
	const search = Route.useSearch();
	const { hasPermission } = useAuth();
	const { servers } = useLoaderData({ from: "/_auth/chatlogs" }) as {
		servers: ServerSimple[];
	};
	const navigate = useNavigate({ from: Route.fullPath });

	const defaultValues: z.input<typeof schema> = {
		body: search.body ?? "",
		persona_name: search.persona_name ?? "",
		server_id: search.server_id ?? 0,
		steam_id: search.steam_id ?? "",
		flagged_only: search.flagged_only ?? false,
		auto_refresh: search.auto_refresh ?? 0,
	};

	const { data: messages, isLoading } = useQuery({
		queryKey: ["chatlogs", { search }],
		queryFn: async () => {
			return await apiGetMessages({
				server_id: search.server_id,
				personaname: search.persona_name,
				query: search.body,
				source_id: search.steam_id,
				limit: search.pageSize ?? defaultRows,
				offset: (search.pageIndex ?? 0) * (search.pageSize ?? defaultRows),
				order_by: "person_message_id",
				desc: (search.sortOrder ?? "desc") === "desc",
				flagged_only: search.flagged_only ?? false,
			});
		},
		refetchInterval: search.auto_refresh,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({
				to: "/chatlogs",
				search: (prev) => ({ ...prev, ...value }),
			});
		},
		validators: {
			onMount: schema,
			onChangeAsyncDebounceMs: 500,
			onChangeAsync: schema,
		},
		defaultValues,
	});

	const clear = async () => {
		form.reset();
		await navigate({
			to: "/chatlogs",
			search: (prev) => ({
				...prev,
				body: undefined,
				persona_name: undefined,
				server_id: undefined,
				steam_id: undefined,
				flagged_only: undefined,
				autoRefresh: undefined,
			}),
		});
		await form.handleSubmit();
	};
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeader title={"Chat Filters"} iconLeft={<FilterAltIcon />}>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Grid container padding={2} spacing={2} justifyContent={"center"} alignItems={"center"}>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"persona_name"}
									children={(field) => {
										return <field.TextField label={"Name"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"steam_id"}
									children={(field) => {
										return <field.SteamIDField />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"body"}
									children={(field) => {
										return <field.TextField label={"Message"} />;
									}}
								/>
							</Grid>

							<Grid size={{ xs: 6, md: 3 }}>
								<form.AppField
									name={"server_id"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Servers"}
												items={servers}
												renderItem={(s) => {
													return (
														<MenuItem value={s.server_id} key={s.server_id}>
															{s.server_name}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
							{hasPermission(PermissionLevel.Moderator) && (
								<>
									<Grid size={{ xs: "auto" }}>
										<form.AppField
											name={"flagged_only"}
											children={(field) => {
												return <field.CheckboxField label={"Flagged Only"} />;
											}}
										/>
									</Grid>
									<Grid size={{ xs: "auto" }}>
										<form.AppField
											name={"auto_refresh"}
											children={(field) => {
												return (
													<field.SelectField
														label={"Action"}
														items={[0, 10000, 30000, 60000, 300000]}
														renderItem={(fa) => {
															return (
																<MenuItem value={fa} key={`fa-${fa}`}>
																	{fa / 1000} secs
																</MenuItem>
															);
														}}
													/>
												);
											}}
										/>
									</Grid>
								</>
							)}

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
				<ContainerWithHeader iconLeft={<ChatIcon />} title={"Chat Logs"}>
					<ChatTable messages={messages ?? []} isLoading={isLoading} />
					<Paginator
						page={search.pageIndex ?? 0}
						rows={search.pageSize ?? defaultRows}
						path={"/chatlogs"}
						data={{ data: [], count: -1 }}
					/>
				</ContainerWithHeader>
			</Grid>
		</Grid>
	);
}
