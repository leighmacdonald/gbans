import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { formatDuration, formatISO9075 } from 'date-fns';
import { intervalToDuration } from 'date-fns/intervalToDuration';
import { z } from 'zod/v4';
import { apiCreateBan, apiGetBanSteam, apiUpdateBanSteam, banTypeString } from '../../api';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useAuth } from '../../hooks/useAuth.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import {
    BanOpts,
    BanReason,
    BanReasons,
    banReasonsCollection,
    BanRecord,
    BanType,
    BanTypeCollection,
    Duration,
    DurationCollection,
    Origin,
    schemaBanPayload
} from '../../schema/bans.ts';
import { Duration8601ToString } from '../../util/time.ts';
import { ErrorDetails } from '../ErrorDetails.tsx';
import { Heading } from '../Heading';
import { LoadingPlaceholder } from '../LoadingPlaceholder.tsx';
import { MarkdownField } from '../form/field/MarkdownField.tsx';

export const BanModal = NiceModal.create(
    ({ banId, reportId, steamId }: { banId?: number; reportId?: number; steamId?: string }) => {
        const { profile } = useAuth();
        const queryClient = useQueryClient();
        const {
            data: ban,
            isLoading,
            isError,
            error
        } = useQuery({
            queryKey: ['ban', { banId }],
            queryFn: async () => {
                if (banId && banId > 0) {
                    return await apiGetBanSteam(Number(banId), true);
                }

                return {} as BanRecord;
            }
        });

        const { sendFlash, sendError } = useUserFlashCtx();
        const modal = useModal();

        const mutation = useMutation({
            mutationKey: ['banSteam'],
            mutationFn: async (values: BanOpts) => {
                let banRecord: BanRecord;
                if (ban?.ban_id) {
                    banRecord = await apiUpdateBanSteam(ban.ban_id, {
                        note: values.note,
                        ban_type: values.ban_type,
                        reason: values.reason,
                        reason_text: values.reason_text,
                        evade_ok: values.evade_ok,
                        cidr: values.cidr,
                        duration: values.duration
                    });
                } else {
                    banRecord = await apiCreateBan({
                        source_id: profile.steam_id,
                        note: values.note,
                        ban_type: values.ban_type,
                        duration: values.duration,
                        reason: values.reason,
                        reason_text: values.reason_text,
                        report_id: values.report_id,
                        target_id: values.target_id,
                        evade_ok: values.evade_ok,
                        demo_name: '',
                        demo_tick: 0,
                        origin: Origin.Web,
                        cidr: values.cidr
                    });
                }
                queryClient.setQueryData(['ban', { banId }], banRecord);
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

        const defaultValues: z.infer<typeof schemaBanPayload> = {
            report_id: ban?.report_id ?? reportId ?? 0,
            target_id: ban?.target_id ?? steamId ?? '',
            ban_type: ban?.ban_type ?? BanType.Banned,
            reason: ban?.reason ?? BanReason.Cheating,
            reason_text: ban?.reason_text ?? '',
            duration: ban ? Duration.durCustom : Duration.dur2w,
            note: ban?.note ?? '',
            evade_ok: ban?.evade_ok ?? false,
            cidr: ban?.cidr ?? '',
            demo_name: '',
            demo_tick: 0,
            origin: Origin.Reported
        };

        const form = useAppForm({
            onSubmit: async ({ value }) => {
                mutation.mutate(value);
            },
            defaultValues,
            validators: {
                onSubmit: schemaBanPayload
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
                        {Number(banId) > 0 ? 'Edit Ban' : 'Create Ban'}
                    </DialogTitle>

                    <DialogContent>
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'target_id'}
                                    children={(field) => {
                                        return (
                                            <field.SteamIDField
                                                label={'Target Steam ID Or Group ID'}
                                                disabled={Boolean(ban?.ban_id)}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'cidr'}
                                    children={(field) => {
                                        return <field.TextField label={'IP/CIDR'} />;
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

                            <Grid size={{ xs: 12 }}>
                                {ban && (
                                    <>
                                        <p>
                                            Expires In:{' '}
                                            {formatDuration(
                                                intervalToDuration({ start: new Date(), end: ban?.valid_until })
                                            )}
                                        </p>
                                        <p>Expires On: {formatISO9075(ban?.valid_until)}</p>
                                    </>
                                )}
                                <form.AppField
                                    name={'duration'}
                                    children={(field) => {
                                        return (
                                            <field.SelectField
                                                label={'Duration'}
                                                items={DurationCollection}
                                                renderItem={(bt) => {
                                                    return (
                                                        <MenuItem value={bt} key={`bt-${bt}`}>
                                                            {Duration8601ToString(bt)}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            {/*<Grid size={{ xs: 6 }}>*/}
                            {/*    <form.AppField*/}
                            {/*        name={'duration'}*/}
                            {/*        children={(field) => {*/}
                            {/*            return <field.TextField label={'Duration'} />;*/}
                            {/*        }}*/}
                            {/*    />*/}
                            {/*</Grid>*/}

                            <Grid size={{ xs: 12 }}>
                                <form.AppField
                                    name={'evade_ok'}
                                    children={(field) => {
                                        return <field.CheckboxField label={'IP Evading Allowed'} />;
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
    }
);
