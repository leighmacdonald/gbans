import { ReactNode } from 'react';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectProps } from '@mui/material/Select';
import { useFieldContext } from '../../contexts/formContext.tsx';

type Props<TData> = {
    label?: string;
    labelLoading?: string;
    items: TData[];
    renderMenu: (item: TData) => ReactNode;
} & SelectProps;

export const SelectField = <TData,>(props: Props<TData>) => {
    const field = useFieldContext<TData>();

    return (
        <FormControl fullWidth>
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
                {props.items.map(props.renderMenu)}
            </Select>
        </FormControl>
    );
};
