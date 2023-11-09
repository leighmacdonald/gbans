import React, { useEffect, useState } from 'react';
import Stack from '@mui/material/Stack';
import { apiGetMessageContext, ASNBanRecord, PersonMessage } from '../../api';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import { LoadingSpinner } from '../LoadingSpinner';
import { PersonMessageTable } from '../PersonMessageTable';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface UnbanASNModalProps
    extends ConfirmationModalProps<ASNBanRecord> {
    messageId: number;
}

export const MessageContextModal = ({ messageId }: UnbanASNModalProps) => {
    const [messages, setMessages] = useState<PersonMessage[]>([]);
    const [selectedMessageIndex, setSelectedMessageIndex] = useState<number>(0);
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        if (messageId <= 0) {
            return;
        }
        apiGetMessageContext(messageId)
            .then((resp) => {
                resp.map((r: PersonMessage, i: number) => {
                    if (r.person_message_id == messageId) {
                        setSelectedMessageIndex(i);
                    }
                    return r;
                });
                setMessages(resp);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setIsLoading(false);
            });
    }, [messageId]);

    return (
        <ConfirmationModal id={'modal-message-context'} fullWidth={true}>
            <Stack spacing={2}>
                <Heading>{`Message Context (#${messageId})`}</Heading>
                <Stack spacing={3} alignItems={'center'}>
                    {(isLoading && <LoadingSpinner />) || (
                        <PersonMessageTable
                            messages={messages}
                            selectedIndex={selectedMessageIndex}
                        />
                    )}
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
