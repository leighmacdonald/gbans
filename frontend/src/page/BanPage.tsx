import React, { useCallback, useMemo, useState, JSX } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import DocumentScannerIcon from '@mui/icons-material/DocumentScanner';
import InfoIcon from '@mui/icons-material/Info';
import { FormControl, Select } from '@mui/material';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import InputLabel from '@mui/material/InputLabel';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import {
    apiCreateBanMessage,
    apiDeleteBanMessage,
    apiSetBanAppealState,
    AppealState,
    AppealStateCollection,
    appealStateString,
    BanAppealMessage,
    BanReasons,
    banTypeString,
    PermissionLevel
} from '../api';
import { AppealMessageView } from '../component/AppealMessageView';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { MDBodyField } from '../component/MDBodyField';
import { ProfileInfoBox } from '../component/ProfileInfoBox';
import { SourceBansList } from '../component/SourceBansList';
import { SteamIDList } from '../component/SteamIDList';
import { ModalBanSteam, ModalUnbanSteam } from '../component/modal';
import { ResetButton, SubmitButton } from '../component/modal/Buttons';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useBan } from '../hooks/useBan';
import { useBanAppealMessages } from '../hooks/useBanAppealMessages';
import { logErr } from '../util/errors';
import { renderDateTime, renderTimeDistance } from '../util/text';

interface NewReplyValues {
    body_md: string;
}

export const BanPage = (): JSX.Element => {
    const [appealState, setAppealState] = useState<AppealState>(
        AppealState.Open
    );
    const [newMessages, setNewMessages] = useState<BanAppealMessage[]>([]);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();
    const { ban_id } = useParams();
    const navigate = useNavigate();
    const id = useMemo(() => Number(ban_id || '0'), [ban_id]);
    const [deletedMessages, setDeletedMessages] = useState<number[]>([]);
    const { data: ban } = useBan(id);
    const { data: messagesServer } = useBanAppealMessages(ban?.ban_id ?? 0);

    const messages = useMemo(() => {
        return [...messagesServer, ...newMessages].filter(
            (m) => !deletedMessages.includes(m.ban_message_id)
        );
    }, [deletedMessages, messagesServer, newMessages]);

    const canPost = useMemo(() => {
        return (
            currentUser.permission_level >= PermissionLevel.Moderator ||
            (ban?.appeal_state == AppealState.Open &&
                ban?.target_id == currentUser.steam_id)
        );
    }, [ban, currentUser]);

    const onSubmit = useCallback(
        async (
            values: NewReplyValues,
            helpers: FormikHelpers<NewReplyValues>
        ) => {
            if (!ban) {
                return;
            }
            try {
                const msg = await apiCreateBanMessage(
                    ban?.ban_id,
                    values.body_md
                );
                setNewMessages((prevState) => {
                    return [...prevState, msg];
                });
                helpers.resetForm();
            } catch (e) {
                sendFlash('error', 'Failed to create message');
                logErr(e);
            }
        },
        [ban, sendFlash]
    );

    const onDelete = useCallback(
        async (message_id: number) => {
            try {
                await apiDeleteBanMessage(message_id);
                setDeletedMessages((prevState) => {
                    return [...prevState, message_id];
                });
                sendFlash('success', 'Deleted message successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Failed to delete message');
            }
        },
        [sendFlash]
    );

    const onSaveAppealState = useCallback(() => {
        apiSetBanAppealState(id, appealState)
            .then(() => {
                sendFlash('success', 'Appeal state updated');
            })
            .catch((reason) => {
                sendFlash('error', 'Could not set appeal state');
                logErr(reason);
                return;
            });
    }, [appealState, id, sendFlash]);

    const onUnban = useCallback(async () => {
        await NiceModal.show(ModalUnbanSteam, {
            banId: ban?.ban_id,
            personaName: ban?.target_personaname
        });
    }, [ban?.ban_id, ban?.target_personaname]);

    const onEditBan = useCallback(async () => {
        await NiceModal.show(ModalBanSteam, {
            banId: ban?.ban_id,
            personaName: ban?.target_personaname,
            existing: ban
        });
    }, [ban]);

    return (
        <Grid container spacing={2}>
            <Grid xs={8}>
                <Stack spacing={2}>
                    {canPost && messages.length == 0 && (
                        <ContainerWithHeader title={`Ban Appeal #${id}`}>
                            <Typography
                                variant={'body2'}
                                padding={2}
                                textAlign={'center'}
                            >
                                You can start the appeal process by replying on
                                this form.
                            </Typography>
                        </ContainerWithHeader>
                    )}

                    {ban &&
                        currentUser.permission_level >=
                            PermissionLevel.Moderator && (
                            <SourceBansList
                                steam_id={ban?.source_id}
                                is_reporter={true}
                            />
                        )}

                    {ban &&
                        currentUser.permission_level >=
                            PermissionLevel.Moderator && (
                            <SourceBansList
                                steam_id={ban?.target_id}
                                is_reporter={false}
                            />
                        )}

                    {messages.map((m) => (
                        <AppealMessageView
                            onDelete={onDelete}
                            message={m}
                            key={`ban-appeal-msg-${m.ban_message_id}`}
                        />
                    ))}
                    {canPost && (
                        <Paper elevation={1}>
                            <Formik<NewReplyValues>
                                onSubmit={onSubmit}
                                initialValues={{ body_md: '' }}
                            >
                                <Stack spacing={2} padding={1}>
                                    <MDBodyField />
                                    <ButtonGroup>
                                        <ResetButton />
                                        <SubmitButton />
                                    </ButtonGroup>
                                </Stack>
                            </Formik>
                        </Paper>
                    )}
                    {!canPost && ban && (
                        <Paper elevation={1}>
                            <Typography
                                variant={'body2'}
                                padding={2}
                                textAlign={'center'}
                            >
                                The ban appeal is closed:{' '}
                                {appealStateString(ban.appeal_state)}
                            </Typography>
                        </Paper>
                    )}
                </Stack>
            </Grid>
            <Grid xs={4}>
                <Stack spacing={2}>
                    {ban && <ProfileInfoBox steam_id={ban.target_id} />}
                    {ban && (
                        <ContainerWithHeader
                            title={'Ban Details'}
                            iconLeft={<InfoIcon />}
                        >
                            <List dense={true}>
                                <ListItem>
                                    <ListItemText
                                        primary={'Reason'}
                                        secondary={BanReasons[ban.reason]}
                                    />
                                </ListItem>
                                <ListItem>
                                    <ListItemText
                                        primary={'Ban Type'}
                                        secondary={banTypeString(ban.ban_type)}
                                    />
                                </ListItem>
                                {ban.reason_text != '' && (
                                    <ListItem>
                                        <ListItemText
                                            primary={'Reason (Custom)'}
                                            secondary={ban.reason_text}
                                        />
                                    </ListItem>
                                )}

                                <ListItem>
                                    <ListItemText
                                        primary={'Created At'}
                                        secondary={renderDateTime(
                                            ban.created_on
                                        )}
                                    />
                                </ListItem>
                                <ListItem>
                                    <ListItemText
                                        primary={'Expires At'}
                                        secondary={renderDateTime(
                                            ban.valid_until
                                        )}
                                    />
                                </ListItem>
                                <ListItem>
                                    <ListItemText
                                        primary={'Expires'}
                                        secondary={renderTimeDistance(
                                            ban.valid_until
                                        )}
                                    />
                                </ListItem>
                                {ban &&
                                    currentUser.permission_level >=
                                        PermissionLevel.Moderator && (
                                        <ListItem>
                                            <ListItemText
                                                primary={'Author'}
                                                secondary={ban.source_id.toString()}
                                            />
                                        </ListItem>
                                    )}
                            </List>
                        </ContainerWithHeader>
                    )}

                    {ban && <SteamIDList steam_id={ban?.target_id} />}

                    {ban &&
                        currentUser.permission_level >=
                            PermissionLevel.Moderator &&
                        ban.note != '' && (
                            <ContainerWithHeader
                                title={'Mod Notes'}
                                iconLeft={<DocumentScannerIcon />}
                            >
                                <Typography variant={'body2'} padding={2}>
                                    {ban.note}
                                </Typography>
                            </ContainerWithHeader>
                        )}

                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <ContainerWithHeader
                            title={'Moderation Tools'}
                            iconLeft={<AddModeratorIcon />}
                        >
                            <Stack spacing={2} padding={2}>
                                <Stack direction={'row'} spacing={2}>
                                    <FormControl fullWidth>
                                        <InputLabel id="appeal-status-label">
                                            Appeal Status
                                        </InputLabel>
                                        <Select<AppealState>
                                            value={appealState}
                                            labelId={'appeal-status-label'}
                                            id={'appeal-status'}
                                            label={'Appeal Status'}
                                            onChange={(evt) => {
                                                setAppealState(
                                                    evt.target
                                                        .value as AppealState
                                                );
                                            }}
                                        >
                                            {AppealStateCollection.map((as) => {
                                                return (
                                                    <MenuItem
                                                        value={as}
                                                        key={as}
                                                    >
                                                        {appealStateString(as)}
                                                    </MenuItem>
                                                );
                                            })}
                                        </Select>
                                    </FormControl>
                                    <Button
                                        variant={'contained'}
                                        onClick={onSaveAppealState}
                                    >
                                        Apply Status
                                    </Button>
                                </Stack>

                                {ban && ban?.report_id > 0 && (
                                    <Button
                                        fullWidth
                                        color={'secondary'}
                                        variant={'contained'}
                                        onClick={() => {
                                            navigate(
                                                `/report/${ban?.report_id}`
                                            );
                                        }}
                                    >
                                        View Report #{ban?.report_id}
                                    </Button>
                                )}
                                <Button
                                    variant={'contained'}
                                    color={'secondary'}
                                    component={Link}
                                    href={`https://logs.viora.sh/messages?q[for_player]=${ban?.target_id.toString()}`}
                                >
                                    Ext. Chat Logs
                                </Button>
                                <ButtonGroup fullWidth variant={'contained'}>
                                    <Button
                                        color={'warning'}
                                        onClick={onEditBan}
                                    >
                                        Edit Ban
                                    </Button>
                                    <Button color={'success'} onClick={onUnban}>
                                        Unban
                                    </Button>
                                </ButtonGroup>
                            </Stack>
                        </ContainerWithHeader>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};
