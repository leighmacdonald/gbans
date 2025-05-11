import { ReactNode } from 'react';
import { FormHelperText } from '@mui/material';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectProps } from '@mui/material/Select';
import { useStore } from '@tanstack/react-form';
import { useFieldContext } from '../../../contexts/formContext.tsx';
import { renderHelpText } from './renderHelpText.ts';

type Props<TData> = {
    label?: string;
    labelLoading?: string;
    items: TData[];
    renderItem: (item: TData) => ReactNode;
    helpText?: string;
} & SelectProps;

export const SelectField = <TData,>(props: Props<TData>) => {
    const field = useFieldContext<TData>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    return (
        <FormControl fullWidth error={errors.length > 0}>
            <InputLabel id={`select-label-${props.name}`}>{props.label}</InputLabel>
            <Select
                {...props}
                id={`select-${props.name}`}
                fullWidth
                variant={'filled'}
                onChange={(event) => {
                    field.handleChange(event.target.value as TData);
                }}
            >
                {props.items.map(props.renderItem)}
            </Select>
            <FormHelperText>{renderHelpText(errors, props.helpText)}</FormHelperText>
        </FormControl>
    );
};
