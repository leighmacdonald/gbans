import React from 'react';
import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import Tooltip from '@mui/material/Tooltip';
import { BaseFormikInputProps } from './BaseFormikInputProps';

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
