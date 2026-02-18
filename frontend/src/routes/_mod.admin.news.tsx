import NiceModal from "@ebay/nice-modal-react";
import AddIcon from "@mui/icons-material/Add";
import DeleteIcon from "@mui/icons-material/Delete";
import EditIcon from "@mui/icons-material/Edit";
import NewspaperIcon from "@mui/icons-material/Newspaper";
import Button from "@mui/material/Button";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute } from "@tanstack/react-router";
import { createColumnHelper, type SortingState } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { z } from "zod/v4";
import { apiGetNewsAll, apiNewsDelete } from "../api/news.ts";
import { ContainerWithHeaderAndButtons } from "../component/ContainerWithHeaderAndButtons.tsx";
import { ModalConfirm, ModalNewsEditor } from "../component/modal";
import { Title } from "../component/Title";
import { FullTable } from "../component/table/FullTable.tsx";
import { TableCellBool } from "../component/table/TableCellBool.tsx";
import { TableCellString } from "../component/table/TableCellString.tsx";
import { useUserFlashCtx } from "../hooks/useUserFlashCtx.ts";
import type { NewsEntry } from "../schema/news.ts";
import { commonTableSearchSchema, initPagination } from "../util/table.ts";
import { renderDateTime } from "../util/time.ts";

const newsSchema = commonTableSearchSchema.extend({
	sortColumn: z.enum(["news_id", "title", "created_on", "updated_on"]).optional(),
	published: z.boolean().optional(),
});

export const Route = createFileRoute("/_mod/admin/news")({
	component: AdminNews,
	validateSearch: (search) => newsSchema.parse(search),
});

function AdminNews() {
	const search = Route.useSearch();
	const queryClient = useQueryClient();
	const [pagination, setPagination] = useState(initPagination(search.pageIndex, search.pageSize));
	const [sorting] = useState<SortingState>([{ id: "news_id", desc: true }]);

	const { sendFlash, sendError } = useUserFlashCtx();

	const { data: news, isLoading } = useQuery({
		queryKey: ["newsList"],
		queryFn: async () => {
			return await apiGetNewsAll();
		},
	});

	const onCreate = async () => {
		try {
			const newEntry = await NiceModal.show<NewsEntry>(ModalNewsEditor);
			queryClient.setQueryData(["newsList"], [...(news ?? []), newEntry]);
			sendFlash("success", `Entry created successfully`);
		} catch (e) {
			sendFlash("error", `Error trying to create entry: ${e}`);
		}
	};

	const deleteMutation = useMutation({
		mutationKey: ["deleteNews"],
		mutationFn: async (variables: { news_id: number }) => {
			await apiNewsDelete(variables.news_id);
			return variables.news_id;
		},
		onSuccess: (news_id) => {
			queryClient.setQueryData(
				["newsList"],
				(news ?? []).filter((e) => e.news_id !== news_id),
			);
			sendFlash("success", `Entry deleted successfully`);
		},
		onError: sendError,
	});

	const columns = useMemo(() => {
		const columnHelper = createColumnHelper<NewsEntry>();

		const onDelete = async (entry: NewsEntry) => {
			try {
				const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
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
		};

		const onEdit = async (entry: NewsEntry) => {
			try {
				const editedEntry = await NiceModal.show<NewsEntry>(ModalNewsEditor, {
					entry: entry,
				});
				queryClient.setQueryData(
					["newsList"],
					news?.map((e) => (e.news_id === editedEntry.news_id ? editedEntry : e)),
				);
				sendFlash("success", `Entry updated successfully`);
			} catch (e) {
				sendFlash("error", `Error trying to update entry: ${e}`);
			}
		};

		return [
			columnHelper.accessor("news_id", {
				header: "ID",
				size: 30,
				cell: (info) => {
					return <TableCellString>{info.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("title", {
				header: "Title",
				size: 400,
				cell: (info) => {
					return <TableCellString>{info.getValue()}</TableCellString>;
				},
			}),
			columnHelper.accessor("created_on", {
				header: "Created",
				size: 120,
				cell: (info) => {
					return <TableCellString>{renderDateTime(info.getValue())}</TableCellString>;
				},
			}),
			columnHelper.accessor("updated_on", {
				header: "Updated",
				size: 120,
				cell: (info) => {
					return <TableCellString>{renderDateTime(info.getValue())}</TableCellString>;
				},
			}),
			columnHelper.accessor("is_published", {
				meta: { tooltip: "Published" },
				header: "Pub",
				size: 30,
				cell: (info) => {
					return <TableCellBool enabled={info.getValue()} />;
				},
			}),
			columnHelper.display({
				id: "edit",
				size: 50,
				cell: (info) => {
					return (
						<ButtonGroup fullWidth>
							<Button
								variant={"contained"}
								color={"warning"}
								startIcon={<EditIcon />}
								onClick={async () => {
									await onEdit(info.row.original);
								}}
							>
								Edit
							</Button>
							<Button
								variant={"contained"}
								color={"error"}
								startIcon={<DeleteIcon />}
								onClick={async () => {
									await onDelete(info.row.original);
								}}
							>
								Delete
							</Button>
						</ButtonGroup>
					);
				},
			}),
		];
	}, [deleteMutation, news, queryClient, sendFlash]);

	return (
		<Grid container spacing={2}>
			<Title>News</Title>
			<Grid size={{ xs: 12 }}>
				<ContainerWithHeaderAndButtons
					title={"News Entries"}
					iconLeft={<NewspaperIcon />}
					buttons={[
						<Button
							color={"success"}
							variant={"contained"}
							key={"addButton"}
							onClick={onCreate}
							startIcon={<AddIcon />}
						>
							Create
						</Button>,
					]}
				>
					<FullTable
						data={news ?? []}
						isLoading={isLoading}
						columns={columns}
						pagination={pagination}
						setPagination={setPagination}
						sorting={sorting}
						toOptions={{ from: Route.fullPath }}
					/>
				</ContainerWithHeaderAndButtons>
			</Grid>
		</Grid>
	);
}
