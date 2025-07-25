import { MouseEvent, useState } from 'react';
import MoreVertIcon from '@mui/icons-material/MoreVert';
import Avatar from '@mui/material/Avatar';
import ButtonGroup from '@mui/material/ButtonGroup';
import Card from '@mui/material/Card';
import CardHeader from '@mui/material/CardHeader';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import { useTheme } from '@mui/material/styles';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { formatDistance } from 'date-fns';
import { z } from 'zod/v4';
import { apiDeleteReportMessage, apiUpdateReportMessage } from '../api';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { reportMessagesQueryOptions } from '../queries/reportMessages.ts';
import { ReportMessage } from '../schema/report.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { MarkDownRenderer } from './MarkdownRenderer';
import { mdEditorRef } from './form/field/MarkdownField.tsx';

export interface ReportMessageViewProps {
    message: ReportMessage;
}

export const ReportMessageView = ({ message }: ReportMessageViewProps) => {
    const theme = useTheme();
    const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const { sendFlash, sendError } = useUserFlashCtx();
    const queryClient = useQueryClient();

    const [editing, setEditing] = useState<boolean>(false);
    const [deleted, setDeleted] = useState<boolean>(false);

    const deleteMessageMutation = useMutation({
        mutationFn: async ({ message_id }: { message_id: number }) => {
            return await apiDeleteReportMessage(message_id);
        },
        onSuccess: (_, { message_id }) => {
            queryClient.setQueryData(
                reportMessagesQueryOptions(message.report_id).queryKey,
                (messages: ReportMessage[]) => (messages ?? []).filter((m) => m.report_message_id != message_id)
            );
            sendFlash('success', 'Deleted message successfully');
        },
        onError: sendError
    });

    const onDelete = async (message_id: number) => {
        deleteMessageMutation.mutate({ message_id });
    };

    const mutation = useMutation({
        mutationKey: ['reportMessage'],
        mutationFn: async (values: { body_md: string }) => {
            return await apiUpdateReportMessage(message.report_message_id, values.body_md);
        },
        onSuccess: (msg) => {
            queryClient.setQueryData(
                reportMessagesQueryOptions(message.report_id).queryKey,
                (msgs: ReportMessage[]) => {
                    return msgs.map((m) => {
                        return m.report_message_id == msg.report_message_id ? msg : m;
                    });
                }
            );
            mdEditorRef.current?.setMarkdown('');
            setEditing(false);
            sendFlash('success', 'Edited message successfully');
        },
        onError: sendError
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                body_md: value.body_md
            });
        },
        defaultValues: {
            body_md: message.message_md
        }
    });

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
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <Paper>
                    <Grid container spacing={2} padding={1}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'body_md'}
                                validators={{
                                    onChange: z.string().min(3)
                                }}
                                children={(field) => {
                                    return <field.MarkdownField label={'Message'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <ButtonGroup>
                                    <form.ResetButton />
                                    <form.SubmitButton />
                                </ButtonGroup>
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </Paper>
            </form>
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
                    slotProps={{
                        paper: {
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
                        onClick={async () => {
                            await onDelete(message.report_message_id);
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
