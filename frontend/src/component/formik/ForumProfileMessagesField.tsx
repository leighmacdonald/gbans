import Checkbox from '@mui/material/Checkbox';
import FormControlLabel from '@mui/material/FormControlLabel';
import FormGroup from '@mui/material/FormGroup';
import Tooltip from '@mui/material/Tooltip';
import { useFormikContext } from 'formik';

interface ForumProfileMessagesFieldValue {
    forum_profile_messages: boolean;
}

export const ForumProfileMessagesField = () => {
    const { values, handleChange } =
        useFormikContext<ForumProfileMessagesFieldValue>();
    return (
        <FormGroup>
            <Tooltip title={'Allow users to comment on your profile'}>
                <FormControlLabel
                    control={
                        <Checkbox checked={values.forum_profile_messages} />
                    }
                    label="Allow Profile Comments"
                    name={'forum_profile_messages'}
                    onChange={handleChange}
                />
            </Tooltip>
        </FormGroup>
    );
};
