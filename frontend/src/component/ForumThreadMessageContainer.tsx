import { useCallback, useMemo, useState } from 'react';
import DeleteForeverIcon from '@mui/icons-material/DeleteForever';
import EditIcon from '@mui/icons-material/Edit';
import { Divider, IconButton, Theme } from '@mui/material';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useRouteContext } from '@tanstack/react-router';
import { isAfter } from 'date-fns/fp';
import { PermissionLevel, permissionLevelString } from '../api';
import { ForumMessage } from '../api/forum.ts';
import { avatarHashToURL, renderDateTime } from '../util/text.tsx';
import { ForumAvatar } from './ForumAvatar.tsx';
import { ForumRowLink } from './ForumRowLink.tsx';
import { MarkDownRenderer } from './MarkdownRenderer.tsx';
import RouterLink from './RouterLink.tsx';

export const ThreadMessageContainer = ({
    message,
    onDelete
}: {
    message: ForumMessage;
    onDelete: (message: ForumMessage) => Promise<void>;
    isFirstMessage: boolean;
}) => {
    const [edit, setEdit] = useState(false);
    const [updatedMessage, setUpdatedMessage] = useState<ForumMessage>();

    const { hasPermission, profile } = useRouteContext({ from: '/_auth/forums/thread/$forum_thread_id' });
    const theme = useTheme();

    const activeMessage = useMemo(() => {
        if (updatedMessage != undefined) {
            return updatedMessage;
        }
        return message;
    }, [message, updatedMessage]);

    const onUpdate = useCallback((updated: ForumMessage) => {
        setUpdatedMessage(updated);
        setEdit(false);
    }, []);

    const editable = useMemo(() => {
        return profile.steam_id == message.source_id || hasPermission(PermissionLevel.Moderator);
    }, [hasPermission, message.source_id, profile.steam_id]);

    return (
        <Paper elevation={1} id={`${activeMessage.forum_message_id}`}>
            <Grid container>
                <Grid xs={2} padding={2} sx={{ backgroundColor: theme.palette.background.paper }}>
                    <Stack alignItems={'center'}>
                        <ForumAvatar
                            alt={activeMessage.personaname}
                            online={activeMessage.online}
                            src={avatarHashToURL(activeMessage.avatarhash, 'medium')}
                        />

                        <ForumRowLink
                            label={activeMessage.personaname}
                            to={`/profile/${activeMessage.source_id}`}
                            align={'center'}
                        />
                        <Typography variant={'subtitle1'} align={'center'}>
                            {permissionLevelString(activeMessage.permission_level)}
                        </Typography>
                    </Stack>
                </Grid>
                <Grid xs={10}>
                    {edit ? (
                        <MessageEditor
                            message={activeMessage}
                            onUpdate={onUpdate}
                            onCancel={() => {
                                setEdit(false);
                            }}
                        />
                    ) : (
                        <Box>
                            <Grid container direction="row" borderBottom={(theme) => theme.palette.divider}>
                                <Grid xs={6}>
                                    <Stack direction={'row'}>
                                        <Typography variant={'body2'} padding={1}>
                                            {renderDateTime(activeMessage.created_on)}
                                        </Typography>
                                        {isAfter(message.created_on, message.updated_on) && (
                                            <Typography variant={'body2'} padding={1}>
                                                {`Edited: ${renderDateTime(activeMessage.updated_on)}`}
                                            </Typography>
                                        )}
                                    </Stack>
                                </Grid>
                                <Grid xs={6}>
                                    <Stack direction="row" justifyContent="end">
                                        <IconButton
                                            color={'error'}
                                            onClick={async () => {
                                                await onDelete(message);
                                            }}
                                        >
                                            <DeleteForeverIcon />
                                        </IconButton>
                                        {editable && (
                                            <IconButton
                                                title={'Edit Post'}
                                                color={'secondary'}
                                                size={'small'}
                                                onClick={() => {
                                                    setEdit(true);
                                                }}
                                            >
                                                <EditIcon />
                                            </IconButton>
                                        )}
                                        <Typography
                                            padding={1}
                                            component={RouterLink}
                                            variant={'body2'}
                                            to={`#${activeMessage.forum_message_id}`}
                                            textAlign={'right'}
                                            color={(theme: Theme) => {
                                                return theme.palette.text.primary;
                                            }}
                                        >
                                            {`#${activeMessage.forum_message_id}`}
                                        </Typography>
                                    </Stack>
                                </Grid>
                            </Grid>
                            <Grid xs={12} padding={1}>
                                <MarkDownRenderer body_md={activeMessage.body_md} />

                                {activeMessage.signature != '' && (
                                    <>
                                        <Divider />
                                        <MarkDownRenderer body_md={activeMessage.signature} />
                                    </>
                                )}
                            </Grid>
                        </Box>
                    )}
                </Grid>
            </Grid>
        </Paper>
    );
};

const MessageEditor = ({
    onCancel
}: {
    message: ForumMessage;
    onUpdate: (msg: ForumMessage) => void;
    onCancel: () => void;
}) => {
    // const onSubmit = useCallback(
    //     async (values: MessageEditValues) => {
    //         try {
    //             const updated = await apiSaveThreadMessage(message.forum_message_id, values.body_md);
    //             onUpdate(updated);
    //         } catch (e) {
    //             logErr(e);
    //         }
    //     },
    //     [message.forum_message_id, onUpdate]
    // );

    return (
        // <Formik<MessageEditValues>
        //     onSubmit={onSubmit}
        //     initialValues={{ body_md: message.body_md }}
        //     validationSchema={validationSchema}
        //     validateOnBlur={true}
        // >
        <Stack padding={1}>
            {/*<MDBodyField />*/}
            <ButtonGroup>
                <Button variant={'contained'} color={'error'} onClick={onCancel}>
                    Cancel
                </Button>
                {/*<SubmitButton />*/}
            </ButtonGroup>
        </Stack>
        // </Formik>
    );
};
