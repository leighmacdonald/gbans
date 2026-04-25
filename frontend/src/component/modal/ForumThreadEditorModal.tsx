import { useMutation } from "@connectrpc/connect-query";
import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import ButtonGroup from "@mui/material/ButtonGroup";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Grid from "@mui/material/Grid";
import { useCallback } from "react";
import { useAppForm } from "../../contexts/formContext.tsx";
import { useUserFlashCtx } from "../../hooks/useUserFlashCtx.ts";
import type { Thread } from "../../rpc/forum/v1/forum_pb.ts";
import { threadDelete, threadEdit } from "../../rpc/forum/v1/forum-ForumService_connectquery.ts";
import { logErr } from "../../util/errors";
import { ConfirmationModal } from "./ConfirmationModal.tsx";

export const ForumThreadEditorModal = NiceModal.create(({ thread }: { thread: Thread }) => {
	const modal = useModal();
	const confirmModal = useModal(ConfirmationModal);
	const { sendFlash, sendError } = useUserFlashCtx();

	const deleteMutation = useMutation(threadDelete, {
		onSuccess: () => {
			sendFlash("success", "Deleted thread successfully");
		},
		onError: (err) => {
			logErr(err);
		},
	});

	const onDelete = useCallback(async () => {
		try {
			const confirmed = await confirmModal.show({
				title: "Confirm Thread Deletion",
				children: "All messages will be deleted",
			});
			if (confirmed) {
				await confirmModal.hide();
				await deleteMutation.mutateAsync({ forumThreadId: thread.forumThreadId });
				thread.forumThreadId = 0;
				modal.resolve(thread);
				await modal.hide();
			} else {
				await confirmModal.hide();
			}
		} catch (e) {
			logErr(e);
		}
	}, [confirmModal, modal, thread, deleteMutation.mutateAsync]);

	const mutation = useMutation(threadEdit, {
		onSuccess: async (resp) => {
			modal.resolve(resp.thread);
			await modal.hide();
		},
		onError: sendError,
	});

	const form = useAppForm({
		onSubmit: async ({ value }) => {
			mutation.mutate({ ...value });
		},
		defaultValues: {
			title: thread.title,
			sticky: thread.sticky,
			locked: thread.locked,
		},
	});

	return (
		<Dialog {...muiDialogV5(modal)} fullWidth>
			<form
				onSubmit={async (e) => {
					e.preventDefault();
					e.stopPropagation();
					await form.handleSubmit();
				}}
			>
				<DialogTitle>{`Edit Thread #${thread.forumThreadId}`}</DialogTitle>

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
								name={"sticky"}
								children={(field) => {
									return <field.CheckboxField label={"Stickied"} />;
								}}
							/>
						</Grid>
						<Grid size={{ xs: 12 }}>
							<form.AppField
								name={"locked"}
								children={(field) => {
									return <field.CheckboxField label={"Locked"} />;
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
									<form.ClearButton onClick={onDelete} />
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
