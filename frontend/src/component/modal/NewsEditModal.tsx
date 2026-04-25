import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import PersonIcon from "@mui/icons-material/Person";
import { Dialog, DialogActions, DialogContent, DialogTitle } from "@mui/material";
import ButtonGroup from "@mui/material/ButtonGroup";
import Grid from "@mui/material/Grid";
import { z } from "zod/v4";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Article } from "../../rpc/news/v1/news_pb.ts";
import { create, edit } from "../../rpc/news/v1/news-NewsService_connectquery.ts";
import { Heading } from "../Heading";

export const NewsEditModal = NiceModal.create(({ entry }: { entry?: Article }) => {
	const modal = useModal();
	const { sendError, sendFlash } = useUserFlashCtx();

	const editMutation = useMutation(edit, {
		onSuccess: async (entry) => {
			modal.resolve(entry.article);
			sendFlash("success", "News edited successfully.");
			await modal.hide();
		},
		onError: sendError,
	});

	const createMutation = useMutation(create, {
		onSuccess: async (entry) => {
			modal.resolve(entry.article);
			sendFlash("success", "News created successfully.");
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			if (entry?.newsId) {
				editMutation.mutate(value);
			} else {
				createMutation.mutate(value);
			}
		},
		defaultValues: {
			title: entry?.title ?? "",
			body_md: entry?.bodyMd ?? "",
			is_published: entry?.isPublished ?? false,
		},
		validators: {
			onSubmit: z.object({
				body_md: z.string().min(10),
				title: z.string().min(4),
				is_published: z.boolean(),
			}),
		},
	});

	return (
		<Dialog {...muiDialogV5(modal)} fullWidth maxWidth={"sm"}>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<DialogTitle component={Heading} iconLeft={<PersonIcon />}>
					News Editor
				</DialogTitle>
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
								name={"body_md"}
								children={(field) => {
									return <field.MarkdownField label={"Body"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"is_published"}
								children={(field) => {
									return <field.CheckboxField label={"Is Published"} />;
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
