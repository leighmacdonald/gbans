import FormControl from '@mui/material/FormControl';
import FormHelperText from '@mui/material/FormHelperText';
import InputLabel from '@mui/material/InputLabel';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import { useFormikContext } from 'formik';
import * as yup from 'yup';
import { PermissionLevel, permissionLevelString } from '../../api';

export const PermissionLevelFieldValidator = yup
    .string()
    .label('Select a permission level')
    .required('permission_level is required');

export interface PermissionLevelFieldProps {
    permission_level: PermissionLevel;
}

export const PermissionLevelField = <T,>({
    levels
}: {
    levels?: PermissionLevel[];
}) => {
    const { values, touched, errors, handleChange } = useFormikContext<
        T & PermissionLevelFieldProps
    >();

    return (
        <FormControl fullWidth>
            <InputLabel id="permission_level-label">
                Permission Level
            </InputLabel>
            <Select<PermissionLevel>
                labelId="permission_level-label"
                id="permission_level"
                name={'permission_level'}
                value={values.permission_level}
                onChange={handleChange}
                error={
                    touched.permission_level && Boolean(errors.permission_level)
                }
            >
                {(levels
                    ? levels
                    : [
                          PermissionLevel.User,
                          PermissionLevel.Reserved,
                          PermissionLevel.Editor,
                          PermissionLevel.Moderator,
                          PermissionLevel.Admin
                      ]
                ).map((v) => (
                    <MenuItem key={`time-${v}`} value={v}>
                        {permissionLevelString(v)}
                    </MenuItem>
                ))}
            </Select>
            <FormHelperText>
                {touched.permission_level &&
                    errors.permission_level &&
                    `${errors.permission_level}`}
            </FormHelperText>
        </FormControl>
    );
};
