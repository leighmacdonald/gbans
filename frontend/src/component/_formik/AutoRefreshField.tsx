import { useEffect } from 'react';
import { useTimer } from 'react-timer-hook';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import { logErr } from '../../util/errors';

interface AutoRefreshFieldProps {
    auto_refresh: number;
}

export const AutoRefreshField = <T,>() => {
    const { values, handleChange, submitForm } = useFormikContext<T & AutoRefreshFieldProps>();

    const { isRunning, restart, pause } = useTimer({
        expiryTimestamp: new Date(),
        autoStart: false,
        onExpire: () => {
            submitForm().catch((reason) => logErr(reason));
        }
    });

    useEffect(() => {
        if (isRunning && values.auto_refresh <= 0) {
            pause();
            return;
        }
        const newTime = new Date();
        newTime.setSeconds(newTime.getSeconds() + values.auto_refresh);
        restart(newTime, true);
    }, [isRunning, pause, restart, values.auto_refresh]);

    return (
        <FormControl fullWidth>
            <InputLabel id="auto_refresh-label">Auto-Refresh</InputLabel>
            <Select<number>
                labelId="auto_refresh-label"
                id="auto_refresh"
                name={'auto_refresh'}
                label="Auto Refresh"
                value={values.auto_refresh}
                onChange={handleChange}
            >
                <MenuItem value={0}>Off</MenuItem>
                <MenuItem value={10}>5s</MenuItem>
                <MenuItem value={15}>15s</MenuItem>
                <MenuItem value={30}>30s</MenuItem>
                <MenuItem value={60}>60s</MenuItem>
            </Select>
        </FormControl>
    );
};
