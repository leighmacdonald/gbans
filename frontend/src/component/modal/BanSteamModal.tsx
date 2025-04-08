import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
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
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { makeValidateSteamIDCallback } from '../../util/validator/makeValidateSteamIDCallback.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { CheckboxSimple } from '../field/CheckboxSimple.tsx';
import { DateTimeSimple } from '../field/DateTimeSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { SteamIDField } from '../field/SteamIDField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

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

        const { Field, Subscribe, handleSubmit, reset } = useForm({
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
            validators: {
                onChangeAsyncDebounceMs: 500,
                onChangeAsync: z.object({
                    report_id: z.number(),
                    target_id: makeValidateSteamIDCallback(),
                    ban_type: z.nativeEnum(BanType),
                    reason: z.nativeEnum(BanReason),
                    reason_text: z.string(),
                    duration: z.nativeEnum(Duration),
                    duration_custom: z.string(),
                    note: z.string(),
                    include_friends: z.boolean(),
                    evade_ok: z.boolean()
                })
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
                        await handleSubmit();
                    }}
                >
                    <DialogTitle component={Heading} iconLeft={<DirectionsRunIcon />}>
                        Ban Steam Profile
                    </DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid xs={12}>
                                <Field
                                    name={'target_id'}
                                    children={(props) => {
                                        return (
                                            <SteamIDField
                                                {...props}
                                                defaultValue={props.state.value}
                                                label={'Target Steam ID'}
                                                fullwidth={true}
                                                disabled={Boolean(existing?.ban_id)}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={12}>
                                <Field
                                    name={'ban_type'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Ban Action Type'}
                                                fullwidth={true}
                                                defaultValue={props.state.value}
                                                items={BanTypeCollection}
                                                renderMenu={(bt) => {
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
                            <Grid xs={12}>
                                <Field
                                    name={'reason'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Reason'}
                                                fullwidth={true}
                                                defaultValue={props.state.value}
                                                items={banReasonsCollection}
                                                renderMenu={(br) => {
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
                            <Grid xs={12}>
                                <Field
                                    name={'reason_text'}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                defaultValue={props.state.value}
                                                label={'Custom Ban Reason'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={6}>
                                <Field
                                    name={'include_friends'}
                                    children={({ state, handleBlur, handleChange }) => {
                                        return (
                                            <CheckboxSimple
                                                checked={state.value}
                                                onChange={(_, v) => handleChange(v)}
                                                onBlur={handleBlur}
                                                label={'Include Friends'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={6}>
                                <Field
                                    name={'evade_ok'}
                                    children={({ state, handleBlur, handleChange }) => {
                                        return (
                                            <CheckboxSimple
                                                checked={state.value}
                                                onChange={(_, v) => handleChange(v)}
                                                onBlur={handleBlur}
                                                label={'IP Evading Allowed'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={6}>
                                <Field
                                    name={'duration'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Duration'}
                                                fullwidth={true}
                                                items={DurationCollection}
                                                defaultValue={props.state.value}
                                                renderMenu={(du) => {
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

                            <Grid xs={6}>
                                <Field
                                    name={'duration_custom'}
                                    children={(props) => {
                                        return (
                                            <DateTimeSimple
                                                {...props}
                                                defaultValue={props.state.value}
                                                label={'Custom Expire Date'}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={12}>
                                <Field
                                    name={'note'}
                                    children={(props) => {
                                        return (
                                            <MarkdownField
                                                {...props}
                                                defaultValue={props.state.value}
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
                            <Grid xs={12} mdOffset="auto">
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => {
                                        return (
                                            <Buttons
                                                reset={reset}
                                                canSubmit={canSubmit}
                                                isSubmitting={isSubmitting}
                                                onClose={async () => {
                                                    await modal.hide();
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                        </Grid>
                    </DialogActions>
                </form>
            </Dialog>
        );
    }
);
