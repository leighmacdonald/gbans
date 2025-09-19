import { useCallback, useMemo } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddModeratorIcon from '@mui/icons-material/AddModerator';
import ChatIcon from '@mui/icons-material/Chat';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import z from 'zod/v4';
import { apiGetBanSteam, apiSetBanAppealState, appealStateString } from '../api';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { AppealState, AppealStateCollection, AppealStateEnum } from '../schema/bans.ts';
import { logErr } from '../util/errors.ts';
import { ButtonLink } from './ButtonLink.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import { ErrorDetails } from './ErrorDetails.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder.tsx';
import { Title } from './Title';
import { ModalBan, ModalUnban } from './modal';

const onSubmit = z.object({
    appeal_state: AppealStateEnum
});

export const BanModPanel = ({ ban_id }: { ban_id: number }) => {
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    const {
        data: ban,
        isLoading,
        isError,
        error
    } = useQuery({
        queryKey: ['ban', { ban_id }],
        queryFn: async () => {
            return await apiGetBanSteam(Number(ban_id), true);
        }
    });

    const enabled = useMemo(() => {
        if (!ban?.valid_until) {
            return false;
        }

        return ban.valid_until ? ban.valid_until < new Date() : false;
    }, [ban?.valid_until]);

    const onUnban = useCallback(async () => {
        await NiceModal.show(ModalUnban, {
            banId: ban_id,
            personaName: ban?.target_personaname
        });
    }, [ban_id, ban?.target_personaname]);

    const onEditBan = useCallback(async () => {
        await NiceModal.show(ModalBan, {
            ban_id: ban_id
        });
    }, [ban_id, ban?.target_personaname]);

    const appealStateMutation = useMutation({
        mutationKey: ['banEdit', { ban_id }],
        mutationFn: async (appeal_state: AppealStateEnum) => {
            try {
                await apiSetBanAppealState(ban_id, appeal_state);
                sendFlash('success', 'Appeal state updated');
            } catch (reason) {
                sendFlash('error', 'Could not set appeal state');
                logErr(reason);
            }
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            if (value.appeal_state == ban?.appeal_state) {
                return;
            }
            appealStateMutation.mutate(value.appeal_state);
        },
        validators: { onSubmit },
        defaultValues: { appeal_state: ban?.appeal_state ?? AppealState.Any }
    });

    if (isLoading) {
        return <LoadingPlaceholder />;
    }

    if (isError) {
        return <ErrorDetails error={error} />;
    }

    return (
        <ContainerWithHeader title={'Moderation Tools'} iconLeft={<AddModeratorIcon />}>
            <Title>Ban Appeal</Title>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <Stack spacing={2} padding={2}>
                    <Stack direction={'row'} spacing={2}>
                        {!enabled ? (
                            <>
                                <form.AppField
                                    name={'appeal_state'}
                                    children={(field) => {
                                        return (
                                            <field.SelectField
                                                label={'Appeal State'}
                                                value={field.state.value}
                                                items={AppealStateCollection}
                                                renderItem={(i) => {
                                                    return (
                                                        <MenuItem value={i} key={i}>
                                                            {appealStateString(i)}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                                <form.AppForm>
                                    <form.SubmitButton label={'Save'} />
                                </form.AppForm>
                            </>
                        ) : (
                            <Typography variant={'h6'} textAlign={'center'}>
                                Ban Expired
                            </Typography>
                        )}
                    </Stack>

                    {Boolean(ban?.report_id) && (
                        <Button
                            fullWidth
                            disabled={!enabled}
                            color={'secondary'}
                            variant={'contained'}
                            onClick={async () => {
                                await navigate({ to: `/report/${ban?.report_id}` });
                            }}
                        >
                            View Report #{ban?.report_id}
                        </Button>
                    )}
                    <ButtonLink
                        variant={'contained'}
                        color={'secondary'}
                        to={'/chatlogs'}
                        search={{ steam_id: ban?.target_id }}
                        startIcon={<ChatIcon />}
                    >
                        Chat Logs
                    </ButtonLink>
                    <ButtonGroup fullWidth variant={'contained'}>
                        <Button color={'warning'} onClick={onEditBan} startIcon={<EditIcon />}>
                            Edit Ban
                        </Button>
                        <Button color={'success'} onClick={onUnban} startIcon={<UndoIcon />}>
                            Unban
                        </Button>
                    </ButtonGroup>
                </Stack>
            </form>
        </ContainerWithHeader>
    );
};
