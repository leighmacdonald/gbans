import { MouseEvent, useState } from 'react';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import Avatar from '@mui/material/Avatar';
import Box from '@mui/material/Box';
import Card from '@mui/material/Card';
import CardHeader from '@mui/material/CardHeader';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { formatDistance } from 'date-fns';
import { z } from 'zod';
import { apiUpdateBanMessage, BanAppealMessage } from '../api';
import { avatarHashToURL } from '../util/text.tsx';
import { MarkDownRenderer } from './MarkdownRenderer';
import { Buttons } from './field/Buttons.tsx';
import { MarkdownField, mdEditorRef } from './field/MarkdownField.tsx';

interface AppealMessageViewProps {
    message: BanAppealMessage;
    onDelete: (report_message_id: number) => void;
}

export const AppealMessageView = ({ message, onDelete }: AppealMessageViewProps) => {
    const theme = useTheme();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const [editing, setEditing] = useState<boolean>(false);
    const [deleted, setDeleted] = useState<boolean>(false);
    const queryClient = useQueryClient();

    const handleClick = (event: MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };

    const handleClose = () => {
        setAnchorEl(null);
    };

    const mutation = useMutation({
        mutationKey: ['banSteam'],
        mutationFn: async (values: { body_md: string }) => {
            const msg = await apiUpdateBanMessage(message.ban_message_id, values.body_md);

            queryClient.setQueryData(['banMessages', { ban_id: message.ban_id }], (prev: BanAppealMessage[]) => {
                return prev.map((m) => (m.ban_message_id == message.ban_message_id ? msg : m));
            });
            mdEditorRef.current?.setMarkdown('');
            setEditing(false);
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                body_md: value.body_md
            });
        },
        defaultValues: {
            body_md: message.message_md
        }
    });

    if (deleted) {
        return <></>;
    }

    if (editing) {
        return (
            <Box component={Paper} padding={1}>
                <form
                    onSubmit={async (e) => {
                        e.preventDefault();
                        e.stopPropagation();
                        await handleSubmit();
                    }}
                >
                    <Grid container spacing={2} padding={1}>
                        <Grid xs={12}>
                            <Field
                                validators={{
                                    onChange: z.string().min(4)
                                }}
                                name={'body_md'}
                                children={(props) => {
                                    return <MarkdownField {...props} label={'Message'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={12} mdOffset="auto">
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </form>
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
                        <Avatar aria-label="Avatar" src={avatarHashToURL(message.avatarhash)}>
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
