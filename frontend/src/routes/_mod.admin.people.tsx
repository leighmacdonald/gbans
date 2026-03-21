import NiceModal from "@ebay/nice-modal-react";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { fromUnixTime } from "date-fns";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import z from "zod/v4";
import { apiSearchPeople } from "../api";
import { PersonEditModal } from "../component/modal/PersonEditModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	filterValue,
	makeSchemaState,
	type OnChangeFn,
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

const validateSearch = z
	.object({
		staff_only: z.boolean().catch(false),
	})
	.extend(makeSchemaState("steam_id").shape);

const columnHelper = createMRTColumnHelper<Person>();
const defaultOptions = createDefaultTableOptions<Person>();

export const Route = createFileRoute("/_mod/admin/people")({
	component: AdminPeople,
	validateSearch,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "People" }, match.context.title("People")],
	}),
});

function AdminPeople() {
	const search = Route.useSearch();
	const { sendFlash } = useUserFlashCtx();
	const { hasPermission } = useAuth();
	const navigate = useNavigate();

	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["people", { search }],
		queryFn: async () => {
			const steam_id = filterValue("steam_id", search.columnFilters);
			const staff_only = Boolean(
				search.columnFilters?.find((filter) => filter.id === "staff_only")?.value ?? false,
			);
			const sort = search.sorting ? sortValueDefault(search.sorting, "steam_id") : { id: "steam_id", desc: true };

			return await apiSearchPeople({
				// personaname: filterValue("body", search.columnFilters),
				desc: sort ? sort.desc : true,
				limit: search.pagination?.pageSize,
				offset: search.pagination ? search.pagination.pageIndex * search.pagination.pageSize : undefined,
				with_permissions: staff_only ? PermissionLevel.Reserved : PermissionLevel.Banned,
				order_by: sort ? sort.id : "steam_id",
				steam_ids: steam_id && steam_id !== "" ? [steam_id] : [],
			});
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
						/>
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
				header: "Vac",
				grow: false,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() > 0} />,
			}),
			columnHelper.accessor("community_banned", {
				header: "CB",
				grow: false,
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
	}, []);

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
		renderRowActionMenuItems: ({ row }) => [
			hasPermission(PermissionLevel.Admin) ? (
				<IconButton color={"warning"} onClick={() => onEditPerson(row.original)} key={"editperms"}>
					<VpnKeyIcon />
				</IconButton>
			) : null,
		],
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Player Search"} />
			</Grid>
		</Grid>
	);
}
