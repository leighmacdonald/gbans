import React, { ChangeEvent, useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import {
    apiCreateBanASN,
    BanReason,
    BanReasons,
    banReasonsList,
    BanType,
    Duration,
    Durations,
    IAPIBanASNRecord
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

export interface BanASNModalProps
    extends ConfirmationModalProps<IAPIBanASNRecord> {
    asnNum?: number;
}

export const BanASNModal = ({ open, setOpen, onSuccess }: BanASNModalProps) => {
    const [targetSteamId, setTargetSteamId] = useState<string>('');
    const [duration, setDuration] = useState<Duration>(Duration.dur48h);
    const [customDuration, setCustomDuration] = useState<string>('');
    const [banReason, setBanReason] = useState<BanReason>(BanReason.Cheating);
    const [noteText, setNoteText] = useState<string>('');
    const [reasonText, setReasonText] = useState<string>('');
    const [asNum, setASNum] = useState<number>(0);

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
        let targetId = new SteamID('');
        if (targetSteamId != '') {
            try {
                const id = new SteamID(targetSteamId);
                if (!id.isValidIndividual()) {
                    sendFlash('error', 'Target steam id invalid');
                    return;
                }
                targetId = id;
            } catch (e) {
                sendFlash('error', 'Target steam id invalid');
                return;
            }
        }
        apiCreateBanASN({
            target_id: targetId.toString(),
            duration: dur,
            as_num: asNum,
            reason_text: reasonText,
            reason: banReason,
            note: noteText,
            ban_type: BanType.Banned
        })
            .then((ban) => {
                sendFlash(
                    'success',
                    `ASN ban created successfully: ${ban.ban_asn_id}`
                );
                onSuccess && onSuccess(ban);
            })
            .catch((err) => {
                sendFlash('error', `Failed to create asn ban: ${err}`);
            });
    }, [
        targetSteamId,
        banReason,
        customDuration,
        duration,
        asNum,
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
                <Heading>Ban AS Number</Heading>
                <Stack spacing={3} alignItems={'center'}>
                    <ProfileSelectionInput
                        fullWidth
                        onProfileSuccess={(profile) => {
                            if (profile) {
                                setTargetSteamId(
                                    profile.player.steam_id.toString
                                );
                            } else {
                                setTargetSteamId('');
                            }
                        }}
                        input={targetSteamId}
                        setInput={setTargetSteamId}
                    />

                    <TextField
                        fullWidth
                        id={'as_num'}
                        label={'Autonomous System Number'}
                        onChange={(evt) => {
                            setASNum(parseInt(evt.target.value));
                        }}
                    />

                    <FormControl fullWidth>
                        <InputLabel id="asn-reason-label">
                            Ban Duration
                        </InputLabel>
                        <Select<BanReason>
                            labelId="asn-reason-label"
                            id="asn-reason"
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
                        <InputLabel id="asn-duration-label">
                            Ban Duration
                        </InputLabel>
                        <Select<Duration>
                            fullWidth
                            labelId="asn-duration-label"
                            id="duration-helper"
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
                            id={'customASNDuration'}
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
