import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import { FilterAction, filterActionString } from '../../api/filters';

interface FilterActionFieldProps {
    action: FilterAction;
}

export const FilterActionField = () => {
    const { values, touched, errors, handleChange } =
        useFormikContext<FilterActionFieldProps>();
    return (
        <FormControl fullWidth>
            <InputLabel id="action-label">On Trigger Action</InputLabel>
            <Select<FilterAction>
                labelId="action-label"
                id="action"
                name={'action'}
                value={values.action}
                onChange={handleChange}
                error={touched.action && Boolean(errors.action)}
            >
                {[FilterAction.Kick, FilterAction.Mute, FilterAction.Ban].map(
                    (v) => (
                        <MenuItem key={`action-${v}`} value={v}>
                            {filterActionString(v)}
                        </MenuItem>
                    )
                )}
            </Select>
            <FormHelperText>{touched.action && errors.action}</FormHelperText>
        </FormControl>
    );
};
