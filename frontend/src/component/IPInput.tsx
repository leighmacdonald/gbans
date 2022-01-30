import TextField from '@mui/material/TextField';
import * as React from 'react';
import { ChangeEvent, useState } from 'react';
import FormControl from '@mui/material/FormControl';
import IPCIDR from 'ip-cidr';

export interface IPInputProps {
    id?: string;
    label?: string;
    onCIDRSuccess: (cidr: string) => void;
}

export const IPInput = ({ id, label, onCIDRSuccess }: IPInputProps) => {
    const maxHosts = 256;
    const [input, setInput] = useState<string>('');
    const [error, setError] = useState<boolean>(false);
    const [errorText, setErrorText] = useState<string>('');

    const onChangeInput = (evt: ChangeEvent<HTMLInputElement>) => {
        let addr = evt.target.value;
        setInput(addr);
        if (addr.length == 0) {
            setError(false);
            setErrorText('');
            return;
        }
        if (!addr.includes('/')) {
            addr = addr + '/32';
        } else {
            const v = addr.split('/');
            if (v.length > 1 && parseInt(v[1]) < 24) {
                setError(true);
                setErrorText(`CIDR range too large >${maxHosts}`);
                return;
            }
        }

        if (!IPCIDR.isValidAddress(addr)) {
            setError(true);
            setErrorText('Invalid ip/cidr address');
            return;
        }
        setError(false);
        setErrorText('');
        onCIDRSuccess(addr);
    };
    return (
        <FormControl sx={{ m: 1, minWidth: 200 }}>
            <TextField
                value={input}
                error={error}
                fullWidth
                helperText={error ? (errorText ? errorText : 'error') : ''}
                id={id ?? 'ip'}
                label={label ?? 'IP / CIDR Range'}
                onChange={onChangeInput}
                color={error ? 'error' : input ? 'success' : 'primary'}
            />
        </FormControl>
    );
};
