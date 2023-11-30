import React, { useState } from 'react';
import Stack from '@mui/material/Stack';
import { ASNBanRecord } from '../../api';
import { Heading } from '../Heading';
import { LoadingSpinner } from '../LoadingSpinner';
import { PersonMessageTable } from '../table/PersonMessageTable';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface UnbanASNModalProps
    extends ConfirmationModalProps<ASNBanRecord> {
    messageId: number;
}

export const MessageContextModal = ({ messageId }: UnbanASNModalProps) => {
    const [selectedMessageIndex] = useState<number>(0);
    const [isLoading] = useState(false);

    return (
        <ConfirmationModal id={'modal-message-context'} fullWidth={true}>
            <Stack spacing={2}>
                <Heading>{`Message Context (#${messageId})`}</Heading>
                <Stack spacing={3} alignItems={'center'}>
                    {(isLoading && <LoadingSpinner />) || (
                        <PersonMessageTable
                            steam_id={''}
                            selectedIndex={selectedMessageIndex}
                        />
                    )}
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
