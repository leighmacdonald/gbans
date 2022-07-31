import { UserMessage, UserProfile } from '../api';
import useTheme from '@mui/material/styles/useTheme';
import React, { MouseEvent, useState } from 'react';
import Box from '@mui/material/Box';
import Paper from '@mui/material/Paper';
import { MDEditor } from './MDEditor';
import { formatDistance, parseJSON } from 'date-fns';
import Card from '@mui/material/Card';
import CardHeader from '@mui/material/CardHeader';
import Avatar from '@mui/material/Avatar';
import IconButton from '@mui/material/IconButton';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import CardContent from '@mui/material/CardContent';
import { RenderedMarkdownBox } from './RenderedMarkdownBox';
import { renderMarkdown } from '../api/wiki';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';

export interface UserMessageViewProps {
    author: UserProfile;
    message: UserMessage;
    onSave: (message: UserMessage) => void;
    onDelete: (report_message_id: number) => void;
}

export const UserMessageView = ({
    author,
    message,
    onSave,
    onDelete
}: UserMessageViewProps) => {
    const theme = useTheme();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const [editing, setEditing] = useState<boolean>(false);
    const [deleted, setDeleted] = useState<boolean>(false);
    const handleClick = (event: MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };
    if (deleted) {
        return <></>;
    }
    if (editing) {
        return (
            <Box component={Paper} padding={1}>
                <MDEditor
                    cancelEnabled
                    onCancel={() => {
                        setEditing(false);
                    }}
                    initialBodyMDValue={message.contents}
                    onSave={(body_md) => {
                        const newMsg = { ...message, contents: body_md };
                        onSave(newMsg);
                        message = newMsg;
                        setEditing(false);
                    }}
                />
            </Box>
        );
    } else {
        let d1 = formatDistance(parseJSON(message.created_on), new Date(), {
            addSuffix: true
        });
        if (message.updated_on != message.created_on) {
            d1 = `${d1} (edited: ${formatDistance(
                parseJSON(message.updated_on),
                new Date(),
                {
                    addSuffix: true
                }
            )})`;
        }
        return (
            <Card elevation={1}>
                <CardHeader
                    sx={{
                        backgroundColor: theme.palette.background.paper
                    }}
                    avatar={
                        <Avatar aria-label="Avatar" src={author.avatar}>
                            ?
                        </Avatar>
                    }
                    action={
                        <IconButton aria-label="Actions" onClick={handleClick}>
                            <MoreVertIcon />
                        </IconButton>
                    }
                    title={author.name}
                    subheader={d1}
                />
                <CardContent>
                    <RenderedMarkdownBox
                        bodyMd={renderMarkdown(message.contents)}
                    />
                </CardContent>
                <Menu
                    anchorEl={anchorEl}
                    id="message-menu"
                    open={open}
                    onClose={handleClose}
                    onClick={handleClose}
                    PaperProps={{
                        elevation: 0,
                        sx: {
                            overflow: 'visible',
                            filter: 'drop-shadow(0px 2px 8px rgba(0,0,0,0.32))',
                            mt: 1.5,
                            '& .MuiAvatar-root': {
                                width: 32,
                                height: 32,
                                ml: -0.5,
                                mr: 1
                            },
                            '&:before': {
                                content: '""',
                                display: 'block',
                                position: 'absolute',
                                top: 0,
                                right: 14,
                                width: 10,
                                height: 10,
                                bgcolor: 'background.paper',
                                transform: 'translateY(-50%) rotate(45deg)',
                                zIndex: 0
                            }
                        }
                    }}
                    transformOrigin={{ horizontal: 'right', vertical: 'top' }}
                    anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
                >
                    <MenuItem
                        onClick={() => {
                            setEditing(true);
                        }}
                    >
                        Edit
                    </MenuItem>
                    <MenuItem
                        onClick={() => {
                            onDelete(message.message_id);
                            setDeleted(true);
                        }}
                    >
                        Delete
                    </MenuItem>
                </Menu>
            </Card>
        );
    }
};
