import { ReactNode } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import { FieldProps } from './common.ts';

type SelectFieldProps<T> = {
    items: T[];
    renderMenu: (item: T) => ReactNode;
} & FieldProps<T>;

export const SelectFieldSimple = <T,>({ state, label, handleChange, handleBlur, items, renderMenu }: SelectFieldProps<T>) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="server-select-label">{label}</InputLabel>
            <Select
                fullWidth
                value={state.value}
                label={label}
                onChange={(e: SelectChangeEvent<T>) => {
                    handleChange(e.target.value as T);
                }}
                onBlur={handleBlur}
            >
                {items.map(renderMenu)}
            </Select>
        </FormControl>
    );
};
