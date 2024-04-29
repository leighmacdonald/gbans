import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import BlockIcon from '@mui/icons-material/Block';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { Heading } from '../Heading';

// interface CIDRBlockEditorProps {
//     source?: CIDRBlockSource;
// }
//
// interface CIDRBlockEditorValues {
//     cidr_block_source_id: number;
//     name: string;
//     url: string;
//     enabled: boolean;
// }

// const validationSchema = yup.object({
//     name: NameFieldValidator,
//     url: URLFieldValidator,
//     enabled: EnabledFieldValidator
// });

export const CIDRBlockEditorModal = NiceModal.create((/**{ source }: CIDRBlockEditorProps*/) => {
    const modal = useModal();

    // const onSave = useCallback(
    //     async (values: CIDRBlockEditorValues) => {
    //         try {
    //             if (values.cidr_block_source_id > 0) {
    //                 const resp = await apiUpdateCIDRBlockSource(values.cidr_block_source_id, values.name, values.url, values.enabled);
    //                 modal.resolve(resp);
    //             } else {
    //                 const resp = await apiCreateCIDRBlockSource(values.name, values.url, values.enabled);
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
        // <Formik<Omit<CIDRBlockSource, 'created_on' | 'updated_on'>>
        //     onSubmit={onSave}
        //     validationSchema={validationSchema}
        //     initialValues={{
        //         cidr_block_source_id: source?.cidr_block_source_id ?? 0,
        //         name: source?.name ?? '',
        //         url: source?.url ?? '',
        //         enabled: source?.enabled ?? true
        //     }}
        // >
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'md'}>
            <DialogTitle component={Heading} iconLeft={<BlockIcon />}>
                CIDR Block Source Editor
            </DialogTitle>
            <DialogContent>
                <Stack spacing={2}>
                    {/*<NameField />*/}
                    {/*<EnabledField />*/}
                    {/*<URLField />*/}
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

export default CIDRBlockEditorModal;
