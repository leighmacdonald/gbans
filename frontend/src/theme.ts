import { createTheme, PaletteMode, PaletteOptions, responsiveFontSizes } from '@mui/material';
import darkScrollbar from '@mui/material/darkScrollbar';

export const blu = '#547d8c';
export const red = '#a7584b';

const baseFontSet = ['"Helvetica Neue"', 'Helvetica', 'Roboto', 'Arial', 'sans-serif'];

export const readableFonts = {
    fontFamily: baseFontSet.join(',')
};

export const tf2Fonts = {
    fontFamily: ['"TF2 Build"', ...baseFontSet].join(','),
    letterSpacing: '0.10rem'
};

declare module '@mui/material/styles' {
    export interface Theme {
        mode: string;
    }
    export interface ThemeOptions {
        mode?: string;
    }
}

export const createThemeByMode = (mode: PaletteMode) => {
    const opts: PaletteOptions =
        mode == 'light'
            ? {
                  primary: {
                      main: '#395c78'
                  },
                  secondary: {
                      main: '#9d312f'
                  },
                  background: {
                      default: 'inherit',
                      paper: '#f5e7de'
                  },
                  common: {
                      white: '#f5e7de',
                      black: '#34302d'
                  }
              }
            : {
                  primary: {
                      main: '#9d312f',
                      dark: '#d14441'
                  },
                  secondary: {
                      main: '#395c78'
                  },
                  background: {
                      default: 'inherit',
                      paper: '#3e281f'
                  },
                  common: {
                      white: '#f5e7de',
                      black: '#34302d'
                  },
                  text: {
                      primary: '#f5e7de',
                      secondary: '#e3d6ce'
                  },
                  divider: '#452c22'
              };

    return responsiveFontSizes(
        createTheme({
            components: {
                MuiTableCell: {
                    styleOverrides: {
                        root: {
                            borderLeft: '0px solid #9d312f',
                            borderBottom: 'none',
                            borderRight: '0px solid  #9d312f'
                            // paddingLeft: '6px',
                            // paddingRight: '6px'
                        }
                    }
                },
                MuiTabs: {
                    styleOverrides: {
                        root: {
                            minHeight: 36,
                            height: 36
                        }
                    }
                },
                MuiTab: {
                    styleOverrides: {
                        root: {
                            minHeight: 36,
                            height: 36
                        }
                    }
                },
                MuiCssBaseline: {
                    styleOverrides: {
                        body: darkScrollbar()
                    }
                },
                MuiTextField: {
                    defaultProps: {
                        variant: 'filled'
                    }
                },
                MuiSelect: {
                    defaultProps: {
                        variant: 'filled'
                    }
                },
                MuiButton: {
                    defaultProps: {
                        variant: 'contained'
                    }
                },
                MuiButtonGroup: {
                    defaultProps: {
                        variant: 'contained'
                    }
                },
                MuiDialogActions: {
                    styleOverrides: {
                        root: {
                            paddingTop: 12,
                            paddingLeft: 24,
                            paddingRight: 24,
                            paddingBottom: 24,
                            backgroundColor: opts.background?.paper
                        }
                    }
                }
            },
            typography: {
                fontFamily: ['"Helvetica Neue"', 'Helvetica', 'Roboto', 'Arial', 'sans-serif'].join(','),
                // allVariants: {
                //     color: mode === 'dark' ? '#34302d' : '#f5e7de'
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
            palette: opts,
            mode: mode
        })
    );
};
