import eslint from '@eslint/js';
import reactRefresh from 'eslint-plugin-react-refresh';
import tseslint from 'typescript-eslint';

export default [
    eslint.configs.recommended,
    ...tseslint.configs.recommended,
    {
        plugins: {
            'react-refresh': reactRefresh
        }
    },
    {
        rules: {
            "@typescript-eslint/no-explicit-any": "error",
            'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
            'no-loop-func': ['error'],
            'no-console': ['error'],
            'no-restricted-imports': [
                'error',
                {
                    patterns: ['@mui/*/*/*']
                }
            ]
        }
    },
    {
        ignores: ['node_modules', 'dist', 'lib']
    }
];
