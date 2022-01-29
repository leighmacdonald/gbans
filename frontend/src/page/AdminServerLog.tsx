import React, { useState } from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Autocomplete from '@mui/material/Autocomplete';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import TextField from '@mui/material/TextField';
import { MsgType } from '../api';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import { SelectChangeEvent } from '@mui/material/Select';
import { ServerSelect } from '../component/ServerSelect';

export interface SelectOption {
    title: string;
    value: number;
}

export const AdminServerLog = (): JSX.Element => {
    const [rate, setRate] = useState<string>('5s');
    const opts: SelectOption[] = [];
    for (const value in Object.keys(MsgType)) {
        if (typeof MsgType[value] !== 'string') {
            continue;
        }
        opts.push({ value: Number(value), title: MsgType[Number(value)] });
    }
    const handleRateChange = (event: SelectChangeEvent) => {
        setRate(event.target.value);
    };
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={9}>
                <Stack spacing={3}>
                    <Paper elevation={1}>
                        <Stack direction={'row'} spacing={3} padding={3}>
                            <Typography variant={'h5'}>Filters</Typography>
                            <Autocomplete
                                multiple
                                limitTags={2}
                                id="msg-types"
                                options={opts}
                                getOptionLabel={(option) => option?.title}
                                defaultValue={[
                                    opts[MsgType.Say],
                                    opts[MsgType.SayTeam]
                                ]}
                                renderInput={(params) => (
                                    <TextField
                                        {...params}
                                        label="Event Types"
                                    />
                                )}
                            />
                            <FormControl
                                variant="filled"
                                sx={{ m: 1, minWidth: 120 }}
                            >
                                <InputLabel id="update-rate-label">
                                    Update Rate
                                </InputLabel>
                                <Select
                                    labelId="update-rate-label"
                                    id="update-rate"
                                    value={rate}
                                    onChange={handleRateChange}
                                >
                                    <MenuItem value={'disable'}>
                                        Disable
                                    </MenuItem>
                                    <MenuItem value={'5s'}>5 Seconds</MenuItem>
                                    <MenuItem value={'15s'}>
                                        15 Seconds
                                    </MenuItem>
                                    <MenuItem value={'30s'}>
                                        30 Seconds
                                    </MenuItem>
                                    <MenuItem value={'60s'}>
                                        60 Seconds
                                    </MenuItem>
                                </Select>
                            </FormControl>
                            <ServerSelect />
                        </Stack>
                    </Paper>
                </Stack>
            </Grid>
            <Grid item xs={3}>
                <Paper elevation={1}>
                    <Box>
                        <Typography variant={'h1'}>cfg</Typography>
                    </Box>
                </Paper>
            </Grid>
        </Grid>
    );
};
