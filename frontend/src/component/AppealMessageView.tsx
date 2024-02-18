import { MouseEvent, useCallback, useState } from 'react';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import ButtonGroup from '@mui/material/ButtonGroup';
import Card from '@mui/material/Card';
import CardHeader from '@mui/material/CardHeader';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import { formatDistance } from 'date-fns';
import { Formik } from 'formik';
import { apiUpdateBanMessage, BanAppealMessage } from '../api';
import { logErr } from '../util/errors';
import { avatarHashToURL } from '../util/text.tsx';
import { MDBodyField } from './MDBodyField';
import { MarkDownRenderer } from './MarkdownRenderer';
import { ResetButton, SubmitButton } from './modal/Buttons';

interface AppealMessageViewProps {
    message: BanAppealMessage;
    onDelete: (report_message_id: number) => void;
}

interface AppealMessageValues {
    body_md: string;
}

export const AppealMessageView = ({
    message,
    onDelete
}: AppealMessageViewProps) => {
    const theme = useTheme();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const [editing, setEditing] = useState<boolean>(false);
    const [deleted, setDeleted] = useState<boolean>(false);

    const onSubmit = useCallback(
        async (values: AppealMessageValues) => {
            try {
                await apiUpdateBanMessage(
                    message.ban_message_id,
                    values.body_md
                );
                message.message_md = values.body_md;
                setEditing(false);
            } catch (e) {
                logErr(e);
            }
        },
        [message]
    );

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
                <Formik<AppealMessageValues>
                    onSubmit={onSubmit}
                    initialValues={{ body_md: message.message_md }}
                >
                    <Stack spacing={1}>
                        <MDBodyField />

                        <ButtonGroup>
                            <ResetButton />
                            <SubmitButton />
                        </ButtonGroup>
                    </Stack>
                </Formik>
            </Box>
        );
    } else {
        const d1 = formatDistance(message.created_on, new Date(), {
            addSuffix: true
        });
        return (
            <Card elevation={1}>
                <CardHeader
                    sx={{
                        backgroundColor: theme.palette.background.paper
                    }}
                    avatar={
                        <Avatar
                            aria-label="Avatar"
                            src={avatarHashToURL(message.avatarhash)}
                        >
                            ?
                        </Avatar>
                    }
                    action={
                        <IconButton aria-label="Actions" onClick={handleClick}>
                            <MoreVertIcon />
                        </IconButton>
                    }
                    title={message.personaname}
                    subheader={d1}
                />

                <MarkDownRenderer body_md={message.message_md} />

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
                            onDelete(message.ban_message_id);
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
