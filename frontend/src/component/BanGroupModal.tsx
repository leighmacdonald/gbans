import React, { ChangeEvent, useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import {
    apiCreateBanGroup,
    BanPayloadGroup,
    BanReason,
    BanReasons,
    banReasonsList,
    BanType,
    Duration,
    Durations,
    IAPIBanGroupRecord
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
import { ProfileSelectionInput } from './ProfileSelectionInput';
import SteamID from 'steamid';
import { logErr } from '../util/errors';

export interface BanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    asnNum?: number;
}

export const BanGroupModal = ({
    open,
    setOpen,
    onSuccess
}: BanGroupModalProps) => {
    const [targetSteamId, setTargetSteamId] = useState<SteamID>(
        new SteamID('')
    );
    const [input, setInput] = useState<string>('');
    const [duration, setDuration] = useState<Duration>(Duration.durInf);
    const [customDuration, setCustomDuration] = useState<string>('');
    const [banReason, setBanReason] = useState<BanReason>(BanReason.External);
    const [noteText, setNoteText] = useState<string>('');
    const [reasonText, setReasonText] = useState<string>('');
    const [groupId, setGroupId] = useState<string>('');

    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (banReason == BanReason.Custom && customDuration == '') {
            sendFlash('error', 'Custom duration cannot be empty');
            return;
        }
        const dur = duration == Duration.durCustom ? customDuration : duration;
        if (!dur) {
            sendFlash('error', 'Custom duration invalid');
            return;
        }
        const opts: BanPayloadGroup = {
            target_id: targetSteamId.toString(),
            duration: dur,
            group_id: groupId,
            reason_text: reasonText,
            reason: banReason,
            note: noteText,
            ban_type: BanType.Banned
        };

        apiCreateBanGroup(opts)
            .then((response) => {
                if (!response.status || !response.result) {
                    sendFlash('error', `Fialed to create group ban`);
                    return;
                }
                sendFlash(
                    'success',
                    `Steam group ban created successfully: ${response.result.ban_group_id}`
                );
                onSuccess && onSuccess(response.result);
            })
            .catch(logErr);
    }, [
        banReason,
        customDuration,
        duration,
        targetSteamId,
        groupId,
        reasonText,
        noteText,
        sendFlash,
        onSuccess
    ]);

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
                <Heading>Ban Steam Group</Heading>
                <Stack spacing={3} alignItems={'center'}>
                    <ProfileSelectionInput
                        fullWidth
                        onProfileSuccess={(profile) => {
                            if (profile) {
                                setTargetSteamId(profile.player.steam_id);
                            } else {
                                setTargetSteamId(new SteamID(''));
                            }
                        }}
                        input={input}
                        setInput={setInput}
                    />
                    <TextField
                        fullWidth
                        id={'group_id'}
                        label={'Steam Group ID'}
                        onChange={(evt) => {
                            setGroupId(evt.target.value);
                        }}
                    />
                    <FormControl fullWidth>
                        <InputLabel id="group-reason-label">Reason</InputLabel>
                        <Select<BanReason>
                            fullWidth
                            labelId={'group-reason-label'}
                            label={'Reason'}
                            id="reason-helper"
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
                            id={'reasonText'}
                            value={reasonText}
                            onChange={(evt) => {
                                setReasonText(evt.target.value);
                            }}
                        />
                    )}
                    <FormControl fullWidth>
                        <InputLabel id="group-duration-label">
                            Ban Duration
                        </InputLabel>
                        <Select<Duration>
                            fullWidth
                            labelId="group-duration-label"
                            id="duration-helper"
                            value={duration}
                            defaultValue={Duration.durInf}
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
                            label={'Custom Curation'}
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
                        onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                            setNoteText((evt.target as HTMLInputElement).value);
                        }}
                        rows={10}
                        variant="outlined"
                    />
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
