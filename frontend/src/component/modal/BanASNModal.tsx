import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import LanIcon from '@mui/icons-material/Lan';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
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
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { emptyOrNullString } from '../../util/types.ts';
import { makeValidateSteamIDCallback } from '../../util/validator/makeValidateSteamIDCallback.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { DateTimeSimple } from '../field/DateTimeSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { SteamIDField } from '../field/SteamIDField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

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
    const { sendFlash, sendError } = useUserFlashCtx();
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
                    valid_until: !emptyOrNullString(values.duration_custom)
                        ? parseISO(values.duration_custom)
                        : undefined,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.target_id,
                    as_num: Number(values.as_num)
                });
                sendFlash('success', 'Created ASN ban successfully');
                modal.resolve(ban_record);
            }
        },
        onSuccess: async (banRecord) => {
            if (existing?.ban_asn_id) {
                sendFlash('success', 'Updated asn ban successfully');
            } else {
                sendFlash('success', 'Created asn ban successfully');
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
                reason: value.reason,
                reason_text: value.reason_text,
                duration: value.duration,
                duration_custom: value.duration_custom,
                note: value.note,
                as_num: value.as_num
            });
        },
        validators: {
            onChangeAsyncDebounceMs: 500,
            onChangeAsync: z.object({
                target_id: makeValidateSteamIDCallback(),
                reason: z.nativeEnum(BanReason),
                reason_text: z.string(),
                duration: z.nativeEnum(Duration),
                duration_custom: z.string(),
                note: z.string(),
                as_num: z.string()
            })
        },
        defaultValues: {
            target_id: existing ? existing.target_id : '',
            reason: existing ? existing.reason : BanReason.Cheating,
            reason_text: existing ? existing.reason_text : '',
            duration: existing ? Duration.durCustom : Duration.dur2w,
            duration_custom: existing ? existing.valid_until.toISOString() : '',
            note: existing ? existing.note : '',
            as_num: existing ? String(existing.as_num) : '0'
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
                <DialogTitle component={Heading} iconLeft={<LanIcon />}>
                    Ban Autonomous System Number Range
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
                                            disabled={Boolean(existing?.ban_asn_id)}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'as_num'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} value={props.state.value} label={'AS Number'} />;
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
                                            defaultValue={props.state.value}
                                            label={'Reason'}
                                            fullwidth={true}
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
                                name={'duration'}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
                                            defaultValue={props.state.value}
                                            label={'Duration'}
                                            fullwidth={true}
                                            items={DurationCollection}
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
});
