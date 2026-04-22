import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import { useMutation } from "@connectrpc/connect-query";
import { categoryEdit } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
import type { Category } from "../../rpc/forum/v1/forum_pb.ts";

// interface ForumCategoryEditorProps {
//     initial_forum_category_id?: number;
// }

// const validationSchema = yup.object({
//     title: titleFieldValidator
// });

export const ForumCategoryEditorModal = NiceModal.create(({ category }: { category?: Category }) => {
	const modal = useModal();
	const { sendError } = useUserFlashCtx();

	const mutation = useMutation(categoryEdit, {
		onSuccess: async (resp) => {
			modal.resolve(resp.category);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate({ ...value, ordering: Number(value.ordering) });
		},
		defaultValues: {
			title: category?.title ?? "",
			description: category?.description ?? "",
			ordering: category?.ordering ? String(category.ordering) : "1",
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
				<DialogTitle>Category Editor</DialogTitle>

				<DialogContent>
					<Grid container spacing={2}>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"title"}
								children={(field) => {
									return <field.TextField label={"Title"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"description"}
								children={(field) => {
									return <field.TextField label={"Description"} rows={5} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"ordering"}
								children={(field) => {
									return <field.NumberField label={"Order"} />;
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
});
