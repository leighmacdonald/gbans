import { useEffect, useState } from 'react';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import { logErr } from '../../util/errors';

interface FilterTestFieldProps {
    pattern: string;
    is_regex: boolean;
}

export const FilterTestField = <T,>() => {
    const [testString, setTestString] = useState<string>('');
    const [matched, setMatched] = useState(false);
    const [validPattern, setValidPattern] = useState(false);
    const { values } = useFormikContext<T & FilterTestFieldProps>();

    useEffect(() => {
        if (!values.pattern) {
            setValidPattern(false);
            setMatched(false);
            return;
        }
        if (values.is_regex) {
            try {
                const p = new RegExp(values.pattern, 'g');
                setMatched(p.test(testString.toLowerCase()));
                setValidPattern(true);
            } catch (e) {
                setValidPattern(false);
                logErr(e);
            }
        } else {
            setMatched(values.pattern.toLowerCase() == testString.toLowerCase());
            setValidPattern(true);
        }
    }, [values.is_regex, values.pattern, testString]);

    return (
        <Stack>
            <TextField
                id="test-string"
                label="Test String"
                value={testString}
                onChange={(event) => {
                    setTestString(event.target.value);
                }}
            />
            {values.pattern && (
                <Typography variant={'caption'} color={validPattern && matched ? 'success' : 'error'}>
                    {validPattern ? (matched ? 'Matched' : 'No Match') : 'Invalid Pattern'}
                </Typography>
            )}
        </Stack>
    );
};
