import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import Button from "@mui/material/Button";
import Grid from "@mui/material/Grid";
import IconButton from "@mui/material/IconButton";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createMRTColumnHelper, useMaterialReactTable } from "material-react-table";
import { useCallback, useMemo } from "react";
import { apiGetNewsAll, apiNewsDelete } from "../api/news.ts";
import { ConfirmationModal } from "../component/modal/ConfirmationModal.tsx";
import { NewsEditModal } from "../component/modal/NewsEditModal.tsx";
import { BoolCell } from "../component/table/BoolCell.tsx";
import { createDefaultTableOptions } from "../component/table/options.ts";
import { SortableTable } from "../component/table/SortableTable.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { NewsEntry } from "../schema/news.ts";
import { renderDateTime } from "../util/time.ts";

export const Route = createFileRoute("/_mod/admin/news")({
	component: AdminNews,
	head: ({ match }) => ({
		meta: [{ name: "description", content: "News Management" }, match.context.title("News")],
	}),
});

const columnHelper = createMRTColumnHelper<NewsEntry>();
const defaultOptions = createDefaultTableOptions<NewsEntry>();

function AdminNews() {
	const queryClient = useQueryClient();
	const { data, isLoading, isError } = useQuery({
		queryKey: ["newsList"],
		queryFn: async () => {
			return (await apiGetNewsAll()) ?? [];
		},
	});

	const { sendFlash, sendError } = useUserFlashCtx();

	const onCreate = useCallback(async () => {
		try {
			const newEntry = await NiceModal.show(NewsEditModal);
			queryClient.setQueryData(["newsList"], [...(data ?? []), newEntry]);
			sendFlash("success", `Entry created successfully`);
		} catch (e) {
			sendFlash("error", `Error trying to create entry: ${e}`);
		}
	}, [data, queryClient, sendFlash]);

	const deleteMutation = useMutation({
		mutationKey: ["deleteNews"],
		mutationFn: async (variables: { news_id: number }) => {
			await apiNewsDelete(variables.news_id);
			return variables.news_id;
		},
		onSuccess: (news_id) => {
			queryClient.setQueryData(
				["newsList"],
				(data ?? []).filter((e) => e.news_id !== news_id),
			);
			sendFlash("success", `Entry deleted successfully`);
		},
		onError: sendError,
	});

	const onDelete = useCallback(
		async (entry: NewsEntry) => {
			try {
				const confirmed = await NiceModal.show(ConfirmationModal, {
					title: "Delete news entry?",
					children: "This cannot be undone",
				});
				if (!confirmed) {
					return;
				}
				deleteMutation.mutate({ news_id: entry.news_id });
			} catch (e) {
				sendFlash("error", `Failed to create confirmation modal: ${e}`);
			}
		},
		[deleteMutation, sendFlash],
	);

	const onEdit = useCallback(
		async (entry: NewsEntry) => {
			try {
				const editedEntry = (await NiceModal.show(NewsEditModal, {
					entry: entry,
				})) as NewsEntry;
				queryClient.setQueryData(
					["newsList"],
					data?.map((e) => (e.news_id === editedEntry.news_id ? editedEntry : e)),
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
			columnHelper.accessor("news_id", {
				header: "ID",
				grow: false,
			}),
			columnHelper.accessor("title", {
				header: "Title",
				grow: true,
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				grow: false,
				enableColumnFilter: false,
				Cell: ({ cell }) => renderDateTime(cell.getValue()),
			}),
			columnHelper.accessor("is_published", {
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

	const table = useMaterialReactTable({
		...defaultOptions,
		columns,
		data: data ?? [],
		enableFilters: true,
		enableRowActions: true,
		state: {
			isLoading,
			showAlertBanner: isError,
		},
		initialState: {
			...defaultOptions.initialState,
			sorting: [{ id: "news_id", desc: false }],
			columnVisibility: {
				title: true,
				news_id: false,
				created_on: false,
				updated_on: true,
				is_published: true,
			},
		},
		renderRowActionMenuItems: ({ row }) => [
			<Button
				key={"editButton"}
				variant={"contained"}
				color={"warning"}
				startIcon={<EditIcon />}
				onClick={async () => {
					await onEdit(row.original);
				}}
			>
				Edit
			</Button>,
			<Button
				key={"deleteButton"}
				variant={"contained"}
				color={"error"}
				startIcon={<DeleteIcon />}
				onClick={async () => {
					await onDelete(row.original);
				}}
			>
				Delete
			</Button>,
		],
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
