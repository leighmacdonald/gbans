import { useState } from 'react';
import { eventName, EventTypeByName } from '../api';
import { SelectChangeEvent } from '@mui/material/Select';
import FormControl from '@mui/material/FormControl';
import MenuItem from '@mui/material/MenuItem';
import InputLabel from '@mui/material/InputLabel';
import Select from '@mui/material/Select';
import * as React from 'react';
import { SelectOption } from '../page/AdminServerLog';

export interface EventTypeSelectProps {
    setEventTypes: (types: number[]) => void;
}

export const EventTypeSelect = ({ setEventTypes }: EventTypeSelectProps) => {
    const [selectedEventTypes, setSelectedEventTypes] = useState<number[]>([]);
    const opts: SelectOption[] = Object.values(EventTypeByName).map((v) => {
        return { value: v, title: eventName(v) };
    });

    const containsAll = (f: number[]): boolean => {
        return f.filter((f) => f == -1).length > 0;
    };

    const handleChange = (event: SelectChangeEvent<number[]>) => {
        let newValue: number[];
        const values = event.target.value as number[];
        if (
            !values ||
            (!containsAll(selectedEventTypes) && containsAll(values))
        ) {
            newValue = [];
        } else if (values.length > 1) {
            newValue = values.filter((f) => f >= 0);
        } else {
            newValue = values;
        }
        setSelectedEventTypes(newValue);
        setEventTypes(newValue);
    };

    return (
        <FormControl fullWidth>
            <InputLabel id="event_types-select-label">Event Types</InputLabel>
            <Select<number[]>
                labelId="event_types-select-label"
                multiple
                id="event_types-select"
                value={selectedEventTypes}
                label="Event Types"
                onChange={handleChange}
            >
                <MenuItem value={-1}>All</MenuItem>
                {opts.map((s) => (
                    <MenuItem value={s.value} key={s.value}>
                        {s.title}
                    </MenuItem>
                ))}
            </Select>
        </FormControl>
    );
};
