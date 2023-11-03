import React, { useCallback } from 'react';
import Stack from '@mui/material/Stack';
import { apiDeleteServer, Server } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface DeleteServerModalProps extends ConfirmationModalProps<Server> {
    server: Server;
}

export const ServerDeleteModal = ({
    onSuccess,
    server
}: DeleteServerModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        apiDeleteServer(server.server_id)
            .then(() => {
                sendFlash('success', `Deleted successfully`);
                onSuccess && onSuccess(server);
            })
            .catch((err) => {
                sendFlash('error', `Failed to unban: ${err}`);
            });
    }, [server, sendFlash, onSuccess]);

    return (
        <ConfirmationModal
            id={'modal-server-delete'}
            onAccept={() => {
                handleSubmit();
            }}
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
        >
            <Stack spacing={2}>
                <Heading>
                    <>
                        Delete Server?: ({server.server_name}){' '}
                        {server.server_name_long}
                    </>
                </Heading>
            </Stack>
        </ConfirmationModal>
    );
};
