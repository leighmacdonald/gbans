import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import LanIcon from '@mui/icons-material/Lan';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod/v4';
import { apiCreateBanASN, apiUpdateBanASN } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import {
    ASNBanRecord,
    BanReason,
    BanReasonEnum,
    BanReasons,
    banReasonsCollection,
    Duration,
    DurationCollection,
    DurationEnum
} from '../../schema/bans.ts';
import { Heading } from '../Heading';

const schema = z.object({
    target_id: z.string(),
    reason: BanReasonEnum,
    reason_text: z.string(),
    duration: DurationEnum,
    duration_custom: z.date(),
    note: z.string(),
    as_num: z.number().positive()
});

export const BanASNModal = NiceModal.create(({ existing }: { existing?: ASNBanRecord }) => {
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();
    const defaultValues: z.input<typeof schema> = {
        target_id: existing?.target_id ?? '',
        reason: existing?.reason ?? BanReason.Cheating,
        reason_text: existing?.reason_text ?? '',
        duration: existing ? Duration.durCustom : Duration.dur2w,
        duration_custom: existing?.valid_until ?? new Date(),
        note: existing?.note ?? '',
        as_num: existing?.as_num ?? 0
    };
    const mutation = useMutation({
        mutationKey: ['banASN'],
        mutationFn: async (values: z.infer<typeof schema>) => {
            if (existing?.ban_asn_id) {
                const ban_record = apiUpdateBanASN(existing.ban_asn_id, {
                    note: values.note,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.target_id,
                    as_num: values.as_num,
                    valid_until: values.duration_custom
                });

                sendFlash('success', 'Updated ASN ban successfully');
                modal.resolve(ban_record);
            } else {
                const ban_record = await apiCreateBanASN({
                    note: values.note,
                    duration: values.duration,
                    valid_until: values.duration_custom,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.target_id,
                    as_num: values.as_num
                });
                sendFlash('success', 'Created ASN ban successfully');
                modal.resolve(ban_record);
            }
            await modal.hide();
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate(value);
        },
        defaultValues,
        validators: {
            onSubmit: schema
        }
    });

    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle component={Heading} iconLeft={<LanIcon />}>
                    Ban Autonomous System Number Range
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'target_id'}
                                children={(field) => {
                                    return (
                                        <field.SteamIDField
                                            disabled={Boolean(existing?.ban_asn_id)}
                                            label={'Target Steam ID'}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'as_num'}
                                children={(field) => {
                                    return <field.TextField label={'AS Number'} />;
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
                                children={(field) => {
                                    return <field.TextField label={'Custom Ban Reason'} />;
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
                                children={(field) => {
                                    return <field.MarkdownField label={'Mod Notes'} />;
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
