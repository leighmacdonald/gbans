import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CloudDoneIcon from '@mui/icons-material/CloudDone';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { Heading } from '../Heading';

// interface CIDRWhitelistEditorProps {
//     source?: CIDRBlockWhitelist;
// }
//
// interface CIDRWhitelistEditorValues {
//     cidr_block_whitelist_id: number;
//     ip: string;
// }
//
// const validationSchema = yup.object({
//     ip: ipFieldValidator
// });

export const CIDRWhitelistEditorModal = NiceModal.create((/**{ source }: CIDRWhitelistEditorProps*/) => {
    const modal = useModal();

    // const onSave = useCallback(
    //     async (values: CIDRWhitelistEditorValues) => {
    //         try {
    //             if (values.cidr_block_whitelist_id > 0) {
    //                 const resp = await apiUpdateCIDRBlockWhitelist(
    //                     values.cidr_block_whitelist_id,
    //                     values.ip
    //                 );
    //                 modal.resolve(resp);
    //             } else {
    //                 const resp = await apiCreateCIDRBlockWhitelist(
    //                     values.ip
    //                 );
    //                 modal.resolve(resp);
    //             }
    //             await modal.hide();
    //         } catch (e) {
    //             modal.reject(e);
    //         }
    //     },
    //     [modal]
    // );

    return (
        // <Formik<
        //     Omit<CIDRWhitelistEditorValues, 'created_on' | 'updated_on'>
        // >
        //     onSubmit={onSave}
        //     validationSchema={validationSchema}
        //     initialValues={{
        //         cidr_block_whitelist_id:
        //             source?.cidr_block_whitelist_id ?? 0,
        //         ip: source?.address ?? ''
        //     }}
        // >
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'md'}>
            <DialogTitle component={Heading} iconLeft={<CloudDoneIcon />}>
                CIDR Block Whitelist Editor
            </DialogTitle>
            <DialogContent>
                <Stack spacing={2}>{/*<IPField />*/}</Stack>
            </DialogContent>
            <DialogActions>
                {/*<CancelButton />*/}
                {/*<SubmitButton />*/}
            </DialogActions>
        </Dialog>
        // </Formik>
    );
});

export default CIDRWhitelistEditorModal;
