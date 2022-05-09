import createTheme from '@mui/material/styles/createTheme';
import darkScrollbar from '@mui/material/darkScrollbar';
import { PaletteMode, PaletteOptions } from '@mui/material';

// const colors = {
//     a: '#395c78',
//     b: '#5b7a8c',
//     c: '#768a88',
//     d: '#6b6a65',
//     e: '#34302d',
//     f: '#462d26',
//     g: '#6a4535',
//     h: '#913a1e',
//     i: '#bd3b3b',
//     j: '#9d312f',
//     k: '#f08149',
//     l: '#ef9849',
//     m: '#f5ad87',
//     n: '#f6b98a',
//     o: '#f5e7de',
//     p: '#c1a18a',
//     q: '#dabdab'
// };

export const createThemeByMode = (mode: PaletteMode) => {
    let opts: PaletteOptions = {};
    if (mode == 'light') {
        opts = {
            mode: 'light',
            primary: {
                main: '#bd3b3b'
            },
            secondary: {
                main: '#836312'
            },
            error: {
                main: '#bd3b3b'
            },
            background: {
                default: '#f5e7de',
                paper: '#dabdab'
            },
            success: {
                main: '#5b7a8c'
            }
        };
    } else {
        opts = {
            mode: 'dark',
            primary: {
                main: '#f5e7de'
            },
            secondary: {
                main: '#836312'
            },
            error: {
                main: '#8d0101'
            },
            background: {
                default: '#462d26',
                paper: '#6a4535'
            }
        };
    }
    return createTheme({
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    body: darkScrollbar()
                }
            }
        },
        typography: {
            allVariants: {
                color: mode === 'light' ? '#34302d' : '#f5e7de'
            },
            fontSize: 12,
            h1: {
                fontSize: 36
            },
            h2: {
                fontSize: 32
            },
            h3: {
                fontSize: 28
            },
            h4: {
                fontSize: 24
            },
            h5: {
                fontSize: 20
            },
            h6: {
                fontSize: 16
            }
        },
        palette: opts
    });
};
