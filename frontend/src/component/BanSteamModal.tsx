import React, { useMemo } from 'react';
import Stack from '@mui/material/Stack';
import {
    apiCreateBanSteam,
    apiGetProfile,
    BanReason,
    BanReasons,
    banReasonsList,
    BanType,
    Duration,
    Durations,
    IAPIBanRecord
} from '../api';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { Heading } from './Heading';
import SteamID from 'steamid';

import * as yup from 'yup';
import { useFormik } from 'formik';
import Button from '@mui/material/Button';
import GavelIcon from '@mui/icons-material/Gavel';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import { SteamIDInput, SteamIDInputValue } from './formik/SteamIDInput';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export interface BanModalProps<Ban> {
    open: boolean;
    setOpen: (open: boolean) => void;
    ban?: Ban;
    reportId?: number;
    steamId?: SteamID;
}

interface BanSteamFormValues extends SteamIDInputValue {
    reportId?: number;
    banType: BanType;
    reason: BanReason;
    reasonText: string;
    duration: Duration;
    durationCustom: string;
    note: string;
}

const validationSchema = yup.object({
    steam_id: yup
        .string()
        .test('checkSteamId', 'Invalid steamid/vanity', async (steamId) => {
            if (!steamId) {
                return false;
            }
            try {
                const resp = await apiGetProfile(steamId);
                return !(!resp.status || !resp.result);
            } catch (e) {
                logErr(e);
                return false;
            }
        })
        .label('Enter your steam_id')
        .required('steam_id is required'),
    reportId: yup.number().min(0, 'Must be positive integer').nullable(),
    banType: yup
        .number()
        .label('Select a ban type')
        .required('ban type is required'),
    reason: yup
        .number()
        .label('Select a reason')
        .required('reason is required'),
    reasonText: yup.string().label('Custom reason'),
    duration: yup
        .string()
        .label('Ban/Mute duration')
        .required('Duration is required'),
    durationCustom: yup.string().label('Custom duration'),
    note: yup.string().label('Hidden Moderator Note')
});

export const BanSteamModal = ({
    open,
    setOpen,
    steamId,
    reportId
}: BanModalProps<IAPIBanRecord>) => {
    const { sendFlash } = useUserFlashCtx();

    const isReadOnlySid = useMemo(() => {
        return !!steamId?.getSteamID64();
    }, [steamId]);

    const formik = useFormik<BanSteamFormValues>({
        initialValues: {
            banType: BanType.NoComm,
            duration: Duration.dur2w,
            durationCustom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: steamId?.getSteamID64() ?? '',
            reasonText: '',
            reportId: reportId
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            try {
                const resp = await apiCreateBanSteam({
                    note: values.note,
                    ban_type: values.banType,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reasonText,
                    report_id: values.reportId,
                    target_id: values.steam_id
                });
                if (!resp.status || !resp.result) {
                    sendFlash('error', 'Error saving ban');
                    return;
                }
                sendFlash('success', 'Ban created successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error saving ban');
            } finally {
                setOpen(false);
            }
        }
    });

    return (
        <form onSubmit={formik.handleSubmit} id={'banForm'}>
            <Dialog
                fullWidth
                open={open}
                onClose={() => {
                    setOpen(false);
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Ban Steam Profile
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <SteamIDInput
                            formik={formik}
                            fullWidth
                            isReadOnly={isReadOnlySid}
                        />

                        <Stack spacing={3} alignItems={'center'}>
                            <TextField
                                fullWidth
                                id={'report_id'}
                                label={'report_id'}
                                name={'report_id'}
                                disabled={true}
                                hidden={true}
                                value={formik.values.reportId}
                                onChange={formik.handleChange}
                            />
                            <FormControl fullWidth>
                                <InputLabel id="actionType-label">
                                    Action Type
                                </InputLabel>
                                <Select<BanType>
                                    fullWidth
                                    label={'Action Type'}
                                    labelId="actionType-label"
                                    id="banType"
                                    name={'banType'}
                                    value={formik.values.banType}
                                    onChange={formik.handleChange}
                                    error={
                                        formik.touched.banType &&
                                        Boolean(formik.errors.banType)
                                    }
                                    defaultValue={BanType.Banned}
                                >
                                    {[BanType.Banned, BanType.NoComm].map(
                                        (v) => (
                                            <MenuItem
                                                key={`time-${v}`}
                                                value={v}
                                            >
                                                {v == BanType.NoComm
                                                    ? 'Mute'
                                                    : 'Ban'}
                                            </MenuItem>
                                        )
                                    )}
                                </Select>
                                <FormHelperText>
                                    {formik.touched.banType &&
                                        formik.errors.banType}
                                </FormHelperText>
                            </FormControl>

                            <FormControl fullWidth>
                                <InputLabel id="reason-label">
                                    Reason
                                </InputLabel>
                                <Select<BanReason>
                                    labelId="reason-label"
                                    id="reason"
                                    name={'reason'}
                                    value={formik.values.reason}
                                    onChange={formik.handleChange}
                                    error={
                                        formik.touched.reason &&
                                        Boolean(formik.errors.reason)
                                    }
                                >
                                    {banReasonsList.map((v) => (
                                        <MenuItem key={`time-${v}`} value={v}>
                                            {BanReasons[v]}
                                        </MenuItem>
                                    ))}
                                </Select>
                                <FormHelperText>
                                    {formik.touched.reason &&
                                        formik.errors.reason}
                                </FormHelperText>
                            </FormControl>

                            <TextField
                                fullWidth
                                id={'reasonText'}
                                label={'Custom Reason'}
                                name={'reasonText'}
                                disabled={
                                    formik.values.reason != BanReason.Custom
                                }
                                value={formik.values.reasonText}
                                onChange={formik.handleChange}
                                error={
                                    formik.touched.reasonText &&
                                    Boolean(formik.errors.reasonText)
                                }
                                helperText={
                                    formik.touched.reasonText &&
                                    formik.errors.reasonText
                                }
                            />

                            <FormControl fullWidth>
                                <InputLabel id="duration-label">
                                    Duration
                                </InputLabel>
                                <Select<Duration>
                                    fullWidth
                                    label={'Ban Duration'}
                                    labelId="duration-label"
                                    id="duration"
                                    name={'duration'}
                                    value={formik.values.duration}
                                    onChange={formik.handleChange}
                                    error={
                                        formik.touched.duration &&
                                        Boolean(formik.errors.duration)
                                    }
                                >
                                    {Durations.map((v) => (
                                        <MenuItem key={`time-${v}`} value={v}>
                                            {v}
                                        </MenuItem>
                                    ))}
                                </Select>
                                <FormHelperText>
                                    {formik.touched.duration &&
                                        formik.errors.duration}
                                </FormHelperText>
                            </FormControl>

                            <TextField
                                fullWidth
                                label={'Custom Duration'}
                                id={'durationCustom'}
                                name={'durationCustom'}
                                disabled={
                                    formik.values.duration != Duration.durCustom
                                }
                                value={formik.values.durationCustom}
                                onChange={formik.handleChange}
                                error={
                                    formik.touched.durationCustom &&
                                    Boolean(formik.errors.durationCustom)
                                }
                                helperText={
                                    formik.touched.durationCustom &&
                                    formik.errors.durationCustom
                                }
                            />

                            <TextField
                                fullWidth
                                id="note"
                                name={'note'}
                                label="Moderator Notes (hidden from public)"
                                multiline
                                value={formik.values.note}
                                onChange={formik.handleChange}
                                error={
                                    formik.touched.note &&
                                    Boolean(formik.errors.note)
                                }
                                helperText={
                                    formik.touched.note && formik.errors.note
                                }
                                rows={10}
                                variant="outlined"
                            />
                        </Stack>
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <Button
                        variant={'contained'}
                        color={'success'}
                        startIcon={<CheckIcon />}
                        type={'submit'}
                        form={'banForm'}
                    >
                        Accept
                    </Button>

                    <Button
                        variant={'contained'}
                        color={'warning'}
                        startIcon={<ClearIcon />}
                        type={'reset'}
                        form={'banForm'}
                    >
                        Reset
                    </Button>
                    <Button
                        variant={'contained'}
                        color={'error'}
                        startIcon={<ClearIcon />}
                        type={'button'}
                        form={'banForm'}
                        onClick={() => {
                            setOpen(false);
                        }}
                    >
                        Cancel
                    </Button>
                </DialogActions>
            </Dialog>
        </form>
    );
};
