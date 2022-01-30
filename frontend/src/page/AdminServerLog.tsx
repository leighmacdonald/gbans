import React, { useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import { findLogs, LogQueryOpts, PlayerProfile, ServerEvent } from '../api';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import { SelectChangeEvent } from '@mui/material/Select';
import { ServerSelect } from '../component/ServerSelect';
import { LogRows } from '../component/LogRow';
import { EventTypeSelect } from '../component/EventTypeSelect';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import { ProfileSelectionInput } from '../component/ProfileSelectionInput';
import DateTimePicker from '@mui/lab/DateTimePicker';
import TextField from '@mui/material/TextField';
import { useTimer } from 'react-timer-hook';
import { IPInput } from '../component/IPInput';

export interface SelectOption {
    title: string;
    value: number;
}

export const AdminServerLog = (): JSX.Element => {
    const [rate, setRate] = useState<number>(5);
    const [logs, setLogs] = useState<ServerEvent[]>([]);
    const [playerProfile, setPlayerProfile] = useState<PlayerProfile>();
    const [eventTypes, setEventTypes] = useState<number[]>([]);
    const [afterDate, setAfterDate] = React.useState<Date | null>(null);
    const [beforeDate, setBeforeDate] = React.useState<Date | null>(null);
    const [selectedServerIDs, setSelectedServerIDs] = useState<number[]>([0]);

    const { restart } = useTimer({
        expiryTimestamp: new Date(),
        autoStart: true,
        onExpire: async () => {
            const opts: LogQueryOpts = {
                source_id: playerProfile?.player.steam_id ?? '',
                limit: 100,
                order_desc: true,
                servers: selectedServerIDs,
                log_types: eventTypes
            };
            setLogs((await findLogs(opts)) ?? []);
            const time = new Date();
            time.setSeconds(time.getSeconds() + rate);
            restart(time);
        }
    });

    const handleRateChange = (event: SelectChangeEvent<number>) => {
        setRate(event.target.value as number);
    };

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Stack spacing={3}>
                    <Paper elevation={1}>
                        <Stack
                            direction={'row'}
                            spacing={3}
                            paddingLeft={3}
                            paddingRight={3}
                            paddingTop={3}
                        >
                            <ProfileSelectionInput
                                renderFooter={false}
                                fullWidth={true}
                                onProfileSuccess={setPlayerProfile}
                            />
                            <IPInput
                                onCIDRSuccess={(cidr) => {
                                    console.log(cidr);
                                }}
                            />
                            <ServerSelect setServerIDs={setSelectedServerIDs} />
                            <FormControl sx={{ m: 1, minWidth: 120 }}>
                                <InputLabel id="update-rate-label">
                                    Update Rate
                                </InputLabel>
                                <Select<number>
                                    labelId="update-rate-label"
                                    color={'secondary'}
                                    id="update-rate"
                                    value={rate}
                                    label="Update Rate"
                                    onChange={handleRateChange}
                                >
                                    <MenuItem value={0}>Disable</MenuItem>
                                    <MenuItem value={5}>5 Seconds</MenuItem>
                                    <MenuItem value={15}>15 Seconds</MenuItem>
                                    <MenuItem value={30}>30 Seconds</MenuItem>
                                    <MenuItem value={60}>60 Seconds</MenuItem>
                                </Select>
                            </FormControl>
                        </Stack>
                        <Stack direction={'row'} spacing={3} padding={3}>
                            <EventTypeSelect setEventTypes={setEventTypes} />
                            <DateTimePicker
                                renderInput={(props) => (
                                    <TextField {...props} />
                                )}
                                label="Sent After"
                                value={afterDate}
                                onChange={(newValue) => {
                                    setAfterDate(newValue);
                                }}
                            />
                            <DateTimePicker
                                renderInput={(props) => (
                                    <TextField {...props} />
                                )}
                                label="Sent Before"
                                value={beforeDate}
                                onChange={(newValue) => {
                                    setBeforeDate(newValue);
                                }}
                            />
                            <ButtonGroup>
                                <Button color={'error'}>Reset</Button>
                                <Button
                                    color={'secondary'}
                                    variant={'contained'}
                                >
                                    Pause
                                </Button>
                                <Button color={'success'} variant={'contained'}>
                                    Apply
                                </Button>
                            </ButtonGroup>
                        </Stack>
                    </Paper>
                    <Paper elevation={1}>
                        <LogRows events={logs} />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
