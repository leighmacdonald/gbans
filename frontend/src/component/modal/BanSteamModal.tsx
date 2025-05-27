import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiCreateBanSteam, apiGetBanSteam, apiUpdateBanSteam, banTypeString } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import {
    BanPayloadSteam,
    BanReason,
    BanReasons,
    banReasonsCollection,
    BanType,
    BanTypeCollection,
    Duration,
    DurationCollection,
    schemaBanPayloadSteam
} from '../../schema/bans.ts';
import { ErrorDetails } from '../ErrorDetails.tsx';
import { Heading } from '../Heading';
import { LoadingPlaceholder } from '../LoadingPlaceholder.tsx';
import { MarkdownField } from '../form/field/MarkdownField.tsx';

export const BanSteamModal = NiceModal.create(({ ban_id }: { ban_id?: number }) => {
    const queryClient = useQueryClient();
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

    const { sendFlash, sendError } = useUserFlashCtx();
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['banSteam'],
        mutationFn: async (values: BanPayloadSteam) => {
            let updated: BanPayloadSteam;
            if (ban?.ban_id) {
                updated = (await apiUpdateBanSteam(ban.ban_id, {
                    note: values.note,
                    ban_type: values.ban_type,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    include_friends: values.include_friends,
                    evade_ok: values.evade_ok,
                    valid_until: values.duration_custom
                })) as unknown as BanPayloadSteam; // TODO fix
            } else {
                updated = (await apiCreateBanSteam({
                    note: values.note,
                    ban_type: values.ban_type,
                    duration: values.duration,
                    valid_until: values.duration_custom,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    report_id: values.report_id,
                    target_id: values.target_id,
                    include_friends: values.include_friends,
                    evade_ok: values.evade_ok
                })) as unknown as BanPayloadSteam;
            }
            queryClient.setQueryData(['ban', { ban_id }], updated);
        },
        onSuccess: async (banRecord) => {
            if (ban?.ban_id) {
                sendFlash('success', 'Updated ban successfully');
            } else {
                sendFlash('success', 'Created ban successfully');
            }
            modal.resolve(banRecord);
            await modal.hide();
        },
        onError: sendError
    });

    const defaultValues: z.infer<typeof schemaBanPayloadSteam> = {
        report_id: ban?.report_id ?? 0,
        target_id: ban?.target_id ?? '',
        ban_type: ban?.ban_type ?? BanType.Banned,
        reason: ban?.reason ?? BanReason.Cheating,
        reason_text: ban?.reason_text ?? '',
        duration: ban ? Duration.durCustom : Duration.dur2w,
        duration_custom: ban?.valid_until ?? new Date(),
        note: ban?.note ?? '',
        include_friends: ban?.include_friends ?? false,
        evade_ok: ban?.evade_ok ?? false
    };

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues,
        validators: {
            onSubmit: schemaBanPayloadSteam
        }
    });

    if (isLoading) {
        return <LoadingPlaceholder />;
    }

    if (isError) {
        return <ErrorDetails error={error} />;
    }

    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<DirectionsRunIcon />}>
                    Ban Steam Profile
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'target_id'}
                                children={(field) => {
                                    return (
                                        <field.SteamIDField label={'Target Steam ID'} disabled={Boolean(ban?.ban_id)} />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'ban_type'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Ban Action Type'}
                                            items={BanTypeCollection}
                                            renderItem={(bt) => {
                                                return (
                                                    <MenuItem value={bt} key={`bt-${bt}`}>
                                                        {banTypeString(bt)}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'reason'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Reason'}
                                            items={banReasonsCollection}
                                            renderItem={(br) => {
                                                return (
                                                    <MenuItem value={br} key={`br-${br}`}>
                                                        {BanReasons[br]}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'reason_text'}
                                validators={{
                                    onSubmit: ({ value, fieldApi }) => {
                                        if (fieldApi.form.getFieldValue('reason') != BanReason.Custom) {
                                            if (value.length == 0) {
                                                return undefined;
                                            }
                                            return 'Must use custom ban reason';
                                        }
                                        const result = z.string().min(5).safeParse(value);
                                        if (!result.success) {
                                            return result.error.message;
                                        }

                                        return undefined;
                                    }
                                }}
                                children={(field) => {
                                    return <field.TextField label={'Custom Ban Reason'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'include_friends'}
                                children={(field) => {
                                    return <field.CheckboxField label={'Include Friends'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'evade_ok'}
                                children={(field) => {
                                    return <field.CheckboxField label={'IP Evading Allowed'} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'duration'}
                                children={(field) => {
                                    return (
                                        <field.SelectField
                                            label={'Duration'}
                                            items={DurationCollection}
                                            renderItem={(du) => {
                                                return (
                                                    <MenuItem value={du} key={`du-${du}`}>
                                                        {du}
                                                    </MenuItem>
                                                );
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 6 }}>
                            <form.AppField
                                name={'duration_custom'}
                                children={(field) => {
                                    return <field.DateTimeField label={'Custom Expire Date'} />;
                                }}
                            />
                        </Grid>

                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'note'}
                                children={(props) => {
                                    return (
                                        <MarkdownField
                                            {...props}
                                            value={props.state.value}
                                            multiline={true}
                                            rows={10}
                                            label={'Mod Notes'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>
                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <ButtonGroup>
                                    <form.ResetButton />
                                    <form.SubmitButton />
                                </ButtonGroup>
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
