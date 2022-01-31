import React, { useState } from 'react';
import { SelectChangeEvent } from '@mui/material/Select';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import ButtonGroup from '@mui/material/ButtonGroup';
import Button from '@mui/material/Button';
import DateTimePicker from '@mui/lab/DateTimePicker';
import TextField from '@mui/material/TextField';
import Switch from '@mui/material/Switch';
import FormControlLabel from '@mui/material/FormControlLabel';
import { ProfileSelectionInput } from '../component/ProfileSelectionInput';
import { useTimer } from 'react-timer-hook';
import { IPInput } from '../component/IPInput';
import { ServerSelect } from '../component/ServerSelect';
import { LogRows } from '../component/LogRow';
import { EventTypeSelect } from '../component/EventTypeSelect';
import { findLogs, LogQueryOpts, PlayerProfile, ServerEvent } from '../api';
import FormGroup from '@mui/material/FormGroup';

export interface SelectOption {
    title: string;
    value: number;
}

export const AdminServerLog = (): JSX.Element => {
    const [rate, setRate] = useState<number>(5);
    const [limit, setLimit] = useState<number>(50);
    const [logs, setLogs] = useState<ServerEvent[]>([]);
    const [playerProfile, setPlayerProfile] = useState<PlayerProfile>();
    const [eventTypes, setEventTypes] = useState<number[]>([]);
    const [afterDate, setAfterDate] = React.useState<Date | null>(null);
    const [beforeDate, setBeforeDate] = React.useState<Date | null>(null);
    const [selectedServerIDs, setSelectedServerIDs] = useState<number[]>([0]);

    const { restart, pause, resume } = useTimer({
        expiryTimestamp: new Date(),
        autoStart: true,
        onExpire: async () => {
            const opts: LogQueryOpts = {
                source_id: playerProfile?.player.steam_id ?? '',
                limit: limit,
                order_desc: true,
                servers: selectedServerIDs,
                log_types: eventTypes
            };
            if (afterDate) {
                opts.sent_after = afterDate;
            }
            if (beforeDate) {
                opts.sent_before = beforeDate;
            }
            setLogs((await findLogs(opts)) ?? []);
            // Restart timer
            const time = new Date();
            time.setSeconds(time.getSeconds() + rate);
            restart(time);
        }
    });

    const handleRateChange = (event: SelectChangeEvent<number>) => {
        setRate(event.target.value as number);
    };

    const handleLimitChange = (event: SelectChangeEvent<number>) => {
        setLimit(event.target.value as number);
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
                                    <MenuItem value={5}>5 Seconds</MenuItem>
                                    <MenuItem value={15}>15 Seconds</MenuItem>
                                    <MenuItem value={30}>30 Seconds</MenuItem>
                                    <MenuItem value={60}>60 Seconds</MenuItem>
                                </Select>
                            </FormControl>

                            <FormGroup sx={{ marginLeft: 0 }}>
                                <FormControlLabel
                                    onChange={(_, checked) => {
                                        if (checked) {
                                            resume();
                                        } else {
                                            pause();
                                        }
                                    }}
                                    control={<Switch defaultChecked />}
                                    label="Refresh"
                                    labelPlacement={'bottom'}
                                />
                            </FormGroup>
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
                            <FormControl sx={{ m: 1, minWidth: 120 }}>
                                <InputLabel id="limit-label">Limit</InputLabel>
                                <Select<number>
                                    labelId="limit-label"
                                    color={'secondary'}
                                    id="limit-rate"
                                    value={limit}
                                    label="Limit"
                                    onChange={handleLimitChange}
                                >
                                    <MenuItem value={10}>10</MenuItem>
                                    <MenuItem value={25}>25</MenuItem>
                                    <MenuItem value={50}>50</MenuItem>
                                    <MenuItem value={100}>100</MenuItem>
                                    <MenuItem value={1000}>1000</MenuItem>
                                    <MenuItem value={10000}>Hurt Me</MenuItem>
                                </Select>
                            </FormControl>
                            <ButtonGroup>
                                <Button color={'error'}>Reset</Button>
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
