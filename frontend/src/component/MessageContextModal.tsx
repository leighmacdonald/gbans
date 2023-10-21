import React, { useEffect, useState } from 'react';
import Stack from '@mui/material/Stack';
import { apiGetMessageContext, IAPIBanASNRecord, PersonMessage } from '../api';
import { logErr } from '../util/errors';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { Heading } from './Heading';
import { LoadingSpinner } from './LoadingSpinner';
import { PersonMessageTable } from './PersonMessageTable';

export interface UnbanASNModalProps
    extends ConfirmationModalProps<IAPIBanASNRecord> {
    messageId: number;
}

export const MessageContextModal = ({
    open,
    setOpen,
    messageId
}: UnbanASNModalProps) => {
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
        <ConfirmationModal
            fullWidth={true}
            open={open}
            setOpen={setOpen}
            onSuccess={() => {
                setOpen(false);
            }}
            onAccept={() => {
                setOpen(false);
            }}
        >
            <Stack spacing={2}>
                <Heading>
                    <>Message Context (#{messageId})</>
                </Heading>
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
