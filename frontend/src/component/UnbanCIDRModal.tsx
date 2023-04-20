import React, { useCallback, useEffect, useState } from 'react';
import Stack from '@mui/material/Stack';
import FormControl from '@mui/material/FormControl';
import TextField from '@mui/material/TextField';
import { apiDeleteCIDRBan, IAPIBanCIDRRecord } from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

export interface UnbanCIDRModalProps
    extends ConfirmationModalProps<IAPIBanCIDRRecord> {
    record: IAPIBanCIDRRecord;
}

export const UnbanCIDRModal = ({
    open,
    setOpen,
    onSuccess,
    record
}: UnbanCIDRModalProps) => {
    const [reasonText, setReasonText] = useState<string>('');
    const { sendFlash } = useUserFlashCtx();

    useEffect(() => {
        setReasonText('');
    }, [record]);

    const handleSubmit = useCallback(() => {
        if (reasonText == '') {
            sendFlash('error', 'Reason cannot be empty');
            return;
        }
        apiDeleteCIDRBan(record.net_id, reasonText)
            .then(() => {
                sendFlash('success', `Unbanned successfully`);
                onSuccess && onSuccess(record);
            })
            .catch((err) => {
                sendFlash('error', `Failed to unban: ${err}`);
            });
    }, [reasonText, record, sendFlash, onSuccess]);

    return (
        <ConfirmationModal
            open={open}
            setOpen={setOpen}
            onSuccess={() => {
                setOpen(false);
            }}
            onCancel={() => {
                setOpen(false);
            }}
            onAccept={() => {
                handleSubmit();
            }}
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
        >
            <Stack spacing={2}>
                <Heading>
                    <>
                        Unban CIDR (#{record.net_id}):
                        {record.cidr.IP}
                    </>
                </Heading>
                <Stack spacing={3} alignItems={'center'}>
                    <FormControl fullWidth>
                        <TextField
                            label={'Reason'}
                            id={'reasonText'}
                            value={reasonText}
                            onChange={(evt) => {
                                setReasonText(evt.target.value);
                            }}
                        />
                    </FormControl>
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
