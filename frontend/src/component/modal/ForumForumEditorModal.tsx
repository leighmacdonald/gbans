import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import MenuItem from "@mui/material/MenuItem";
import { useMemo } from "react";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { Privilege } from "../../rpc/person/v1/privilege_pb.ts";
import type { Category, Forum } from "../../rpc/forum/v1/forum_pb.ts";
import { useMutation } from "@connectrpc/connect-query";
import { forumCreate } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
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
				forum_category_id: defaultCategory,
				title: forum?.title ?? "",
				description: forum?.description ?? "",
				ordering: forum?.ordering ? String(forum?.ordering) : "0",
				permission_level: forum?.permissionLevel ?? Privilege.USER,
			},
		});

		const catIds = useMemo(() => {
			return categories.map((c) => c.forumCategoryId);
		}, [categories]);

		return (
			<Dialog {...muiDialogV5(modal)} fullWidth maxWidth={"lg"}>
				<form
					onSubmit={async (e) => {
						e.preventDefault();
						e.stopPropagation();
						await form.handleSubmit();
					}}
				>
					<DialogTitle>Category Editor</DialogTitle>

					<DialogContent>
						<Grid container spacing={2}>
							<Grid size={{ xs: 12 }}>
								<form.AppField
									name={"forum_category_id"}
									children={(field) => {
										return (
											<field.SelectField
												label={"Category"}
												items={catIds}
												renderItem={(catId) => {
													return (
														<MenuItem value={catId} key={`cat-${catId}`}>
															{categories.find((c) => c.forumCategoryId === catId)
																?.title ?? ""}
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
									name={"permission_level"}
									validators={{
										onChange: z.enum(Privilege),
									}}
									children={(field) => {
										return (
											<field.SelectField
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
