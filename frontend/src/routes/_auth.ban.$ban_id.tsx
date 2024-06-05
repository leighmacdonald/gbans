import { useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import DocumentScannerIcon from '@mui/icons-material/DocumentScanner';
import EditIcon from '@mui/icons-material/Edit';
import InfoIcon from '@mui/icons-material/Info';
import UndoIcon from '@mui/icons-material/Undo';
import { FormControl, Select } from '@mui/material';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import InputLabel from '@mui/material/InputLabel';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import {
    apiCreateBanMessage,
    apiDeleteBanMessage,
    apiGetBanMessages,
    apiGetBanSteam,
    apiSetBanAppealState,
    AppealState,
    AppealStateCollection,
    appealStateString,
    BanReasons,
    banTypeString,
    PermissionLevel
} from '../api';
import { AppealMessageView } from '../component/AppealMessageView.tsx';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ErrorDetails } from '../component/ErrorDetails.tsx';
import { MarkDownRenderer } from '../component/MarkdownRenderer.tsx';
import { ProfileInfoBox } from '../component/ProfileInfoBox.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { SourceBansList } from '../component/SourceBansList.tsx';
import { SteamIDList } from '../component/SteamIDList.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { MarkdownField, mdEditorRef } from '../component/field/MarkdownField.tsx';
import { ModalBanSteam, ModalUnbanSteam } from '../component/modal';
import { AppError, ErrorCode } from '../error.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { logErr } from '../util/errors.ts';
import { renderDateTime, renderTimeDistance } from '../util/text.tsx';

export const Route = createFileRoute('/_auth/ban/$ban_id')({
    component: BanPage,
    loader: ({ context, abortController, params }) => {
        const { ban_id } = params;
        return context.queryClient.fetchQuery({
            queryKey: ['ban', { ban_id }],
            queryFn: async () => {
                const ban = await apiGetBanSteam(Number(ban_id), true, abortController);
                if (!ban) {
                    throw new AppError(ErrorCode.NotFound);
                }
                return ban;
            }
        });
    },
    errorComponent: (e) => {
        return <ErrorDetails error={e.error} />;
    }
});

function BanPage() {
    const [appealState, setAppealState] = useState<AppealState>(AppealState.Open);
    const { permissionLevel, profile } = useRouteContext({ from: '/_auth/ban/$ban_id' });
    const ban = Route.useLoaderData();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();
    const queryClient = useQueryClient();

    const { data: messages } = useQuery({
        queryKey: ['banMessages', { ban_id: ban.ban_id }],
        queryFn: async () => {
            return await apiGetBanMessages(ban.ban_id);
        }
    });

    const canPost = useMemo(() => {
        return (
            permissionLevel() >= PermissionLevel.Moderator ||
            (ban?.appeal_state == AppealState.Open && ban?.target_id == profile.steam_id)
        );
    }, [ban?.appeal_state, ban?.target_id, permissionLevel, profile.steam_id]);

    const onDelete = useCallback(
        async (message_id: number) => {
            try {
                await apiDeleteBanMessage(message_id);
                queryClient.setQueryData(
                    ['banMessages', { ban_id: ban.ban_id }],
                    messages?.filter((m) => {
                        return m.ban_message_id != message_id;
                    })
                );
                sendFlash('success', 'Deleted message successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Failed to delete message');
            }
        },
        [ban.ban_id, messages, queryClient, sendFlash]
    );

    const onSaveAppealState = useCallback(() => {
        apiSetBanAppealState(ban.ban_id, appealState)
            .then(() => {
                sendFlash('success', 'Appeal state updated');
            })
            .catch((reason) => {
                sendFlash('error', 'Could not set appeal state');
                logErr(reason);
                return;
            });
    }, [appealState, ban.ban_id, sendFlash]);

    const onUnban = useCallback(async () => {
        await NiceModal.show(ModalUnbanSteam, {
            banId: ban.ban_id,
            personaName: ban?.target_personaname
        });
    }, [ban.ban_id, ban?.target_personaname]);

    const onEditBan = useCallback(async () => {
        await NiceModal.show(ModalBanSteam, {
            banId: ban.ban_id,
            personaName: ban.target_personaname,
            existing: ban
        });
    }, [ban]);

    const expired = useMemo(() => {
        return ban.valid_until ? ban.valid_until < new Date() : true;
    }, [ban.valid_until]);

    const modTools = useMemo(() => {
        return (
            <ContainerWithHeader title={'Moderation Tools'} iconLeft={<AddModeratorIcon />}>
                <Title>Moderation Tools</Title>
                <Stack spacing={2} padding={2}>
                    <Stack direction={'row'} spacing={2}>
                        {!expired && (
                            <>
                                <FormControl fullWidth>
                                    <InputLabel id="appeal-status-label">Appeal Status</InputLabel>
                                    <Select<AppealState>
                                        value={ban?.appeal_state}
                                        labelId={'appeal-status-label'}
                                        id={'appeal-status'}
                                        label={'Appeal Status'}
                                        onChange={(evt) => {
                                            setAppealState(evt.target.value as AppealState);
                                        }}
                                    >
                                        {AppealStateCollection.map((as) => {
                                            return (
                                                <MenuItem value={as} key={as}>
                                                    {appealStateString(as)}
                                                </MenuItem>
                                            );
                                        })}
                                    </Select>
                                </FormControl>

                                <Button variant={'contained'} onClick={onSaveAppealState}>
                                    Apply Status
                                </Button>
                            </>
                        )}
                        {expired && (
                            <Typography variant={'h6'} textAlign={'center'}>
                                Ban Expired
                            </Typography>
                        )}
                    </Stack>

                    {ban && ban?.report_id > 0 && (
                        <Button
                            fullWidth
                            disabled={expired}
                            color={'secondary'}
                            variant={'contained'}
                            onClick={async () => {
                                await navigate({ to: `/report/${ban?.report_id}` });
                            }}
                        >
                            View Report #{ban?.report_id}
                        </Button>
                    )}
                    <Button
                        variant={'contained'}
                        color={'secondary'}
                        component={RouterLink}
                        to={'/chatlogs'}
                        search={{ steam_id: ban.target_id }}
                    >
                        Chat Logs
                    </Button>
                    <ButtonGroup fullWidth variant={'contained'}>
                        <Button color={'warning'} onClick={onEditBan} startIcon={<EditIcon />}>
                            Edit Ban
                        </Button>
                        <Button color={'success'} onClick={onUnban} disabled={expired} startIcon={<UndoIcon />}>
                            Unban
                        </Button>
                    </ButtonGroup>
                </Stack>
            </ContainerWithHeader>
        );
    }, [ban, expired, navigate, onEditBan, onSaveAppealState, onUnban]);

    const mutation = useMutation({
        mutationKey: ['banSteam'],
        mutationFn: async (values: { body_md: string }) => {
            if (!ban) {
                return;
            }
            const msg = await apiCreateBanMessage(ban?.ban_id, values.body_md);

            queryClient.setQueryData(['banMessages', { ban_id: ban.ban_id }], [...(messages ?? []), msg]);
            sendFlash('success', 'Created message successfully');
            mdEditorRef.current?.setMarkdown('');
            reset();
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                body_md: value.body_md
            });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            body_md: ''
        }
    });

    return (
        <Grid container spacing={2}>
            <Grid xs={8}>
                <Stack spacing={2}>
                    {canPost && (messages ?? []).length == 0 && (
                        <ContainerWithHeader title={`Ban Appeal #${ban.ban_id}`}>
                            <Typography variant={'body2'} padding={2} textAlign={'center'}>
                                You can start the appeal process by replying on this form.
                            </Typography>
                        </ContainerWithHeader>
                    )}

                    {permissionLevel() >= PermissionLevel.Moderator && (
                        <SourceBansList steam_id={ban?.source_id} is_reporter={true} />
                    )}

                    {permissionLevel() >= PermissionLevel.Moderator && (
                        <SourceBansList steam_id={ban?.target_id} is_reporter={false} />
                    )}

                    {(messages ?? []).map((m) => (
                        <AppealMessageView onDelete={onDelete} message={m} key={`ban-appeal-msg-${m.ban_message_id}`} />
                    ))}
                    {canPost && (
                        <Paper elevation={1}>
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
                                            name={'body_md'}
                                            validators={{
                                                onChange: z.string().min(2)
                                            }}
                                            children={(props) => {
                                                return <MarkdownField {...props} label={'Message'} />;
                                            }}
                                        />
                                    </Grid>
                                    <Grid xs={12} mdOffset="auto">
                                        <Subscribe
                                            selector={(state) => [state.canSubmit, state.isSubmitting]}
                                            children={([canSubmit, isSubmitting]) => {
                                                return (
                                                    <Buttons
                                                        reset={reset}
                                                        canSubmit={canSubmit}
                                                        isSubmitting={isSubmitting}
                                                    />
                                                );
                                            }}
                                        />
                                    </Grid>
                                </Grid>
                            </form>
                        </Paper>
                    )}
                    {!canPost && (
                        <Paper elevation={1}>
                            <Typography variant={'body2'} padding={2} textAlign={'center'}>
                                The ban appeal is closed: {appealStateString(ban.appeal_state)}
                            </Typography>
                        </Paper>
                    )}
                </Stack>
            </Grid>
            <Grid xs={4}>
                <Stack spacing={2}>
                    <ProfileInfoBox steam_id={ban.target_id} />

                    <ContainerWithHeader title={'Ban Details'} iconLeft={<InfoIcon />}>
                        <List dense={true}>
                            <ListItem>
                                <ListItemText primary={'Reason'} secondary={BanReasons[ban.reason]} />
                            </ListItem>
                            <ListItem>
                                <ListItemText primary={'Ban Type'} secondary={banTypeString(ban.ban_type)} />
                            </ListItem>
                            {ban.reason_text != '' && (
                                <ListItem>
                                    <ListItemText primary={'Reason (Custom)'} secondary={ban.reason_text} />
                                </ListItem>
                            )}

                            <ListItem>
                                <ListItemText primary={'Created At'} secondary={renderDateTime(ban.created_on)} />
                            </ListItem>
                            <ListItem>
                                <ListItemText primary={'Expires At'} secondary={renderDateTime(ban.valid_until)} />
                            </ListItem>
                            <ListItem>
                                <ListItemText primary={'Expires'} secondary={renderTimeDistance(ban.valid_until)} />
                            </ListItem>
                            {permissionLevel() >= PermissionLevel.Moderator && (
                                <ListItem>
                                    <ListItemText primary={'Author'} secondary={ban.source_id.toString()} />
                                </ListItem>
                            )}
                        </List>
                    </ContainerWithHeader>

                    <SteamIDList steam_id={ban?.target_id} />

                    {permissionLevel() >= PermissionLevel.Moderator && ban.note != '' && (
                        <ContainerWithHeader title={'Mod Notes'} iconLeft={<DocumentScannerIcon />}>
                            <Typography variant={'body2'} padding={2}>
                                <MarkDownRenderer body_md={ban.note} />
                            </Typography>
                        </ContainerWithHeader>
                    )}

                    {permissionLevel() >= PermissionLevel.Moderator && modTools}
                </Stack>
            </Grid>
        </Grid>
    );
}
