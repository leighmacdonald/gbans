import { useMutation, useQuery } from "@connectrpc/connect-query";
import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import { useQueryClient } from "@tanstack/react-query";
import { createFileRoute, stripSearchParams, useNavigate } from "@tanstack/react-router";
import {
	createMRTColumnHelper,
	type MRT_ColumnFiltersState,
	type MRT_PaginationState,
	type MRT_SortingState,
	useMaterialReactTable,
} from "material-react-table";
import { useCallback, useMemo } from "react";
import { ConfirmationModal } from "../component/modal/ConfirmationModal.tsx";
import { NewsEditModal } from "../component/modal/NewsEditModal.tsx";
import { RowActionContainer } from "../component/RowActionContainer.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import {
	createDefaultTableOptions,
	makeRowActionsDefOptions,
	makeSchemaDefaults,
	makeSchemaState,
	type OnChangeFn,
} from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { Article } from "../rpc/news/v1/news_pb.ts";
import { all, delete$ } from "../rpc/news/v1/news-NewsService_connectquery.ts";
import { renderTimestamp } from "../util/time.ts";

const columnHelper = createMRTColumnHelper<Article>();
const defaultOptions = createDefaultTableOptions<Article>();
const defaultValues = makeSchemaDefaults({ defaultColumn: "news_id" });
const validateSearch = makeSchemaState("news_id");

export const Route = createFileRoute("/_mod/admin/news")({
	component: AdminNews,
	validateSearch,
	search: {
		middlewares: [stripSearchParams(defaultValues)],
	},
	head: ({ match }) => ({
		meta: [{ name: "description", content: "News Management" }, match.context.title("News")],
	}),
});

function AdminNews() {
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const search = Route.useSearch();

	const { data, isLoading, isError } = useQuery(all);

	const { sendFlash, sendError } = useUserFlashCtx();

	const onCreate = useCallback(async () => {
		try {
			const newEntry = await NiceModal.show(NewsEditModal);
			queryClient.setQueryData(["newsList"], [...(data?.articles ?? []), newEntry]);
			sendFlash("success", `Entry created successfully`);
		} catch (e) {
			sendFlash("error", `Error trying to create entry: ${e}`);
		}
	}, [data, queryClient, sendFlash]);

	const deleteMutation = useMutation(delete$, {
		onSuccess: (_, req) => {
			queryClient.setQueryData(
				["newsList"],
				(data?.articles ?? []).filter((e) => e.newsId !== req.newsId),
			);
			sendFlash("success", `Entry deleted successfully`);
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (entry: Article) => {
			try {
				const confirmed = await NiceModal.show(ConfirmationModal, {
					title: "Delete news entry?",
					children: "This cannot be undone",
				});
				if (!confirmed) {
					return;
				}
				deleteMutation.mutate({ newsId: entry.newsId });
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[deleteMutation, sendFlash],
	);

	const onEdit = useCallback(
		async (entry: Article) => {
			try {
				const editedEntry = (await NiceModal.show(NewsEditModal, {
					entry: entry,
				})) as Article;
				queryClient.setQueryData(
					["newsList"],
					(data?.articles ?? []).map((e) => (e.newsId === editedEntry.newsId ? editedEntry : e)),
				);
				sendFlash("success", `Entry updated successfully`);
			} catch (e) {
				sendFlash("error", `Error trying to update entry: ${e}`);
			}
		},
		[queryClient, sendFlash, data],
	);

	const columns = useMemo(() => {
		return [
			columnHelper.accessor("newsId", {
				header: "ID",
				grow: false,
			}),
			columnHelper.accessor("title", {
				header: "Title",
				grow: true,
			}),
			columnHelper.accessor("createdOn", {
				header: "Created",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
			columnHelper.accessor("updatedOn", {
				header: "Updated",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => renderTimestamp(cell.getValue()),
			}),
			columnHelper.accessor("isPublished", {
				meta: { tooltip: "Published" },
				filterVariant: "checkbox",
				header: "Published",
				grow: false,
				Cell: ({ cell }) => {
					return <BoolCell enabled={cell.getValue()} />;
				},
			}),
		];
	}, []);

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

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data?.articles ?? [],
		enableFilters: true,
		enableRowActions: true,
		enableFacetedValues: true,
		onColumnFiltersChange: setColumnFilters,
		onPaginationChange: setPagination,
		onSortingChange: setSorting,
		displayColumnDefOptions: makeRowActionsDefOptions(2),
		state: {
			isLoading,
			showAlertBanner: isError,
			columnFilters: search.columnFilters,
			sorting: search.sorting,
			pagination: search.pagination,
		},
		renderRowActions: ({ row }) => (
			<RowActionContainer>
				<IconButton
					key={"delete"}
					color={"error"}
					onClick={async () => {
						await onDelete(row.original);
					}}
				>
					<DeleteIcon />
				</IconButton>
				<IconButton
					key={"edit"}
					color={"warning"}
					onClick={async () => {
						await onEdit(row.original);
					}}
				>
					<EditIcon />
				</IconButton>
			</RowActionContainer>
		),
		initialState: {
			...defaultOptions.initialState,
			columnVisibility: {
				title: true,
				news_id: false,
				created_on: false,
				updated_on: true,
				is_published: true,
			},
		},
	});
	return (
		<Grid container spacing={2}>
			<Grid size={{ xs: 12 }}>
				<SortableTable
					table={table}
					title={"News Entries"}
					buttons={[
						<IconButton key={"addButton"} onClick={onCreate}>
							<AddIcon />
						</IconButton>,
					]}
				/>
			</Grid>
		</Grid>
	);
}
