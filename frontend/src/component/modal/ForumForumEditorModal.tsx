import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Category, Forum } from "../../rpc/forum/v1/forum_pb.ts";
import { forumCreate } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { Privilege } from "../../rpc/person/v1/privilege_pb.ts";
import { enumValues } from "../../util/lists.ts";

export const ForumForumEditorModal = NiceModal.create(
	({ forum, categories }: { forum?: Forum; categories: Category[] }) => {
		const modal = useModal();
		const { sendError } = useUserFlashCtx();

		const mutation = useMutation(forumCreate, {
			onSuccess: async (resp) => {
				modal.resolve(resp.forum);
				await modal.hide();
			},
			onError: sendError,
		});

		const defaultCategory = forum?.forumCategoryId
			? (categories.find((value) => value.forumCategoryId === forum.forumCategoryId)?.forumCategoryId ??
				categories[0].forumCategoryId)
			: categories[0].forumCategoryId;

		const form = useAppForm({
			onSubmit: async ({ value }) => {
				mutation.mutate({ ...value, ordering: Number(value.ordering) });
			},
			defaultValues: {
				forumCategoryId: defaultCategory,
				title: forum?.title ?? "",
				description: forum?.description ?? "",
				ordering: forum?.ordering ? String(forum?.ordering) : "0",
				permissionLevel: forum?.permissionLevel ?? Privilege.USER,
			},
		});

		return (
			<Dialog {...muiDialogV5(modal)} fullWidth maxWidth={"lg"}>
				<form
					onSubmit={async (e) => {
						e.preventDefault();
						e.stopPropagation();
						await form.handleSubmit();
					}}
				>
					<DialogTitle>Forum Editor</DialogTitle>

					<DialogContent>
						<Grid container spacing={2}>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"forumCategoryId"}
									children={(field) => {
										return (
											<field.SelectForumCategoryField
												label={"Category"}
												items={categories}
												renderItem={(category) => {
													return (
														<MenuItem
															value={category.forumCategoryId}
															key={`cat-${category.forumCategoryId}`}
														>
															{category.title}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"title"}
									validators={{
										onChange: z.string().min(1),
									}}
									children={(field) => {
										return <field.TextField label={"Title"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"description"}
									validators={{
										onChange: z.string().min(1),
									}}
									children={(field) => {
										return <field.TextField label={"Description"} rows={5} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"ordering"}
									validators={{
										onChange: z.string().min(1),
									}}
									children={(field) => {
										return <field.TextField label={"Order"} />;
									}}
								/>
							</Grid>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"permissionLevel"}
									validators={{
										onChange: z.enum(Privilege),
									}}
									children={(field) => {
										return (
											<field.SelectPrivilegeField
												label={"Permissions Required"}
												items={enumValues(Privilege)}
												renderItem={(pl) => {
													return (
														<MenuItem value={pl} key={`pl-${pl}`}>
															{Privilege[pl]}
														</MenuItem>
													);
												}}
											/>
										);
									}}
								/>
							</Grid>
						</Grid>
					</DialogContent>

					<DialogActions>
						<Grid container>
							<Grid size={{ xs: 12 }}>
								<form.AppForm>
									<ButtonGroup>
										<form.ResetButton />
										<form.SubmitButton />
									</ButtonGroup>
								</form.AppForm>
							</Grid>
						</Grid>
					</DialogActions>
				</form>
			</Dialog>
		);
	},
);
