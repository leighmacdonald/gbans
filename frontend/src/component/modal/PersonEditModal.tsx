import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import PersonIcon from '@mui/icons-material/Person';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { Person } from '../../api';
import { Heading } from '../Heading';

export interface PersonEditModalProps {
    person: Person;
}

// type PersonEditFormValues = {
//     permission_level: PermissionLevel;
// } & TargetIDInputValue;

export const PersonEditModal = NiceModal.create(({ person }: PersonEditModalProps) => {
    const modal = useModal();

    // const onSave = useCallback(
    //     async (values: PersonEditFormValues) => {
    //         const abortConroller = new AbortController();
    //         try {
    //             const resp = await apiUpdatePlayerPermission(
    //                 person.steam_id,
    //                 {
    //                     permission_level: values.permission_level
    //                 },
    //                 abortConroller
    //             );
    //             modal.resolve(resp);
    //             await modal.hide();
    //         } catch (e) {
    //             modal.reject(e);
    //         }
    //     },
    //     [modal, person.steam_id]
    // );

    return (
        // <Formik<PersonEditFormValues>
        //     onSubmit={onSave}
        //     initialValues={{
        //         permission_level:
        //             person.permission_level ?? PermissionLevel.User,
        //         target_id: person.steam_id
        //     }}
        // >
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'sm'}>
            <DialogTitle component={Heading} iconLeft={<PersonIcon />}>
                Person Editor: {person.personaname}
            </DialogTitle>
            <DialogContent>
                <Stack spacing={2}>
                    {/*<TargetIDField*/}
                    {/*    isReadOnly={!emptyOrNullString(person.steam_id)}*/}
                    {/*/>*/}
                    {/*<PermissionLevelField />*/}
                </Stack>
            </DialogContent>
            <DialogActions>
                {/*<CancelButton />*/}
                {/*<SubmitButton />*/}
            </DialogActions>
        </Dialog>
        //</Formik>
    );
});

export default PersonEditModal;
