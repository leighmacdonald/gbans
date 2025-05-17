import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import LanIcon from '@mui/icons-material/Lan';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { parseISO } from 'date-fns';
import { z } from 'zod';
import {
    apiCreateBanASN,
    apiUpdateBanASN,
    ASNBanRecord,
    BanReason,
    BanReasons,
    banReasonsCollection,
    Duration,
    DurationCollection
} from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

type BanASNFormValues = {
    target_id: string;
    ban_asn_id?: number;
    as_num: string;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom?: string;
    note: string;
};

export const BanASNModal = NiceModal.create(({ existing }: { existing?: ASNBanRecord }) => {
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['banASN'],
        mutationFn: async (values: BanASNFormValues) => {
            if (existing?.ban_asn_id) {
                const ban_record = apiUpdateBanASN(existing.ban_asn_id, {
                    note: values.note,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.target_id,
                    as_num: Number(values.as_num),
                    valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined
                });

                sendFlash('success', 'Updated ASN ban successfully');
                modal.resolve(ban_record);
            } else {
                const ban_record = await apiCreateBanASN({
                    note: values.note,
                    duration: values.duration,
                    valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.target_id,
                    as_num: Number(values.as_num)
                });
                sendFlash('success', 'Created ASN ban successfully');
                modal.resolve(ban_record);
            }
            await modal.hide();
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({
                target_id: value.target_id,
                reason: value.reason,
                reason_text: value.reason_text,
                duration: value.duration,
                duration_custom: value.duration_custom,
                note: value.note,
                as_num: value.as_num
            });
        },
        defaultValues: {
            target_id: existing ? existing.target_id : '',
            reason: existing ? existing.reason : BanReason.Cheating,
            reason_text: existing ? existing.reason_text : '',
            duration: existing ? Duration.durCustom : Duration.dur2w,
            duration_custom: existing ? existing.valid_until.toISOString() : '',
            note: existing ? existing.note : '',
            as_num: existing ? String(existing.as_num) : ''
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
                                // validators={makeSteamidValidators()}
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
                                validators={{
                                    onChange: z.string()
                                }}
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
                                            return result.error.errors.map((e) => e.message).join(',');
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
                                name={'duration'}
                                validators={{
                                    onChange: z.nativeEnum(Duration)
                                }}
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
                                validators={{
                                    onChange: z.string()
                                }}
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
