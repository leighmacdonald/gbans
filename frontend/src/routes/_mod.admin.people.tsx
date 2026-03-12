import NiceModal from "@ebay/nice-modal-react";
import VpnKeyIcon from "@mui/icons-material/VpnKey";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import Typography from "@mui/material/Typography";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { fromUnixTime } from "date-fns";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo, useState } from "react";
import { apiSearchPeople } from "../api";
import { PersonEditModal } from "../component/modal/PersonEditModal.tsx";
import { PersonCell } from "../component/PersonCell.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
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

export const Route = createFileRoute("/_mod/admin/people")({
	component: AdminPeople,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "People" }, match.context.title("People")],
	}),
});

const columnHelper = createMRTColumnHelper<Person>();
const defaultOptions = createDefaultTableOptions<Person>();

function AdminPeople() {
	const [columnFilters, setColumnFilters] = useState<MRT_ColumnFiltersState>([]);
	const [globalFilter, setGlobalFilter] = useState("");
	const [sorting, setSorting] = useState<MRT_SortingState>([]);
	const [pagination, setPagination] = useState<MRT_PaginationState>({
		pageIndex: 0,
		pageSize: 50,
	});
	const { sendFlash } = useUserFlashCtx();
	const { hasPermission } = useAuth();
	const { data, isLoading, isError, isRefetching } = useQuery({
		queryKey: ["people", { columnFilters, globalFilter, pagination, sorting }],
		queryFn: async () => {
			const steam_id = columnFilters.find((filter) => filter.id === "steam_id")?.value;
			const sort = sorting.find((sort) => sort);
			return await apiSearchPeople({
				personaname: "",
				desc: sort ? sort.desc : false,
				limit: pagination.pageSize,
				offset: pagination.pageIndex * pagination.pageSize,
				staff_only: false,
				order_by: sort ? sort.id : "created_on",
				steam_ids: steam_id && steam_id !== "" ? [String(steam_id)] : [],
				ip: "",
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
	const columns = useMemo(() => {
		return [
			columnHelper.accessor("steam_id", {
				header: "Profile",
				grow: true,
				Cell: ({ row }) => {
					return (
						<PersonCell
							showCopy={true}
							steam_id={row.original.steam_id}
							personaname={row.original.persona_name}
							avatar_hash={row.original.avatarhash}
						/>
					);
				},
			}),
			columnHelper.accessor("community_visibility_state", {
				header: "Visibility",
				size: 50,
				Cell: ({ cell }) => (
					<Typography variant={"body1"}>
						{cell.getValue() === communityVisibilityState.Public ? "Public" : "Private"}
					</Typography>
				),
			}),
			columnHelper.accessor("vac_bans", {
				header: "Vac",
				size: 20,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue() > 0} />,
			}),
			columnHelper.accessor("community_banned", {
				header: "CB",
				size: 20,
				Cell: ({ cell }) => <BoolCell enabled={cell.getValue()} />,
			}),

			columnHelper.accessor("time_created", {
				header: "Created",
				size: 50,
				Cell: ({ cell }) => <TableCellRelativeDateField date={fromUnixTime(cell.getValue())} />,
			}),

			columnHelper.accessor("created_on", {
				header: "Seen",
				size: 80,
				Cell: ({ cell }) => <TableCellRelativeDateField date={cell.getValue()} />,
			}),

			columnHelper.accessor("permission_level", {
				header: "Perms",
				size: 80,
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
			columnHelper.display({
				header: "Act",
				grow: false,
				Cell: (info) => {
					return hasPermission(PermissionLevel.Admin) ? (
						<IconButton color={"warning"} onClick={() => onEditPerson(info.row.original)}>
							<VpnKeyIcon />
						</IconButton>
					) : null;
				},
			}),
		];
	}, [onEditPerson, hasPermission]);

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ? data.data : [],
		rowCount: data ? data.count : 0,
		enableFilters: true,
		state: {
			columnFilters,
			globalFilter,
			isLoading,
			pagination,
			showAlertBanner: isError,
			showProgressBars: isRefetching,
			sorting,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "updated_on", desc: true }],
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
		muiToolbarAlertBannerProps: isError
			? {
					color: "error",
					children: "Error loading data",
				}
			: undefined,
		onColumnFiltersChange: setColumnFilters,
		onGlobalFilterChange: setGlobalFilter,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable table={table} title={"Player Search"} />
			</Grid>
		</Grid>
	);
}
