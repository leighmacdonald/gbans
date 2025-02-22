import { useCallback } from 'react';
import Stack from '@mui/material/Stack';
import { apiDeleteServer, Server } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface DeleteServerModalProps extends ConfirmationModalProps<Server> {
    server: Server;
}

export const ServerDeleteModal = ({ onSuccess, server }: DeleteServerModalProps) => {
    const { sendError, sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        apiDeleteServer(server.server_id)
            .then(() => {
                sendFlash('success', `Deleted successfully`);
                if (onSuccess) {
                    onSuccess(server);
                }
            })
            .catch(sendError);
    }, [server, sendFlash, sendError, onSuccess]);

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
                        Delete Server?: ({server.short_name}) {server.name}
                    </>
                </Heading>
            </Stack>
        </ConfirmationModal>
    );
};
