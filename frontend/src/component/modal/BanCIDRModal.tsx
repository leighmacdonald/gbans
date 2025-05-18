import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import RouterIcon from '@mui/icons-material/Router';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import {
    apiCreateBanCIDR,
    apiUpdateBanCIDR,
    BanReason,
    BanReasons,
    banReasonsCollection,
    CIDRBanRecord,
    Duration,
    DurationCollection
} from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';

const schema = z.object({
    target_id: z.string(),
    reason: z.nativeEnum(BanReason),
    reason_text: z.string(),
    duration: z.nativeEnum(Duration),
    duration_custom: z.date(),
    note: z.string(),
    cidr: z.string().cidr({ version: 'v4' })
});

type BanCIDRFormValues = z.infer<typeof schema> & { existing?: CIDRBanRecord };

export const BanCIDRModal = NiceModal.create(({ existing }: { existing?: CIDRBanRecord }) => {
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();
    const defaultValues: z.input<typeof schema> = {
        target_id: existing ? existing.target_id : '',
        reason: existing ? existing.reason : BanReason.Cheating,
        reason_text: existing ? existing.reason_text : '',
        duration: existing ? Duration.durCustom : Duration.dur2w,
        duration_custom: existing?.valid_until ?? new Date(),
        note: existing?.note ?? '',
        cidr: existing?.cidr ?? ''
    };
    const mutation = useMutation({
        mutationKey: ['banCIDR'],
        mutationFn: async (values: BanCIDRFormValues) => {
            if (existing?.net_id) {
                const ban_record = apiUpdateBanCIDR(existing.net_id, {
                    note: values.note,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    cidr: values.cidr,
                    target_id: values.target_id,
                    valid_until: values.duration_custom
                });
                sendFlash('success', 'Updated CIDR ban successfully');
                modal.resolve(ban_record);
            } else {
                const ban_record = await apiCreateBanCIDR({
                    note: values.note,
                    duration: values.duration,
                    valid_until: values.duration_custom,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.target_id,
                    cidr: values.cidr
                });
                sendFlash('success', 'Created CIDR ban successfully');
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
                cidr: value.cidr
            });
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
                <DialogTitle component={Heading} iconLeft={<RouterIcon />}>
                    Ban CIDR Range
                </DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'target_id'}
                                children={(field) => {
                                    return (
                                        <field.SteamIDField
                                            label={'Target Steam ID'}
                                            disabled={Boolean(existing?.net_id)}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'cidr'}
                                children={(field) => {
                                    return <field.TextField label={'IP/CIDR Range'} />;
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
                                    return <field.MarkdownField multiline={true} rows={10} label={'Mod Notes'} />;
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
        // </Formik>
    );
});
