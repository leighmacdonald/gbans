import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation } from '@tanstack/react-query';
import { parseISO } from 'date-fns';
import { z } from 'zod';
import {
    apiCreateBanSteam,
    apiUpdateBanSteam,
    BanReason,
    BanReasons,
    banReasonsCollection,
    BanType,
    BanTypeCollection,
    banTypeString,
    Duration,
    DurationCollection,
    SteamBanRecord
} from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Heading } from '../Heading';
import { DateTimeField } from '../field/DateTimeField.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';

type BanSteamFormValues = {
    report_id?: number;
    target_id: string;
    ban_type: BanType;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom?: string;
    note: string;
    include_friends: boolean;
    evade_ok: boolean;
};

export const BanSteamModal = NiceModal.create(
    ({ existing, steamId = '' }: { existing?: SteamBanRecord; steamId?: string }) => {
        const { sendFlash, sendError } = useUserFlashCtx();
        const modal = useModal();

        const mutation = useMutation({
            mutationKey: ['banSteam'],
            mutationFn: async (values: BanSteamFormValues) => {
                if (existing?.ban_id) {
                    return await apiUpdateBanSteam(existing.ban_id, {
                        note: values.note,
                        ban_type: values.ban_type,
                        reason: values.reason,
                        reason_text: values.reason_text,
                        include_friends: values.include_friends,
                        evade_ok: values.evade_ok,
                        valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined
                    });
                } else {
                    return await apiCreateBanSteam({
                        note: values.note,
                        ban_type: values.ban_type,
                        duration: values.duration,
                        valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined,
                        reason: values.reason,
                        reason_text: values.reason_text,
                        report_id: values.report_id,
                        target_id: values.target_id,
                        include_friends: values.include_friends,
                        evade_ok: values.evade_ok
                    });
                }
            },
            onSuccess: async (banRecord) => {
                if (existing?.ban_id) {
                    sendFlash('success', 'Updated ban successfully');
                } else {
                    sendFlash('success', 'Created ban successfully');
                }
                modal.resolve(banRecord);
                await modal.hide();
            },
            onError: sendError
        });

        const form = useAppForm({
            onSubmit: async ({ value }) => {
                mutation.mutate({
                    target_id: value.target_id,
                    ban_type: value.ban_type,
                    reason: value.reason,
                    reason_text: value.reason_text,
                    duration: value.duration,
                    duration_custom: value.duration_custom,
                    evade_ok: value.evade_ok,
                    include_friends: value.include_friends,
                    note: value.note,
                    report_id: value.report_id
                });
            },
            defaultValues: {
                report_id: existing ? existing.report_id : 0,
                target_id: existing ? existing.target_id : steamId,
                ban_type: existing ? existing.ban_type : BanType.Banned,
                reason: existing ? existing.reason : BanReason.Cheating,
                reason_text: existing ? existing.reason_text : '',
                duration: existing ? Duration.durCustom : Duration.dur2w,
                duration_custom: existing ? existing.valid_until.toISOString() : '',
                note: existing ? existing.note : '',
                include_friends: existing ? existing.include_friends : false,
                evade_ok: existing ? existing.evade_ok : false
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
                    <DialogTitle component={Heading} iconLeft={<DirectionsRunIcon />}>
                        Ban Steam Profile
                    </DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'target_id'}
                                    //validators={makeSteamidValidators()}
                                    children={(field) => {
                                        return (
                                            <field.SteamIDField
                                                label={'Target Steam ID'}
                                                disabled={Boolean(existing?.ban_id)}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'ban_type'}
                                    validators={{
                                        onChange: z.nativeEnum(BanType)
                                    }}
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
                                    name={'include_friends'}
                                    validators={{
                                        onChange: z.boolean()
                                    }}
                                    children={(field) => {
                                        return <field.CheckboxField label={'Include Friends'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6 }}>
                                <form.AppField
                                    name={'evade_ok'}
                                    validators={{
                                        onChange: z.boolean()
                                    }}
                                    children={(field) => {
                                        return <field.CheckboxField label={'IP Evading Allowed'} />;
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
                                    children={(props) => {
                                        return <DateTimeField {...props} label={'Custom Expire Date'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'note'}
                                    validators={{
                                        onChange: z.string()
                                    }}
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
                                    <form.ResetButton />
                                    <form.SubmitButton />
                                </form.AppForm>
                            </Grid>
                        </Grid>
                    </DialogActions>
                </form>
            </Dialog>
        );
    }
);
