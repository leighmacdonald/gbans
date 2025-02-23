import { useState, MouseEvent } from 'react';
import ScrollableFeed from 'react-scrollable-feed';
import { useModal } from '@ebay/nice-modal-react';
import BlockIcon from '@mui/icons-material/Block';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import Grid from '@mui/material/Grid2';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { ChatLog } from '../api';
import { useQueueCtx } from '../hooks/useQueueCtx.ts';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { QueueChatName } from './QueueChatName.tsx';
import { QueuePurgeModal } from './modal/QueuePurgeModal.tsx';
import { QueueStatusModal } from './modal/QueueStatusModal.tsx';

export const QueueChatMessageContainer = ({ showControls }: { showControls: boolean }) => {
    const { messages, isReady } = useQueueCtx();

    if (!isReady) {
        return <LoadingPlaceholder />;
    }

    return (
        <ScrollableFeed>
            {messages.map((message, i) => {
                return (
                    <div key={`${message.message_id}-${i}`} style={{ overflowWrap: 'break-word', paddingRight: 4 }}>
                        <QueueChatMessage message={message} showControls={showControls} />
                    </div>
                );
            })}
        </ScrollableFeed>
    );
};

const QueueChatMessage = ({ message, showControls }: { message: ChatLog; showControls: boolean }) => {
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);

    const purgeModal = useModal(QueuePurgeModal);
    const statusModal = useModal(QueueStatusModal);

    const handleClick = (event: MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const showPurge = async () => {
        await purgeModal.show({ message });
        handleClose();
    };

    const showBan = async () => {
        await statusModal.show({ steam_id: message.steam_id });
        handleClose();
    };

    return (
        <Grid container key={`queuemsg-${message.message_id}-id`} spacing={1}>
            <Grid size={{ xs: 'auto' }} paddingLeft={2}>
                <QueueChatName
                    onClick={(e: MouseEvent<HTMLElement>) => {
                        e.preventDefault();
                        handleClick(e);
                    }}
                    personaname={message.personaname}
                    steam_id={message.steam_id}
                    avatarhash={message.avatarhash}
                />

                <Menu
                    id="demo-positioned-menu"
                    aria-labelledby="demo-positioned-button"
                    anchorEl={anchorEl}
                    open={open}
                    onClose={handleClose}
                    anchorOrigin={{
                        vertical: 'top',
                        horizontal: 'left'
                    }}
                    transformOrigin={{
                        vertical: 'top',
                        horizontal: 'left'
                    }}
                >
                    <MenuItem>
                        <QueueChatName
                            personaname={message.personaname}
                            steam_id={message.steam_id}
                            avatarhash={message.avatarhash}
                        />
                    </MenuItem>
                    <MenuItem>
                        <Typography variant={'body1'} padding={1}>
                            {message.body_md}
                        </Typography>
                    </MenuItem>
                    {showControls && (
                        <MenuItem onClick={showBan}>
                            <ListItemIcon>
                                <BlockIcon fontSize="small" color={'error'} />
                            </ListItemIcon>
                            <ListItemText>Set Chat Status</ListItemText>
                        </MenuItem>
                    )}
                    {showControls && (
                        <MenuItem onClick={showPurge}>
                            <ListItemIcon>
                                <DeleteForeverIcon color={'warning'} fontSize="small" />
                            </ListItemIcon>
                            <ListItemText>Purge Message(s)</ListItemText>
                        </MenuItem>
                    )}
                </Menu>
            </Grid>

            <Grid size={{ xs: 10 }} paddingRight={2}>
                <Stack direction={'row'}>
                    <div style={{ overflowWrap: 'break-word', width: '95%' }}>
                        <Typography variant="body1" color="text" sx={{ borderLeft: '2px solid #666', paddingLeft: 1 }}>
                            {message.body_md}
                        </Typography>
                    </div>
                </Stack>
            </Grid>
        </Grid>
    );
};
