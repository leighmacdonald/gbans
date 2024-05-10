import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import LanIcon from '@mui/icons-material/Lan';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
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
import { makeSteamidValidators } from '../../util/validator/makeSteamidValidators.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { DateTimeSimple } from '../field/DateTimeSimple.tsx';
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
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();

    const mutation = useMutation({
        mutationKey: ['banASN'],
        mutationFn: async (values: BanASNFormValues) => {
            console.log(values);
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
        validatorAdapter: zodValidator,
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
                                validators={makeSteamidValidators()}
                                children={(props) => {
                                    return (
                                        <SteamIDField
                                            {...props}
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
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'AS Number'} />;
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
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Custom Ban Reason'} />;
                                }}
                            />
                        </Grid>
                        <Grid xs={6}>
                            <Field
                                name={'duration'}
                                validators={{
                                    onChange: z.nativeEnum(Duration)
                                }}
                                children={(props) => {
                                    return (
                                        <SelectFieldSimple
                                            {...props}
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
                                    return <DateTimeSimple {...props} label={'Custom Expire Date'} />;
                                }}
                            />
                        </Grid>

                        <Grid xs={12}>
                            <Field
                                name={'note'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(props) => {
                                    return (
                                        <TextFieldSimple {...props} multiline={true} rows={10} label={'Mod Notes'} />
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

export default BanASNModal;
