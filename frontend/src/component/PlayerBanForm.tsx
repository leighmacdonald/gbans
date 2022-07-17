import * as React from 'react';
import { ChangeEvent, SyntheticEvent, useCallback, useState } from 'react';
import IPCIDR from 'ip-cidr';
import {
    apiCreateBan,
    Ban,
    BanPayload,
    BanReason,
    BanReasons,
    banReasonsList,
    BanType,
    PlayerProfile,
    SteamID
} from '../api';
import { Nullable } from '../util/types';
import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormHelperText from '@mui/material/FormHelperText';
import FormLabel from '@mui/material/FormLabel';
import InputLabel from '@mui/material/InputLabel';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import VoiceOverOffSharp from '@mui/icons-material/VoiceOverOffSharp';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

const ip2int = (ip: string): number =>
    ip
        .split('.')
        .reduce((ipInt, octet) => (ipInt << 8) + parseInt(octet, 10), 0) >>> 0;

export enum Duration {
    dur15m = '15m',
    dur6h = '6h',
    dur12h = '12h',
    dur24h = '24h',
    dur48h = '48h',
    dur72h = '72h',
    dur1w = '1w',
    dur2w = '2w',
    dur1M = '1M',
    dur6M = '6M',
    dur1y = '1y',
    durInf = '0',
    durCustom = 'custom'
}

const Durations = [
    Duration.dur15m,
    Duration.dur6h,
    Duration.dur12h,
    Duration.dur24h,
    Duration.dur48h,
    Duration.dur72h,
    Duration.dur1w,
    Duration.dur2w,
    Duration.dur1M,
    Duration.dur6M,
    Duration.dur1y,
    Duration.durInf,
    Duration.durCustom
];

export interface PlayerBanFormProps {
    onProfileChanged?: (profile: Nullable<PlayerProfile>) => void;
    onBanSuccess?: (ban: Ban) => void;
    steamId: SteamID;
    reportId?: number;
}

type BanMethod = 'steam' | 'network';

export const PlayerBanForm = ({
    steamId,
    reportId,
    onBanSuccess
}: PlayerBanFormProps): JSX.Element => {
    const [duration, setDuration] = useState<Duration>(Duration.dur48h);
    const [customDuration, setCustomDuration] = useState<string>('');
    const [actionType, setActionType] = useState<BanType>(BanType.Banned);
    const [banReason, setBanReason] = useState<BanReason>(BanReason.Cheating);
    const [noteText, setNoteText] = useState<string>('');
    const [reasonText, setReasonText] = useState<string>('');
    const [network, setNetwork] = useState<string>('');
    const [networkSize, setNetworkSize] = useState<number>(0);
    const [banMethodType, setBanMethodType] = useState<BanMethod>('steam');

    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (!steamId) {
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
        const opts: BanPayload = {
            steam_id: steamId,
            ban_type: actionType,
            duration: dur,
            network: banMethodType === 'steam' ? '' : network,
            reason_text: reasonText,
            reason: banReason,
            note: noteText
        };
        if (reportId) {
            opts.report_id = reportId as number;
        }
        apiCreateBan(opts)
            .then((ban) => {
                sendFlash('success', `Ban created successfully: ${ban.ban_id}`);
                onBanSuccess && onBanSuccess(ban);
            })
            .catch((err) => {
                sendFlash('error', `Failed to create ban: ${err}`);
            });
    }, [
        steamId,
        banReason,
        customDuration,
        duration,
        actionType,
        banMethodType,
        network,
        reasonText,
        noteText,
        reportId,
        sendFlash,
        onBanSuccess
    ]);

    const handleUpdateNetwork = (evt: SyntheticEvent) => {
        const value = (evt.target as HTMLInputElement).value;
        setNetwork(value);
        if (value !== '') {
            try {
                const cidr = new IPCIDR(value);
                if (cidr != undefined) {
                    setNetworkSize(
                        ip2int(cidr?.end()) - ip2int(cidr?.start()) + 1
                    );
                }
            } catch (e) {
                if (e instanceof TypeError) {
                    // TypeError on invalid input we can ignore
                } else {
                    throw e;
                }
            }
        }
    };

    const handleUpdateReason = (evt: SelectChangeEvent<BanReason>) => {
        setBanReason(evt.target.value as BanReason);
    };

    const handleUpdateDuration = (evt: SelectChangeEvent<Duration>) => {
        setDuration(evt.target.value as Duration);
    };

    const handleActionTypeChange = (evt: SelectChangeEvent<BanType>) => {
        setActionType(evt.target.value as BanType);
    };

    const handleUpdateNote = (evt: ChangeEvent<HTMLInputElement>) => {
        setNoteText((evt.target as HTMLInputElement).value);
    };

    const onChangeType = (evt: ChangeEvent<HTMLInputElement>) => {
        setBanMethodType(evt.target.value as BanMethod);
    };

    return (
        <>
            <Heading>Ban A Player</Heading>
            <Stack spacing={3} padding={2}>
                <FormControl fullWidth>
                    <InputLabel id="actionType-label">Action Type</InputLabel>
                    <Select<BanType>
                        fullWidth
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

                <FormControl component="fieldset" fullWidth>
                    <FormLabel component="legend">Ban Type</FormLabel>
                    <RadioGroup
                        aria-label="Ban Type"
                        name="Ban Type"
                        value={banMethodType}
                        onChange={onChangeType}
                        defaultValue={'steam'}
                        row
                    >
                        <FormControlLabel
                            value="steam"
                            control={<Radio />}
                            label="Steam"
                        />
                        <FormControlLabel
                            value="network"
                            control={<Radio />}
                            label="IP / Network"
                        />
                    </RadioGroup>
                </FormControl>

                {banMethodType === 'network' && (
                    <>
                        <TextField
                            fullWidth={true}
                            id={'network'}
                            label={'Network Range (CIDR Format)'}
                            onChange={handleUpdateNetwork}
                        />
                        <Typography variant={'body1'}>
                            Current number of hosts in range: {networkSize}
                        </Typography>
                    </>
                )}
                <Select<BanReason>
                    fullWidth
                    labelId="reason-label"
                    id="reason-helper"
                    value={banReason}
                    onChange={handleUpdateReason}
                >
                    {banReasonsList.map((v) => (
                        <MenuItem key={`time-${v}`} value={v}>
                            {BanReasons[v]}
                        </MenuItem>
                    ))}
                </Select>
                {banReason == BanReason.Custom && (
                    <FormControl fullWidth>
                        <InputLabel id="reasonText-label">Reason</InputLabel>
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
                    <InputLabel id="duration-label">Ban Duration</InputLabel>
                    <Select<Duration>
                        fullWidth
                        labelId="duration-label"
                        id="duration-helper"
                        value={duration}
                        onChange={handleUpdateDuration}
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
                        onChange={handleUpdateNote}
                        rows={10}
                        variant="outlined"
                    />
                </FormControl>

                <Button
                    key={'submit'}
                    value={'Create Ban'}
                    onClick={handleSubmit}
                    variant="contained"
                    color="error"
                    startIcon={<VoiceOverOffSharp />}
                >
                    Ban Player
                </Button>
            </Stack>
        </>
    );
};
