import { create } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import { useMutation, useQuery } from "@connectrpc/connect-query";
import BarChartIcon from "@mui/icons-material/BarChart";
import CalendarMonthIcon from "@mui/icons-material/CalendarMonth";
import EmojiEventsIcon from "@mui/icons-material/EmojiEvents";
import GroupIcon from "@mui/icons-material/Group";
import Autocomplete from "@mui/material/Autocomplete";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import MenuItem from "@mui/material/MenuItem";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import { alpha } from "@mui/material/styles";
import MuiTextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import Grid from "@mui/system/Grid";
import type { DateView } from "@mui/x-date-pickers/models";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, redirect, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { format, subDays, subYears } from "date-fns";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_ColumnPinningState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useEffect, useMemo, useState } from "react";
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
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import { FilterSchema } from "../rpc/database/query/v1/filter_pb.ts";
import { Permission } from "../rpc/roles/v1/roles_pb.ts";
import {
	type OverallStats,
	QueryStatsRequestSchema,
	TimeBucket,
	Variant,
	type VariantStats,
} from "../rpc/stats/v1/stats_pb.ts";
import { buckets, queryStats, refreshStats, weaponList } from "../rpc/stats/v1/stats-StatsService_connectquery.ts";
import { classList } from "../tf2.tsx";
import { logErr } from "../util/errors.ts";
import { ensureFeatureEnabled } from "../util/features.ts";
import { enumValues } from "../util/lists.ts";
import { toTitleCase } from "../util/strings.ts";

const defaultValues = { ...makeSchemaDefaults({ defaultColumn: "rank" }) };
const validateSearch = z
	.object({
		timeBucket: z.enum(TimeBucket).optional().default(TimeBucket.ALLTIME),
		statsBucketId: z.number().optional(),
		time: z.coerce.date().optional().default(subDays(new Date(), 1)),
		variant: z.enum(Variant).optional().default(Variant.OVERALL_UNSPECIFIED),
		variantKey: z.string().optional(),
	})
	.extend(makeSchemaState("rank").shape);

export const Route = createFileRoute("/_auth/stats")({
	component: StatsComponent,
	beforeLoad: ({ context }) => {
		ensureFeatureEnabled(context.appInfo.chatlogsEnabled);
		if (!context.auth?.hasPermission(Permission.STATS_READ)) {
			throw redirect({ to: "/permission" });
		}
	},
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "Game Stats" }, match.context.title("Game Stats")],
	}),
});

// Rows arrive as VariantStats (weapons/classes) or OverallStats (global
// leaderboards) depending on the selected variant. Both share the displayed
// stat fields; overall-only fields are read through OverallStats below.
type StatRow = VariantStats | OverallStats;

const columnHelper = createMRTColumnHelper<StatRow>();
const defaultOptions = createDefaultTableOptions<StatRow>();

function StatsComponent() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const [selectedVariant, setSelectedVariant] = useState(search.variant ?? Variant.OVERALL_UNSPECIFIED);
	const { profile, hasPermission } = useAuth();
	const { sendFlash } = useUserFlashCtx();
	const queryClient = useQueryClient();

	// The local selection drives the Filter By dropdown before submit; keep it
	// in sync when the URL search params change (submit, back/forward).
	useEffect(() => {
		setSelectedVariant(search.variant ?? Variant.OVERALL_UNSPECIFIED);
	}, [search.variant]);
	const [selectedTimeBucket, setSelectedTimeBucket] = useState(search.timeBucket ?? TimeBucket.DAILY);

	// The date picker adapts to the selected range for the same reason.
	useEffect(() => {
		setSelectedTimeBucket(search.timeBucket ?? TimeBucket.DAILY);
	}, [search.timeBucket]);

	// Date selection granularity follows the time range: picking a day makes
	// no sense for a yearly leaderboard. MUI has no week picker, so weekly
	// uses a day picker (any day resolves to its week).
	const dateConfig = useMemo((): {
		label: string;
		views?: DateView[];
		openTo?: DateView;
		disabled?: boolean;
		helperText?: string;
	} => {
		switch (selectedTimeBucket) {
			case TimeBucket.YEARLY:
				return { label: "Year", views: ["year"], openTo: "year" };
			case TimeBucket.MONTHLY:
				return { label: "Month", views: ["year", "month"], openTo: "month" };
			case TimeBucket.WEEKLY:
				return { label: "Week Of", helperText: "Any day within the week" };
			case TimeBucket.ALLTIME:
				return { label: "Date", disabled: true, helperText: "Not used for all-time" };
			default:
				return { label: "Day" };
		}
	}, [selectedTimeBucket]);
	const refreshMutation = useMutation(refreshStats, {
		onSuccess: (resp) => {
			sendFlash("success", `Refreshed ${resp.refreshedViews.length} stats views`);
			queryClient.invalidateQueries();
		},
		onError: (e) => {
			logErr(e);
			sendFlash("error", "Stats refresh failed");
		},
	});

	const {
		data: variantKeys,
		isLoading: isLoadingVariantKeys,
		isError: isErrorVariantKeys,
	} = useQuery(weaponList, {});

	const { data: statBuckets, isLoading: isLoadingStatBuckets, isError: isErrorStatBuckets } = useQuery(buckets, {});

	// No explicit bucket in the URL means the default bucket (casual).
	const defaultBucketId = useMemo(() => {
		return statBuckets?.buckets.find((bucket) => bucket.isDefault)?.statsBucketId;
	}, [statBuckets]);
	const effectiveBucketId = search.statsBucketId ?? defaultBucketId;

	const qOpts = useMemo(() => {
		const sort = search.sorting?.find((sort) => sort);
		return create(QueryStatsRequestSchema, {
			statsBucketId: effectiveBucketId ?? 1,
			timeBucket: search.timeBucket ?? TimeBucket.ALLTIME,
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
	}, [search, effectiveBucketId]);

	const { data, isLoading, isError, error, isRefetching } = useQuery(queryStats, qOpts, {
		enabled: !isLoadingStatBuckets && !isErrorStatBuckets && !isLoadingVariantKeys && !isErrorVariantKeys,
		retry: false,
	});

	const weaponOptions = useMemo(() => {
		return (variantKeys?.weapons ?? []).toSorted();
	}, [variantKeys]);

	// Class options are static; weapon options come from the server. Never
	// fall back to the class list for weapons: an empty/failed weapon list
	// must render empty, not the wrong items.
	const filterOptions = useMemo(() => {
		switch (selectedVariant) {
			case Variant.WEAPONS:
				return weaponOptions;
			case Variant.CLASSES:
				return classList;
			default:
				return [];
		}
	}, [weaponOptions, selectedVariant]);

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

	// The player column stays pinned on the left; other columns remain
	// user-pinnable.
	const [columnPinning, setColumnPinning] = useState<MRT_ColumnPinningState>({
		left: ["player"],
		right: [],
	});
	const onColumnPinningChange: OnChangeFn<MRT_ColumnPinningState> = useCallback((updater) => {
		setColumnPinning((old) => {
			const next = typeof updater === "function" ? updater(old) : updater;
			const left = next.left ?? [];
			return {
				left: left.includes("player") ? left : ["player", ...left],
				right: next.right ?? [],
			};
		});
	}, []);

	const bucketName = useMemo(() => {
		return statBuckets?.buckets.find((b) => b.statsBucketId === effectiveBucketId)?.bucketName ?? "All Groups";
	}, [effectiveBucketId, statBuckets]);

	// Only enabled buckets are selectable; disabled ones (1ku/mge/dodgeball)
	// stay queryable via direct links.
	const enabledBuckets = useMemo(() => {
		return (statBuckets?.buckets ?? []).filter((b) => b.isEnabled);
	}, [statBuckets]);

	const periodLabel = useMemo(() => {
		switch (search.timeBucket) {
			case TimeBucket.WEEKLY:
				return "Weekly";
			case TimeBucket.MONTHLY:
				return "Monthly";
			case TimeBucket.YEARLY:
				return "Yearly";
			case TimeBucket.ALLTIME:
				return "All-Time";
			default:
				return "Daily";
		}
	}, [search.timeBucket]);

	const variantLabel = useMemo(() => {
		var base = "Overall";
		switch (search.variant) {
			case Variant.KILLS:
				base = "Kills";
				break;
			case Variant.HEALING:
				base = "Healing";
				break;
			case Variant.WEAPONS:
				base = "Weapons";
				break;
			case Variant.CLASSES:
				base = "Classes";
				break;
			default:
				base = "Overall";
				break;
		}
		if ((search.variant === Variant.WEAPONS || search.variant === Variant.CLASSES) && search.variantKey) {
			return `${base} · ${toTitleCase(search.variantKey)}`;
		}
		return base;
	}, [search.variant, search.variantKey]);

	const title = useMemo(() => {
		return `${periodLabel} ${toTitleCase(bucketName)} Leaders`;
	}, [periodLabel, bucketName]);

	// Weapon/class icon for the heading, only when a specific one is selected.
	// Weapons without a downloaded icon hide the image via onError.
	const variantIconUrl = useMemo(() => {
		if (!search.variantKey) {
			return null;
		}
		if (search.variant === Variant.WEAPONS) {
			return `/weapons/${search.variantKey}.png`;
		}
		if (search.variant === Variant.CLASSES) {
			return `/classes/${search.variantKey}.png`;
		}
		return null;
	}, [search.variant, search.variantKey]);

	const rows: StatRow[] = useMemo(() => {
		return (data?.statContainer.value?.stats ?? []) as StatRow[];
	}, [data]);

	const leader = useMemo(() => {
		return rows.find((row) => row.rank === "1");
	}, [rows]);

	const leaderText = useMemo(() => {
		if (!leader) {
			return "";
		}
		if (search.variant === Variant.HEALING) {
			return `leads with ${leader.healing} healing`;
		}
		const killsText = `leads with ${leader.kills} kills`;
		if (search.variant === Variant.WEAPONS || search.variant === Variant.CLASSES) {
			return killsText;
		}
		return `${killsText} · ${(leader as OverallStats).points} points`;
	}, [leader, search.variant]);

	const playerCount = useMemo(() => {
		return Number(data?.count ?? 0);
	}, [data]);

	// The backend resolves the rank against the full window, so it is shown
	// even when the user's row is outside the loaded page.
	const myRank = useMemo(() => {
		return Number(data?.myRank ?? 0);
	}, [data]);

	const columns = useMemo(() => {
		const isOverall = search.variant !== Variant.WEAPONS && search.variant !== Variant.CLASSES;
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
				enablePinning: true,
				grow: false,

				Cell: ({ row }) => (
					<PersonCell
						avatarHash={row.original.player?.avatarHash ?? ""}
						personaName={row.original.player?.name ?? row.original.player?.steamId ?? ""}
						steamId={row.original.player?.steamId ?? ""}
					/>
				),
			}),

			...(isOverall
				? [
						columnHelper.accessor((row) => (row as OverallStats).points ?? "", {
							id: "points",
							header: "Points",
							size: 100,
							grow: false,
							enableSorting: false,
							enableColumnFilter: false,
						}),
					]
				: []),

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
			columnHelper.accessor("healing", {
				header: "Healing",
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
	}, [search.variant]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: rows,
		rowCount: Number(data ? data.count : 0),
		enableFilters: false,
		enableColumnActions: false,
		state: {
			columnFilters: search.columnFilters,
			columnPinning,
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
		enableColumnPinning: true,
		onColumnFiltersChange: setColumnFilters,
		onColumnPinningChange,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		muiTableBodyRowProps: ({ row }) =>
			row.original.player?.steamId === profile.steamId
				? {
						// The sx function receives the theme object itself, not a
						// { theme } wrapper.
						sx: (theme) => ({
							backgroundColor: alpha(theme.palette.primary.main, 0.18),
							"&:hover": { backgroundColor: alpha(theme.palette.primary.main, 0.3) },
						}),
					}
				: {},
		muiToolbarAlertBannerProps: renderTableError(error),
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			await navigate({ to: "/stats", search: { ...search, ...value } });
		},
		defaultValues: {
			statsBucketID: effectiveBucketId,
			variant: search.variant,
			timeBucket: search.timeBucket,
			time: search.time,
			variantKey: search.variantKey,
		},
	});

	// The default bucket (is_default, "casual") only resolves once the buckets
	// query has loaded, which is after the form's initial values are set. Fill
	// the dropdown with it then, unless the URL already pins an explicit bucket
	// or the user has already chosen one.
	useEffect(() => {
		if (search.statsBucketId == null && defaultBucketId != null && form.state.values.statsBucketID == null) {
			form.setFieldValue("statsBucketID", defaultBucketId);
		}
	}, [defaultBucketId, search.statsBucketId, form]);

	if (isError) {
		return <ErrorDetails error={error} />;
	}
	const showVariantKey = selectedVariant === Variant.WEAPONS || selectedVariant === Variant.CLASSES;
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<Paper sx={{ padding: 2.5, overflow: "hidden" }}>
					<Stack spacing={1.5}>
						<Stack
							direction={{ xs: "column", sm: "row" }}
							spacing={1}
							sx={{ justifyContent: "space-between", alignItems: { sm: "flex-start" } }}
						>
							<Box>
								<Typography variant="overline" color="textSecondary" sx={{ letterSpacing: 1.5 }}>
									Game Stats
								</Typography>
								<Stack direction="row" spacing={1.5} sx={{ alignItems: "center" }}>
									{variantIconUrl && (
										<Box
											sx={{
												width: 64,
												height: 44,
												display: "flex",
												alignItems: "center",
												justifyContent: "center",
												flexShrink: 0,
											}}
										>
											<Box
												component="img"
												src={variantIconUrl}
												alt=""
												onError={(e) => {
													e.currentTarget.parentElement?.style.setProperty("display", "none");
												}}
												sx={{
													maxWidth: "100%",
													maxHeight: "100%",
													objectFit: "contain",
													imageRendering: "pixelated",
												}}
											/>
										</Box>
									)}
									<Typography variant="h4" sx={{ fontWeight: 700, lineHeight: 1.1 }}>
										{title}
									</Typography>
								</Stack>
							</Box>
							<Stack
								direction="row"
								spacing={1.5}
								sx={{ alignItems: "center", flexShrink: 0, justifyContent: "flex-end" }}
							>
								{myRank > 0 && (
									<Box>
										<Typography
											variant="overline"
											color="textSecondary"
											sx={{ letterSpacing: 1.5 }}
										>
											Your Rank
										</Typography>
										<Typography variant="h5" sx={{ fontWeight: 700, lineHeight: 1.1 }}>
											#{myRank}
										</Typography>
									</Box>
								)}
								{hasPermission(Permission.DEMO_ADMIN) && (
									<Button
										variant="outlined"
										size="small"
										disabled={refreshMutation.isPending}
										onClick={() => refreshMutation.mutate({})}
										sx={{ flexShrink: 0 }}
									>
										{refreshMutation.isPending ? "Refreshing…" : "Refresh Stats"}
									</Button>
								)}
							</Stack>
						</Stack>
						<Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: "wrap" }}>
							<Chip
								icon={<GroupIcon />}
								label={toTitleCase(bucketName)}
								size="small"
								variant="outlined"
							/>
							<Chip
								icon={<CalendarMonthIcon />}
								label={
									search.timeBucket === TimeBucket.ALLTIME
										? periodLabel
										: `${periodLabel} · ${format(search.time ?? new Date(), "PPP")}`
								}
								size="small"
								variant="outlined"
							/>
							<Chip icon={<BarChartIcon />} label={variantLabel} size="small" variant="outlined" />
							<Chip
								icon={<EmojiEventsIcon />}
								label={`${playerCount} player${playerCount === 1 ? "" : "s"}`}
								size="small"
								variant="outlined"
							/>
						</Stack>
						{leader?.player && (
							<Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
								<EmojiEventsIcon sx={{ color: "#ffd54f" }} fontSize="small" />
								<PersonCell
									avatarHash={leader.player.avatarHash ?? ""}
									personaName={leader.player.name ?? leader.player.steamId ?? ""}
									steamId={leader.player.steamId ?? ""}
								/>
								<Typography variant="body2" color="textSecondary">
									{leaderText}
								</Typography>
							</Stack>
						)}
					</Stack>
				</Paper>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<Paper sx={{ padding: 2 }}>
					<form
						onSubmit={async (e) => {
							e.preventDefault();
							e.stopPropagation();
							await form.handleSubmit();
						}}
					>
						<Stack direction={"row"} spacing={2} useFlexGap sx={{ flexWrap: "wrap" }}>
							<Box sx={{ minWidth: 160, flex: "1 1 160px" }}>
								<form.AppField
									name={"statsBucketID"}
									children={(field) => {
										return (
											<field.SelectBucketField
												items={enabledBuckets}
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
							</Box>
							<Box sx={{ minWidth: 160, flex: "1 1 160px" }}>
								<form.AppField
									name={"timeBucket"}
									children={(field) => {
										return (
											<field.SelectStatsTimeBucketField
												items={enumValues(TimeBucket)}
												label="Time Range"
												variant={"standard"}
												handleChange={(e) => {
													setSelectedTimeBucket(e);
												}}
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
							</Box>
							<Box sx={{ minWidth: 200, flex: "1 1 200px" }}>
								<form.AppField
									name={"time"}
									children={(field) => {
										return (
											<field.DatePickerField
												label={dateConfig.label}
												views={dateConfig.views}
												openTo={dateConfig.openTo}
												disabled={dateConfig.disabled}
												disableFuture
												minDate={subYears(new Date(), 20)}
												helpText={dateConfig.helperText}
											/>
										);
									}}
								/>
							</Box>
							<Box sx={{ minWidth: 160, flex: "1 1 160px" }}>
								<form.AppField
									name={"variant"}
									children={(field) => {
										return (
											<field.SelectStatsVariantField
												items={enumValues(Variant)}
												label="Stat Type"
												variant={"standard"}
												handleChange={(e) => {
													setSelectedVariant(e);
													form.setFieldValue("variantKey", "");
												}}
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
							</Box>

							{showVariantKey && (
								<Box sx={{ minWidth: 200, flex: "1 1 200px" }}>
									<form.AppField
										name={"variantKey"}
										children={(field) => {
											return (
												<Autocomplete
													options={filterOptions}
													value={field.state.value ?? ""}
													onChange={(_, value) => {
														field.handleChange(value ?? "");
													}}
													getOptionLabel={(option) => toTitleCase(option)}
													renderInput={(params) => (
														<MuiTextField
															{...params}
															label={
																selectedVariant === Variant.CLASSES ? "Class" : "Weapon"
															}
															variant="standard"
															placeholder={`Select a ${selectedVariant === Variant.CLASSES ? "class" : "weapon"}…`}
														/>
													)}
												/>
											);
										}}
									/>
								</Box>
							)}

							<Box sx={{ display: "flex", alignItems: "flex-end", paddingBottom: 0.5 }}>
								<form.AppForm>
									<form.SubmitButton label="Show Me" />
								</form.AppForm>
							</Box>
						</Stack>
					</form>
				</Paper>
			</Grid>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={title} iconLeft={<EmojiEventsIcon />} />
			</Grid>
		</Grid>
	);
}
