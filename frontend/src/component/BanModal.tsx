import React, {
    ChangeEvent,
    SyntheticEvent,
    useCallback,
    useState
} from 'react';
import { ProfileSelectionInput } from './ProfileSelectionInput';
import Stack from '@mui/material/Stack';
import {
    apiCreateBan,
    IAPIBanRecord,
    BanPayload,
    BanReason,
    BanReasons,
    banReasonsList,
    BanType,
    SteamID
} from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormHelperText from '@mui/material/FormHelperText';
import FormLabel from '@mui/material/FormLabel';
import RadioGroup from '@mui/material/RadioGroup';
import FormControlLabel from '@mui/material/FormControlLabel';
import Radio from '@mui/material/Radio';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import IPCIDR from 'ip-cidr';
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

type BanMethod = 'steam' | 'network';

export interface BanModalProps<Ban> extends ConfirmationModalProps<Ban> {
    ban?: Ban;
    reportId?: number;
    steamId?: SteamID;
}

export const BanModal = ({
    open,
    setOpen,
    reportId,
    onSuccess,
    steamId
}: BanModalProps<IAPIBanRecord>) => {
    const [targetSteamId, setTargetSteamId] = useState<SteamID>(
        steamId ?? BigInt(0)
    );
    const [input, setInput] = useState<SteamID>(BigInt(0));
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
        const opts: BanPayload = {
            steam_id: targetSteamId,
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
        banMethodType,
        network,
        reasonText,
        noteText,
        reportId,
        sendFlash,
        onSuccess
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
                                setTargetSteamId(BigInt(0));
                            }
                        }}
                        input={input}
                        setInput={setInput}
                    />
                )}
                <Stack spacing={3} alignItems={'center'}>
                    <FormControl fullWidth>
                        <InputLabel id="actionType-label">
                            Action Type
                        </InputLabel>
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

                    <FormControl>
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
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
