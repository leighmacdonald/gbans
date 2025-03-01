import { ReactNode } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent, SelectProps } from '@mui/material/Select';
import { FieldProps } from './common.ts';

type SelectFieldProps<T> = {
    items: T[];
    renderMenu: (item: T) => ReactNode;
} & FieldProps<T>;

export const SelectFieldSimple = <T,>({
    value,
    label,
    handleChange,
    handleBlur,
    items,
    renderMenu
}: SelectFieldProps<T> & SelectProps<T>) => {
    return (
        <FormControl fullWidth>
            <InputLabel id="server-select-label">{label}</InputLabel>
            <Select
                fullWidth
                value={value}
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
