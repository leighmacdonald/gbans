import React from 'react';
import FormGroup from '@mui/material/FormGroup';
import FormControlLabel from '@mui/material/FormControlLabel';
import Checkbox from '@mui/material/Checkbox';
import { BaseFormikInputProps } from './BaseFormikInputProps';
import Tooltip from '@mui/material/Tooltip';

interface IncludeFriendsFieldValue {
    include_friends: boolean;
}

export const IncludeFriendsField = ({
    formik,
    isReadOnly
}: BaseFormikInputProps<IncludeFriendsFieldValue>) => {
    return (
        <FormGroup>
            <Tooltip
                title={
                    'Periodically update known friends lists and include them in the ban'
                }
            >
                <FormControlLabel
                    control={
                        <Checkbox
                            checked={formik.values.include_friends}
                            disabled={isReadOnly ?? false}
                        />
                    }
                    label="Include Friends"
                    name={'include_friends'}
                    onChange={formik.handleChange}
                />
            </Tooltip>
        </FormGroup>
    );
};
