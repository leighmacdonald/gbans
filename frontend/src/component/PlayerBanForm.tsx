import * as React from 'react';
import { SyntheticEvent, useEffect } from 'react';
import IPCIDR from 'ip-cidr';
import {
    apiCreateBan,
    apiGetProfile,
    BanPayload,
    PlayerProfile
} from '../util/api';
import { Nullable } from '../util/types';

import {
    Button,
    FormControl,
    FormControlLabel,
    FormHelperText,
    FormLabel,
    Grid,
    InputLabel,
    MenuItem,
    Radio,
    RadioGroup,
    Select,
    TextField,
    Typography
} from '@material-ui/core';
import { VoiceOverOffSharp } from '@material-ui/icons';
import { log } from '../util/errors';

const ip2int = (ip: string): number =>
    ip
        .split('.')
        .reduce((ipInt, octet) => (ipInt << 8) + parseInt(octet, 10), 0) >>> 0;

type BanType = 'network' | 'steam';

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
    onProfileChanged?: (profile: PlayerProfile) => void;
}

export const PlayerBanForm = ({
    onProfileChanged
}: PlayerBanFormProps): JSX.Element => {
    const [fSteam, setFSteam] = React.useState<string>(
        'https://steamcommunity.com/id/SQUIRRELLY/'
    );
    const [duration, setDuration] = React.useState<Duration>(Duration.dur48h);
    const [reasonText, setReasonText] = React.useState<string>('');
    const [noteText, setNoteText] = React.useState<string>('');
    const [network, setNetwork] = React.useState<string>('');
    const [networkSize, setNetworkSize] = React.useState<number>(0);
    const [banType, setBanType] = React.useState<BanType>('steam');
    const [profile, setProfile] = React.useState<Nullable<PlayerProfile>>();
    const loadPlayerSummary = async () => {
        try {
            const p = (await apiGetProfile(fSteam)) as PlayerProfile;
            setProfile(p);
            if (onProfileChanged) {
                onProfileChanged(p);
            }
        } catch (e) {
            log(e);
        }
    };
    useEffect(() => {
        // Validate results
    }, [profile]);
    const handleUpdateFSteam = React.useCallback(loadPlayerSummary, [
        onProfileChanged,
        setProfile,
        fSteam
    ]);

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
        evt: React.ChangeEvent<{ value: unknown }>
    ) => {
        setDuration((evt.target.value as Duration) ?? Duration.durInf);
    };
    const handleUpdateNote = (evt: SyntheticEvent) => {
        setNoteText((evt.target as HTMLInputElement).value);
    };

    const onChangeFStream = (evt: React.ChangeEvent<HTMLInputElement>) => {
        setFSteam((evt.target as HTMLInputElement).value);
    };

    const onChangeType = (evt: React.ChangeEvent<HTMLInputElement>) => {
        setBanType(evt.target.value as BanType);
    };

    return (
        <form noValidate>
            <Grid container spacing={3}>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id={'query'}
                        label={'Steam ID / Profile URL'}
                        onChange={onChangeFStream}
                        onBlur={handleUpdateFSteam}
                    />
                </Grid>

                <Grid item xs={12}>
                    <FormControl component="fieldset" fullWidth>
                        <FormLabel component="legend">Ban Type</FormLabel>
                        <RadioGroup
                            aria-label="gender"
                            name="gender1"
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
                </Grid>
                {banType === 'network' && (
                    <>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth={true}
                                id={'network'}
                                label={'Network Range (CIDR Format)'}
                                onChange={handleUpdateNetwork}
                            />
                        </Grid>
                        <Grid item>
                            <Typography variant={'body1'}>
                                Current number of hosts in range: {networkSize}
                            </Typography>
                        </Grid>
                    </>
                )}
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id={'reason'}
                        label={'Ban Reason'}
                        onChange={handleUpdateReasonText}
                    />
                </Grid>
                <Grid item xs={12}>
                    <FormControl fullWidth>
                        <InputLabel id="duration-label">
                            Ban Duration
                        </InputLabel>
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
                            Choosing custom will allow you to input a custom
                            duration
                        </FormHelperText>
                    </FormControl>
                </Grid>
                <Grid item xs={12}>
                    <FormControl fullWidth>
                        <InputLabel id="duration-label">
                            Ban Duration
                        </InputLabel>
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
                            Choosing custom will allow you to input a custom
                            duration
                        </FormHelperText>
                    </FormControl>
                </Grid>
                <Grid item xs={12}>
                    <FormControl fullWidth>
                        <TextField
                            id="note-field"
                            label="Moderator Notes (hidden from public)"
                            multiline
                            value={noteText}
                            onChange={handleUpdateNote}
                            rows={10}
                            defaultValue={noteText}
                            variant="outlined"
                        />
                    </FormControl>
                </Grid>
                <Grid item xs={12}>
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
                </Grid>
            </Grid>
        </form>
    );
};
