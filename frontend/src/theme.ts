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

export const readableFonts = {
    fontFamily: [
        '"Helvetica Neue"',
        'Helvetica',
        'Roboto',
        'Arial',
        'sans-serif'
    ].join(',')
};

export const createThemeByMode = (_: PaletteMode) => {
    const opts: PaletteOptions = {
        mode: 'light',
        primary: {
            main: '#9d312f'
        },
        secondary: {
            main: '#395c78'
        },
        background: {
            default: '#dabdab',
            paper: '#f5e7de'
        }
    };
    // if (mode == 'light') {
    //     opts = {
    //         mode: 'light',
    //         primary: {
    //             main: '#9d312f'
    //         },
    //         secondary: {
    //             main: '#395c78'
    //         },
    //         background: {
    //             default: '#dabdab',
    //             paper: '#f5e7de'
    //         }
    //     };
    // } else {
    //     opts = {
    //         mode: 'dark',
    //         primary: {
    //             main: '#f5e7de'
    //         },
    //         secondary: {
    //             main: '#836312'
    //         },
    //         error: {
    //             main: '#8d0101'
    //         },
    //         background: {
    //             paper: '#462d26',
    //             default: '#6a4535'
    //         }
    //     };
    // }
    return createTheme({
        components: {
            MuiCssBaseline: {
                styleOverrides: {
                    body: darkScrollbar()
                }
            }
        },
        typography: {
            fontFamily: [
                '"TF2 Build"',
                '"Helvetica Neue"',
                'Helvetica',
                'Roboto',
                'Arial',
                'sans-serif'
            ].join(','),
            // allVariants: {
            //     color: mode === 'light' ? '#34302d' : '#f5e7de'
            // },
            body1: {
                ...readableFonts
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
