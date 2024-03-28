import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import { PlayerProfile } from '../../api';
import { Nullable } from '../../util/types.ts';

export interface BaseFormikInputProps {
    id?: string;
    label?: string;
    initialValue?: string;
    isReadOnly?: boolean;
    onProfileSuccess?: (profile: Nullable<PlayerProfile>) => void;
}

export type SourceIDFieldValue = {
    source_id: string;
};

export const SourceIDField = ({ disabled = false }: { disabled?: boolean }) => {
    const { values, touched, errors, handleChange } =
        useFormikContext<SourceIDFieldValue>();
    return (
        <TextField
            variant={'outlined'}
            fullWidth
            disabled={disabled}
            name={'source_id'}
            id={'source_id'}
            label={'Author Steam ID'}
            value={values.source_id}
            onChange={handleChange}
            error={touched.source_id && Boolean(errors.source_id)}
            helperText={
                touched.source_id && errors.source_id && `${errors.source_id}`
            }
        />
    );
};
