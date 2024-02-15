module.exports = {
  root: true,
  env: { browser: true, es2020: true },
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:react-hooks/recommended',
  ],
  ignorePatterns: ['dist', '.eslintrc.cjs'],
  parser: '@typescript-eslint/parser',
  plugins: ['react-refresh', 'no-loops'],
  rules: {
    "@typescript-eslint/no-explicit-any": "warn",
    "no-loops/no-loops": "warn",
    "no-restricted-imports": [
      "error",
      {
        "patterns": [
          "@mui/*/*/*"
        ]
      }
    ],
    'react-refresh/only-export-components': [
      'warn',
      { allowConstantExport: true },
    ],
  },
}
