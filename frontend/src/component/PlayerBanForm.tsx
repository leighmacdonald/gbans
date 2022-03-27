import * as React from 'react';
import { SyntheticEvent, useState } from 'react';
import IPCIDR from 'ip-cidr';
import { apiCreateBan, BanPayload, PlayerProfile } from '../api';
import { Nullable } from '../util/types';
import { log } from '../util/errors';
import Button from '@mui/material/Button';
import FormControl from '@mui/material/FormControl';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormHelperText from '@mui/material/FormHelperText';
import FormLabel from '@mui/material/FormLabel';
import InputLabel from '@mui/material/InputLabel';
import Radio from '@mui/material/Radio';
import RadioGroup from '@mui/material/RadioGroup';
import Select from '@mui/material/Select';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import VoiceOverOffSharp from '@mui/icons-material/VoiceOverOffSharp';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import { ProfileSelectionInput } from './ProfileSelectionInput';

const ip2int = (ip: string): number =>
    ip
        .split('.')
        .reduce((ipInt, octet) => (ipInt << 8) + parseInt(octet, 10), 0) >>> 0;

type BanType = 'network' | 'steam';

type ActionType = 'ban' | 'mute';

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
    durInf = '∞',
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
    profile?: PlayerProfile;
}

export const PlayerBanForm = ({
    onProfileChanged
}: PlayerBanFormProps): JSX.Element => {
    const [duration, setDuration] = React.useState<Duration>(Duration.dur48h);
    const [actionType, setActionType] = React.useState<ActionType>('ban');
    const [reasonText, setReasonText] = React.useState<string>('');
    const [noteText, setNoteText] = React.useState<string>('');
    const [network, setNetwork] = React.useState<string>('');
    const [networkSize, setNetworkSize] = React.useState<number>(0);
    const [banType, setBanType] = React.useState<BanType>('steam');
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>();
    const [steamID, setSteamID] = useState<string>('');

    const handleSubmit = React.useCallback(async () => {
        if (!profile || profile?.player?.steam_id === '') {
            return;
        }
        const opts: BanPayload = {
            steam_id: profile.player.steamid ?? '',
            ban_type: 2,
            duration: duration,
            network: banType === 'steam' ? '' : network,
            reason_text: reasonText,
            reason: 0
        };
        const r = await apiCreateBan(opts);
        log(`${r}`);
    }, [profile, reasonText, network, banType, duration]);

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

    const handleUpdateReasonText = (evt: SyntheticEvent) => {
        setReasonText((evt.target as HTMLInputElement).value);
    };

    const handleUpdateDuration = (
        evt: React.ChangeEvent<{ value: unknown }> | any
    ) => {
        setDuration((evt.target.value as Duration) ?? Duration.durInf);
    };

    const handleActionTypeChange = (
        evt: React.ChangeEvent<{ value: unknown }> | any
    ) => {
        setActionType((evt.target.value as ActionType) ?? 'ban');
    };

    const handleUpdateNote = (evt: SyntheticEvent | any) => {
        setNoteText((evt.target as HTMLInputElement).value);
    };

    const onChangeType = (evt: React.ChangeEvent<HTMLInputElement> | any) => {
        setBanType(evt.target.value as BanType);
    };

    return (
        <Stack spacing={3} padding={3}>
            {profile && (
                <ProfileSelectionInput
                    fullWidth
                    input={steamID}
                    setInput={setSteamID}
                    onProfileSuccess={(p) => {
                        setProfile(p);
                        onProfileChanged && onProfileChanged(p);
                    }}
                />
            )}
            <FormControl fullWidth>
                <InputLabel id="actionType-label">Action Type</InputLabel>
                <Select
                    fullWidth
                    labelId="actionType-label"
                    id="actionType-helper"
                    value={actionType}
                    onChange={handleActionTypeChange}
                >
                    {['ban', 'mute'].map((v) => (
                        <MenuItem key={`time-${v}`} value={v}>
                            {v}
                        </MenuItem>
                    ))}
                </Select>
                <FormHelperText>
                    Choosing custom will allow you to input a custom duration
                </FormHelperText>
            </FormControl>
            {actionType == 'ban' && (
                <FormControl component="fieldset" fullWidth>
                    <FormLabel component="legend">Ban Type</FormLabel>
                    <RadioGroup
                        aria-label="Ban Type"
                        name="Ban Type"
                        value={banType}
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
            )}
            {actionType == 'ban' && banType === 'network' && (
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

            <TextField
                fullWidth
                id={'reason'}
                label={actionType == 'ban' ? 'Ban Reason' : 'Mute Reason'}
                onChange={handleUpdateReasonText}
            />
            <FormControl fullWidth>
                <InputLabel id="duration-label">Ban Duration</InputLabel>
                <Select
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
                    Choosing custom will allow you to input a custom duration
                </FormHelperText>
            </FormControl>

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
                fullWidth
                key={'submit'}
                value={'Create Ban'}
                onClick={handleSubmit}
                variant="contained"
                color="primary"
                startIcon={<VoiceOverOffSharp />}
            >
                Ban Player
            </Button>
        </Stack>
    );
};
