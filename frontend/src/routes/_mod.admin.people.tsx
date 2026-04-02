import NiceModal from "@ebay/nice-modal-react";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import { useTheme } from "@mui/material";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import { fromUnixTime } from "date-fns";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiSearchPeople } from "../api";
import { PersonEditModal } from "../component/modal/PersonEditModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import RouterLink from "../component/RouterLink.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	filterValueBool,
	filterValueNumber,
	filterValueNumberArray,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
	setColumnFilter,
	sortValueDefault,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { TableCellRelativeDateField } from "../component/table/TableCellRelativeDateField.tsx";
import { useAuth } from "../hooks/useAuth.ts";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import {
	communityVisibilityState,
	PermissionLevel,
	type PermissionLevelEnum,
	type Person,
	permissionLevelString,
} from "../schema/people.ts";

const defaultValues = makeSchemaDefaults({ defaultColumn: "created_on" });
const validateSearch = makeSchemaState("created_on");
const columnHelper = createMRTColumnHelper<Person>();
const defaultOptions = createDefaultTableOptions<Person>();

export const Route = createFileRoute("/_mod/admin/people")({
	component: AdminPeople,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "People" }, match.context.title("People")],
	}),
});

function AdminPeople() {
	const search = Route.useSearch();
	const navigate = useNavigate();
	const theme = useTheme();
	const { sendFlash } = useUserFlashCtx();
	const { hasPermission } = useAuth();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["people", { search }],
		queryFn: async ({ signal }) => {
			const steam_id = filterValue("steam_id", search.columnFilters);
			const sort = search.sorting ? sortValueDefault(search.sorting, "created_on") : undefined;

			return await apiSearchPeople(
				{
					// personaname: filterValue("body", search.columnFilters),
					desc: sort ? sort.desc : true,
					limit: search.pagination?.pageSize,
					offset: search.pagination ? search.pagination.pageIndex * search.pagination?.pageSize : 0,
					order_by: sort ? sort.id : "created_on",
					with_permissions: filterValueNumberArray<Person, PermissionLevelEnum>(
						"permission_level",
						search.columnFilters,
					),
					steam_ids: steam_id && steam_id !== "" ? [steam_id] : [],
					vac_bans: filterValueNumber("vac_bans", search.columnFilters),
					game_bans: filterValueNumber("game_bans", search.columnFilters),
					community_banned: filterValueBool("community_banned", search.columnFilters),
					// time_created_before: filterValueDate("time_created_before", search.columnFilters),
					// time_created_after: filterValueDate("time_created_after", search.columnFilters),
				},
				signal,
			);
		},
	});

	const onEditPerson = useCallback(
		async (person: Person) => {
			try {
				await NiceModal.show(PersonEditModal, {
					person,
				});
				sendFlash("success", "Updated permission level successfully");
			} catch (e) {
				sendFlash("error", `${e}`);
			}
		},
		[sendFlash],
	);
	const setSorting: OnChangeFn<MRT_SortingState> = useCallback(
		(updater) => {
			navigate({
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
		(updater) => {
			navigate({
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
		(updater) => {
			navigate({
				to: Route.fullPath,
				search: {
					...search,
					pagination: search.pagination
						? typeof updater === "function"
							? updater(search.pagination)
							: updater
						: undefined,
				},
			});
		},
		[search, navigate],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("steam_id", {
				header: "SteamID",
				grow: true,
				Cell: ({ row }) => {
					return (
						<PersonCell
							steam_id={row.original.steam_id}
							personaname={row.original.persona_name}
							avatar_hash={row.original.avatarhash}
						>
							<RouterLink
								style={{
									color:
										theme.palette.mode === "dark"
											? theme.palette.primary.light
											: theme.palette.primary.dark,
								}}
								to={Route.fullPath}
								search={setColumnFilter(search, "steam_id", row.original.steam_id)}
							>
								{row.original.persona_name ?? row.original.steam_id}
							</RouterLink>
						</PersonCell>
					);
				},
			}),
			columnHelper.accessor("community_visibility_state", {
				header: "Visibility",
				grow: false,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>
						{cell.getValue() === communityVisibilityState.Public ? "Public" : "Private"}
					</Typography>
				),
			}),
			columnHelper.accessor("vac_bans", {
				header: "Vac Bans",
				grow: false,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>{cell.getValue() > 0 ? cell.getValue() : ""}</Typography>
				),
			}),
			columnHelper.accessor("community_banned", {
				header: "Comm Ban",
				grow: false,
				filterVariant: "checkbox",
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),

			columnHelper.accessor("time_created", {
				header: "Created",
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={fromUnixTime(cell.getValue())} />,
			}),

			columnHelper.accessor("created_on", {
				header: "First Seen",
				grow: false,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),

			columnHelper.accessor("permission_level", {
				header: "Perms",
				grow: false,
				filterVariant: "multi-select",
				filterSelectOptions: Object.values(PermissionLevel).map((perm) => ({
					label: permissionLevelString(perm),
					value: perm,
				})),
				Cell: ({ row }) => (
					<Typography>
						{permissionLevelString(
							row.original
								? row.original.permission_level
								: (PermissionLevel.Guest as PermissionLevelEnum),
						)}
					</Typography>
				),
			}),
		];
	}, [theme, search]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.data : [],
		rowCount: data ? data.count : 0,
		enableFilters: true,
		enableRowActions: true,
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
				steam_id: true,
				source_id: true,
				body: true,
				created_on: true,
			},
		},
		manualFiltering: true,
		manualPagination: true,
		manualSorting: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					disabled={!hasPermission(PermissionLevel.Admin)}
					color={"warning"}
					onClick={() => onEditPerson(row.original)}
					key={"editperms"}
				>
					<VpnKeyIcon />
				</IconButton>
			</RowActionContainer>
		),
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Player Search"} />
			</Grid>
		</Grid>
	);
}
