import ScrollableFeed from 'react-scrollable-feed';
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { ServerQueueMessage } from '../api';
import { useQueueCtx } from '../hooks/useQueueCtx.ts';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { QueueChatName } from './QueueChatName.tsx';

export const QueueChatMessageContainer = ({ showControls }: { showControls: boolean }) => {
    const { messages, isReady } = useQueueCtx();

    if (!isReady) {
        return <LoadingPlaceholder />;
    }

    return (
        <ScrollableFeed>
            {messages.map((message, i) => {
                return (
                    <QueueChatMessage
                        message={message}
                        key={`${message.message_id}-${i}`}
                        showControls={showControls}
                    />
                );
            })}
        </ScrollableFeed>
    );
};

const QueueChatMessage = ({ message, showControls }: { message: ServerQueueMessage; showControls: boolean }) => {
    return (
        <Grid container key={`${message.message_id}-id`} spacing={1} overflow={'hidden'}>
            <Grid xs={2}>
                <QueueChatName
                    personaname={message.personaname}
                    steam_id={message.steam_id}
                    avatarhash={message.avatarhash}
                />
            </Grid>
            <Grid xs={10}>
                <Stack direction={'row'}>
                    {showControls && (
                        <IconButton
                            color={'primary'}
                            sx={{
                                size: '10',
                                padding: 0,
                                borderLeft: '1px solid #666',
                                borderRadius: 0,
                                paddingLeft: 1
                            }}
                        >
                            <ManageAccountsIcon color={'error'} />
                        </IconButton>
                    )}
                    <Typography variant="body1" color="text" sx={{ borderLeft: '1px solid #666', paddingLeft: 1 }}>
                        {message.body_md}
                    </Typography>
                </Stack>
            </Grid>
        </Grid>
    );
};
