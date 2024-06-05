import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import RouterIcon from '@mui/icons-material/Router';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import MenuItem from '@mui/material/MenuItem';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { parseISO } from 'date-fns';
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
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { makeSteamidValidators } from '../../util/validator/makeSteamidValidators.ts';
import { Heading } from '../Heading';
import { Buttons } from '../field/Buttons.tsx';
import { DateTimeSimple } from '../field/DateTimeSimple.tsx';
import { MarkdownField } from '../field/MarkdownField.tsx';
import { SelectFieldSimple } from '../field/SelectFieldSimple.tsx';
import { SteamIDField } from '../field/SteamIDField.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type BanCIDRFormValues = {
    target_id: string;
    cidr: string;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom?: string;
    note: string;
    existing?: CIDRBanRecord;
};

export const BanCIDRModal = NiceModal.create(({ existing }: { existing?: CIDRBanRecord }) => {
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();

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
                    valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined
                });
                sendFlash('success', 'Updated CIDR ban successfully');
                modal.resolve(ban_record);
            } else {
                const ban_record = await apiCreateBanCIDR({
                    note: values.note,
                    duration: values.duration,
                    valid_until: values.duration_custom ? parseISO(values.duration_custom) : undefined,
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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
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
        validatorAdapter: zodValidator,
        defaultValues: {
            target_id: existing ? existing.target_id : '',
            reason: existing ? existing.reason : BanReason.Cheating,
            reason_text: existing ? existing.reason_text : '',
            duration: existing ? Duration.durCustom : Duration.dur2w,
            duration_custom: existing ? existing.valid_until.toISOString() : '',
            note: existing ? existing.note : '',
            cidr: existing ? existing.cidr : ''
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
                <DialogTitle component={Heading} iconLeft={<RouterIcon />}>
                    Ban CIDR Range
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
                                            disabled={Boolean(existing?.net_id)}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                        <Grid xs={12}>
                            <Field
                                name={'cidr'}
                                validators={{
                                    onChange: z.string()
                                }}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'IP/CIDR Range'} />;
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
                                    return <MarkdownField {...props} multiline={true} rows={10} label={'Mod Notes'} />;
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
        // </Formik>
    );
});
