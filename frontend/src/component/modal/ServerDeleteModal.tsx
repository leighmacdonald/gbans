import { useCallback } from 'react';
import Stack from '@mui/material/Stack';
import { apiDeleteServer } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Server } from '../../schema/server.ts';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface DeleteServerModalProps extends ConfirmationModalProps<Server> {
    server: Server;
}

export const ServerDeleteModal = ({ onSuccess, server }: DeleteServerModalProps) => {
    const { sendError, sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(async () => {
        try {
            await apiDeleteServer(server.server_id);
            sendFlash('success', `Deleted successfully`);
            if (onSuccess) {
                onSuccess(server);
            }
        } catch (error) {
            sendError(error);
        }
    }, [server, sendFlash, sendError, onSuccess]);

    return (
        <ConfirmationModal
            id={'modal-server-delete'}
            onAccept={async () => {
                await handleSubmit();
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
