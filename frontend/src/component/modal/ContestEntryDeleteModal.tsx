import NiceModal, { muiDialogV5, useModal } from "@ebay/nice-modal-react";
import Dialog from "@mui/material/Dialog";
import DialogActions from "@mui/material/DialogActions";
import DialogContent from "@mui/material/DialogContent";
import DialogTitle from "@mui/material/DialogTitle";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";

export const ContestEntryDeleteModal = NiceModal.create(({ contestEntryId }: { contestEntryId: string }) => {
	const modal = useModal();

	// const onSubmit = useCallback(async () => {
	//     try {
	//         await apiContestEntryDelete(contest_entry_id);
	//         modal.resolve();
	//     } catch (e) {
	//         modal.reject(e);
	//     } finally {
	//         await modal.hide();
	//     }
	// }, [contest_entry_id, modal]);

	return (
		// <Formik initialValues={{}} onSubmit={onSubmit}>
		<Dialog {...muiDialogV5(modal)}>
			<DialogTitle>Are you sure you want to delete contest entry? ({contestEntryId})</DialogTitle>

			<DialogContent>
				<Stack spacing={2}>
					<Typography variant={"body1"}>
						This is irreversible and will also remove user vote history for the entry
					</Typography>
				</Stack>
			</DialogContent>

			<DialogActions>
				{/*<CancelButton />*/}
				{/*<SubmitButton />*/}
			</DialogActions>
		</Dialog>
		// </Formik>
	);
});
