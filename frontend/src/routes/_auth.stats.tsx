import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useQuery } from "@connectrpc/connect-query";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import Grid from "@mui/system/Grid";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { format, subDays } from "date-fns";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo, useState } from "react";
import z from "zod/v4";
import { ErrorDetails } from "../component/ErrorDetails.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import {
	createDefaultTableOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { useAppForm } from "../contexts/formContext.tsx";
import { renderTableError } from "../error.tsx";
import { FilterSchema } from "../rpc/database/query/v1/filter_pb.ts";
import { QueryStatsRequestSchema, TimeBucket, Variant, type VariantStats } from "../rpc/stats/v1/stats_pb.ts";
import { buckets, queryStats, weaponList } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { classList } from "../tf2.tsx";
import { ensureFeatureEnabled } from "../util/features.ts";
import { enumValues } from "../util/lists.ts";
import { toTitleCase } from "../util/strings.ts";

const defaultValues = { ...makeSchemaDefaults({ defaultColumn: "rank" }) };
const validateSearch = z
	.object({
		timeBucket: z.enum(TimeBucket).optional().default(TimeBucket.DAILY),
		statsBucketId: z.number().optional().default(1),
		time: z.coerce.date().optional().default(subDays(new Date(), 1)),
		variant: z.enum(Variant).optional().default(Variant.OVERALL_UNSPECIFIED),
		variantKey: z.string().optional(),
	})
	.extend(makeSchemaState("rank").shape);

export const Route = createFileRoute("/_auth/stats")({
	component: StatsComponent,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogsEnabled);
	},
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Game Stats" }, match.context.title("Game Stats")],
	}),
});

const columnHelper = createMRTColumnHelper<VariantStats>();
const defaultOptions = createDefaultTableOptions<VariantStats>();

function StatsComponent() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const [selectedVariant, setSelectedVariant] = useState(Variant.OVERALL_UNSPECIFIED);

	const {
		data: variantKeys,
		isLoading: isLoadingVariantKeys,
		isError: isErrorVariantKeys,
	} = useQuery(weaponList, {});

	const { data: statBuckets, isLoading: isLoadingStatBuckets, isError: isErrorStatBuckets } = useQuery(buckets, {});

	const qOpts = useMemo(() => {
		const sort = search.sorting?.find((sort) => sort);
		return create(QueryStatsRequestSchema, {
			statsBucketId: search.statsBucketId ?? 1,
			timeBucket: search.timeBucket ?? TimeBucket.DAILY,
			time: timestampFromDate(search.time ?? subDays(new Date(), 1)),
			variant: search.variant ?? Variant.OVERALL_UNSPECIFIED,
			variantKey: search.variantKey ?? "",
			filter: create(FilterSchema, {
				limit: String(search.pagination?.pageSize ?? 25),
				desc: sort ? sort.desc : false,
				offset: String(search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : 0),
				orderBy: sort ? sort.id : "rank",
			}),
		});
	}, [search]);

	const { data, isLoading, isError, error, isRefetching } = useQuery(queryStats, qOpts, {
		enabled: !isLoadingStatBuckets && !isErrorStatBuckets && !isLoadingVariantKeys && !isErrorVariantKeys,
		retry: false,
	});

	const sortedVariantKeys = useMemo(() => {
		if (!variantKeys?.weapons) {
			return classList;
		}

		switch (selectedVariant) {
			case Variant.WEAPONS:
				return variantKeys.weapons.toSorted();
			case Variant.CLASSES:
				return classList;
			default:
				return [];
		}
	}, [variantKeys, selectedVariant]);

	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					sorting: typeof updater === "function" ? updater(search.sorting ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setColumnFilters: OnChangeFn<MRT_ColumnFiltersState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					columnFilters: typeof updater === "function" ? updater(search.columnFilters ?? []) : updater,
				},
			});
		},
		[search, navigate],
	);

	const setPagination: OnChangeFn<MRT_PaginationState> = useCallback(
		async (updater) => {
			await navigate({
				to: Route.fullPath,
				search: {
					...search,
					pagination:
						typeof updater === "function"
							? updater(search.pagination ?? { pageIndex: 0, pageSize: 50 })
							: updater,
				},
			});
		},
		[search, navigate],
	);

	const title = useMemo(() => {
		const bucket = statBuckets?.buckets.find((b) => b.statsBucketId === search.statsBucketId)?.bucketName ?? "";
		var period = "";
		switch (search.timeBucket) {
			case TimeBucket.DAILY:
				period = "Daily";
				break;
		}

		return `${period}	${bucket} Stats (${format(search.time ?? new Date(), "PPP")}) [${search.variantKey}]`;
	}, [search, statBuckets]);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("rank", {
				header: "#",
				size: 50,
				grow: false,
				enableSorting: true,
				enableColumnFilter: false,
				Cell: ({ cell }) => <Typography> #{cell.getValue()}</Typography>,
			}),

			columnHelper.accessor("player", {
				header: "Player",
				enableColumnFilter: false,
				enableSorting: false,
				grow: false,

				Cell: ({ row }) => (
					<PersonCell
						avatarHash={row.original.player?.avatarHash ?? ""}
						personaName={row.original.player?.name ?? row.original.player?.steamId ?? ""}
						steamId={row.original.player?.steamId ?? ""}
					/>
				),
			}),

			columnHelper.accessor("kills", {
				header: "Kills",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),

			columnHelper.accessor("assists", {
				header: "Assts",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),

			columnHelper.accessor("deaths", {
				header: "Deaths",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("damage", {
				header: "Damage",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("damageTaken", {
				header: "DT",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("dominations", {
				header: "Doms",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("dominated", {
				header: "Domd",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("airshots", {
				header: "Airshots",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("headshots", {
				header: "HS (K)",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
				Cell: ({ row }) => (
					<Typography>
						{row.original.headshots} ({row.original.headshotKills})
					</Typography>
				),
			}),
			columnHelper.accessor("backstabs", {
				header: "BS (K)",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
				Cell: ({ row }) => (
					<Typography>
						{row.original.backstabs} ({row.original.backstabKills})
					</Typography>
				),
			}),

			columnHelper.accessor("wasHeadshot", {
				header: "Was HS",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("wasBackstabbed", {
				header: "Was BS",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("healing", {
				header: "Healing",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("drops", {
				header: "Drops",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),

			columnHelper.accessor("nearFullChargeDeath", {
				header: "NFCD",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("chargesUber", {
				header: "Ubers",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("chargesKritz", {
				header: "Kritz",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("chargesVacc", {
				header: "Vacc",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
			columnHelper.accessor("chargesQuickfix", {
				header: "Qf",
				size: 100,
				grow: false,
				enableSorting: false,
				enableColumnFilter: false,
			}),
		];
	}, []);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.statContainer.value?.stats ?? [],
		rowCount: Number(data ? data.count : 0),
		enableFilters: false,
		enableColumnActions: false,
		state: {
			columnFilters: search.columnFilters,
			isLoading: isLoading || isRefetching,
			pagination: search.pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting: search.sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				rank: true,
				steamID: true,
				kills: true,
				assists: true,
				deaths: true,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		muiToolbarAlertBannerProps: renderTableError(error),
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({ to: "/stats", search: { ...search, ...value } });
		},
		defaultValues: {
			statsBucketID: search.statsBucketId,
			variant: search.variant,
			timeBucket: search.timeBucket,
			time: search.time,
			variantKey: search.variantKey,
		},
	});

	// const activeVariants = useMemo(() => 	{}, [search.]);

	if (isError) {
		return <ErrorDetails error={error} />;
	}
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Paper>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Stack direction={"row"} padding={1} spacing={2}>
							<form.AppField
								name={"statsBucketID"}
								children={(field) => {
									return (
										<field.BucketField
											items={statBuckets?.buckets ?? []}
											label="Stats Group"
											variant={"standard"}
											renderItem={(item) => {
												return (
													<MenuItem key={item.bucketName} value={item.statsBucketId}>
														{toTitleCase(item.bucketName)}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>
							<form.AppField
								name={"timeBucket"}
								children={(field) => {
									return (
										<field.StatsTimeBucketField
											items={enumValues(TimeBucket)}
											label="Time Range"
											variant={"standard"}
											renderItem={(item) => {
												return (
													<MenuItem key={item} value={item}>
														{toTitleCase(TimeBucket[item])}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>
							<form.AppField
								name={"variant"}
								children={(field) => {
									return (
										<field.StatsVariantField
											items={enumValues(Variant)}
											label="Stat Type"
											variant={"standard"}
											handleChange={(e) => setSelectedVariant(e)}
											renderItem={(item) => {
												const v = item as Variant;
												return (
													<MenuItem key={v} value={v}>
														{toTitleCase(Variant[v])}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>

							<form.AppField
								name={"variantKey"}
								children={(field) => {
									return (
										<field.SelectStringField
											items={sortedVariantKeys}
											label="Filter By"
											variant={"standard"}
											disabled={
												selectedVariant !== Variant.CLASSES &&
												selectedVariant !== Variant.WEAPONS
											}
											renderItem={(item) => {
												return (
													<MenuItem key={item} value={item}>
														{toTitleCase(item)}
													</MenuItem>
												);
											}}
										/>
									);
								}}
							/>

							<form.AppForm>
								<form.SubmitButton fullWidth label="Show Me" />
							</form.AppForm>
						</Stack>
					</form>
				</Paper>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={title} />
			</Grid>
		</Grid>
	);
}
