import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import Tooltip from '@mui/material/Tooltip';
import { useFormikContext } from 'formik';

interface IncludeFriendsFieldValue {
    include_friends: boolean;
}

export const IncludeFriendsField = () => {
    const { values, handleChange } = useFormikContext<IncludeFriendsFieldValue>();
    return (
        <FormGroup>
            <Tooltip title={'Periodically update known friends lists and include them in the ban'}>
                <FormControlLabel
                    control={<Checkbox checked={values.include_friends} />}
                    label="Include Friends"
                    name={'include_friends'}
                    onChange={handleChange}
                />
            </Tooltip>
        </FormGroup>
    );
};
