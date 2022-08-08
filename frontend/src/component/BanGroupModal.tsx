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

export interface BanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    asnNum?: number;
}

export const BanGroupModal = ({
    open,
    setOpen,
    onSuccess
}: BanGroupModalProps) => {
    const [duration, setDuration] = useState<Duration>(Duration.dur48h);
    const [customDuration, setCustomDuration] = useState<string>('');
    const [banReason, setBanReason] = useState<BanReason>(BanReason.Cheating);
    const [noteText, setNoteText] = useState<string>('');
    const [reasonText, setReasonText] = useState<string>('');
    const [groupId, setGroupId] = useState<bigint>(BigInt(0));

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
            duration: dur,
            group_id: groupId,
            reason_text: reasonText,
            reason: banReason,
            note: noteText,
            ban_type: BanType.Banned
        };

        apiCreateBanGroup(opts)
            .then((ban) => {
                sendFlash(
                    'success',
                    `ASN ban created successfully: ${ban.ban_group_id}`
                );
                onSuccess && onSuccess(ban);
            })
            .catch((err) => {
                sendFlash('error', `Failed to create ban: ${err}`);
            });
    }, [
        banReason,
        customDuration,
        duration,
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
                    <TextField
                        fullWidth={true}
                        id={'group_id'}
                        label={'Steam Group ID'}
                        onChange={(evt) => {
                            setGroupId(BigInt(parseInt(evt.target.value)));
                        }}
                    />

                    <Select<BanReason>
                        fullWidth
                        labelId="reason-label"
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
                    {banReason == BanReason.Custom && (
                        <FormControl fullWidth>
                            <InputLabel id="reasonText-label">
                                Reason
                            </InputLabel>
                            <TextField
                                id={'reasonText'}
                                value={reasonText}
                                onChange={(evt) => {
                                    setReasonText(evt.target.value);
                                }}
                            />
                        </FormControl>
                    )}
                    <FormControl fullWidth>
                        <InputLabel id="duration-label">
                            Ban Duration
                        </InputLabel>
                        <Select<Duration>
                            fullWidth
                            labelId="duration-label"
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
                        <FormControl fullWidth>
                            <InputLabel id="customDuration-label">
                                Custom Duration
                            </InputLabel>
                            <TextField
                                id={'customDuration'}
                                value={customDuration}
                                onChange={(evt) => {
                                    setCustomDuration(evt.target.value);
                                }}
                            />
                        </FormControl>
                    )}

                    <FormControl fullWidth>
                        <TextField
                            id="note-field"
                            label="Moderator Notes (hidden from public)"
                            multiline
                            value={noteText}
                            onChange={(evt: ChangeEvent<HTMLInputElement>) => {
                                setNoteText(
                                    (evt.target as HTMLInputElement).value
                                );
                            }}
                            rows={10}
                            variant="outlined"
                        />
                    </FormControl>
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
