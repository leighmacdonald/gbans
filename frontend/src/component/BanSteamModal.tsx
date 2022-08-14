import React, { ChangeEvent, useCallback, useState } from 'react';
import { ProfileSelectionInput } from './ProfileSelectionInput';
import Stack from '@mui/material/Stack';
import {
    apiCreateBanSteam,
    IAPIBanRecord,
    BanPayloadSteam,
    BanReason,
    BanReasons,
    banReasonsList,
    BanType,
    Duration,
    Durations
} from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import TextField from '@mui/material/TextField';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';
import SteamID from 'steamid';

export interface BanModalProps<Ban> extends ConfirmationModalProps<Ban> {
    ban?: Ban;
    reportId?: number;
    steamId?: SteamID;
}

export const BanSteamModal = ({
    open,
    setOpen,
    reportId,
    onSuccess,
    steamId
}: BanModalProps<IAPIBanRecord>) => {
    const [targetSteamId, setTargetSteamId] = useState<SteamID>(
        steamId ?? new SteamID('')
    );
    const [steamIdInput, setSteamIdInput] = useState<string>('');
    const [duration, setDuration] = useState<Duration>(Duration.dur48h);
    const [customDuration, setCustomDuration] = useState<string>('');
    const [actionType, setActionType] = useState<BanType>(BanType.Banned);
    const [banReason, setBanReason] = useState<BanReason>(BanReason.Cheating);
    const [noteText, setNoteText] = useState<string>('');
    const [reasonText, setReasonText] = useState<string>('');

    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (!targetSteamId) {
            sendFlash('error', 'no steamId');
            return;
        }
        if (banReason == BanReason.Custom && customDuration == '') {
            sendFlash('error', 'Custom duration cannot be empty');
            return;
        }
        const dur = duration == Duration.durCustom ? customDuration : duration;
        if (!dur) {
            sendFlash('error', 'Custom duration invalid');
            return;
        }
        const opts: BanPayloadSteam = {
            target_id: targetSteamId.toString(),
            ban_type: actionType,
            duration: dur,
            reason_text: reasonText,
            reason: banReason,
            note: noteText
        };
        if (reportId) {
            opts.report_id = reportId as number;
        }
        apiCreateBanSteam(opts)
            .then((ban) => {
                sendFlash('success', `Ban created successfully: ${ban.ban_id}`);
                onSuccess && onSuccess(ban);
            })
            .catch((err) => {
                sendFlash('error', `Failed to create ban: ${err}`);
            });
    }, [
        targetSteamId,
        banReason,
        customDuration,
        duration,
        actionType,
        reasonText,
        noteText,
        reportId,
        sendFlash,
        onSuccess
    ]);

    const handleActionTypeChange = (evt: SelectChangeEvent<BanType>) => {
        setActionType(evt.target.value as BanType);
    };

    const handleUpdateNote = (evt: ChangeEvent<HTMLInputElement>) => {
        setNoteText((evt.target as HTMLInputElement).value);
    };

    return (
        <ConfirmationModal
            open={open}
            setOpen={setOpen}
            onSuccess={() => {
                setOpen(false);
            }}
            onCancel={() => {
                setOpen(false);
            }}
            onAccept={() => {
                handleSubmit();
            }}
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
        >
            <Stack spacing={2}>
                <Heading>Ban Player</Heading>
                {!steamId && (
                    <ProfileSelectionInput
                        fullWidth
                        onProfileSuccess={(profile) => {
                            if (profile) {
                                setTargetSteamId(profile.player.steam_id);
                            } else {
                                setTargetSteamId(new SteamID(''));
                            }
                        }}
                        input={steamIdInput}
                        setInput={setSteamIdInput}
                    />
                )}
                <Stack spacing={3} alignItems={'center'}>
                    <FormControl fullWidth>
                        <InputLabel id="actionType-label">
                            Action Type
                        </InputLabel>
                        <Select<BanType>
                            fullWidth
                            label={'Action Type'}
                            labelId="actionType-label"
                            id="actionType-helper"
                            value={actionType}
                            defaultValue={BanType.Banned}
                            onChange={handleActionTypeChange}
                        >
                            {[BanType.Banned, BanType.NoComm].map((v) => (
                                <MenuItem key={`time-${v}`} value={v}>
                                    {v == BanType.NoComm ? 'Mute' : 'Ban'}
                                </MenuItem>
                            ))}
                        </Select>
                        <FormHelperText>
                            Choosing custom will allow you to input a custom
                            duration
                        </FormHelperText>
                    </FormControl>
                    <FormControl fullWidth>
                        <InputLabel id="steam-reason-label">Reason</InputLabel>
                        <Select<BanReason>
                            labelId="steam-reason-label"
                            id="steam-reason"
                            value={banReason}
                            onChange={(evt: SelectChangeEvent<BanReason>) => {
                                setBanReason(evt.target.value as BanReason);
                            }}
                        >
                            {banReasonsList.map((v) => (
                                <MenuItem key={`time-${v}`} value={v}>
                                    {BanReasons[v]}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    {banReason == BanReason.Custom && (
                        <TextField
                            fullWidth
                            label={'Custom Reason'}
                            value={reasonText}
                            onChange={(evt) => {
                                setReasonText(evt.target.value);
                            }}
                        />
                    )}
                    <FormControl fullWidth>
                        <InputLabel id="steam-duration-label">
                            Duration
                        </InputLabel>
                        <Select<Duration>
                            fullWidth
                            label={'Ban Duration'}
                            labelId="steam-duration-label"
                            id="steam-duration"
                            value={duration}
                            onChange={(evt: SelectChangeEvent<Duration>) => {
                                setDuration(evt.target.value as Duration);
                            }}
                        >
                            {Durations.map((v) => (
                                <MenuItem key={`time-${v}`} value={v}>
                                    {v}
                                </MenuItem>
                            ))}
                        </Select>
                        <FormHelperText>
                            Choosing custom will allow you to input a custom
                            duration
                        </FormHelperText>
                    </FormControl>

                    {duration == Duration.durCustom && (
                        <TextField
                            fullWidth
                            label={'Custom Duration'}
                            id={'customDuration'}
                            value={customDuration}
                            onChange={(evt) => {
                                setCustomDuration(evt.target.value);
                            }}
                        />
                    )}

                    <TextField
                        fullWidth
                        id="note-field"
                        label="Moderator Notes (hidden from public)"
                        multiline
                        value={noteText}
                        onChange={handleUpdateNote}
                        rows={10}
                        variant="outlined"
                    />
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
