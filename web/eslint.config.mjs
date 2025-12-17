const eslintConfig = [
  {
    ignores: ["dist/*", "node_modules/*", "**/*.test.ts", "**/*.test.tsx", "**/*.ts", "**/*.tsx"],
  },
  {
    files: ["**/*.js", "**/*.jsx", "**/*.mjs"],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: "module",
    },
  },
];

export default eslintConfig;
