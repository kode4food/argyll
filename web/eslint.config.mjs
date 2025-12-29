const eslintConfig = [
  {
    ignores: [
      "dist/*",
      "node_modules/*",
      "coverage/*",
      "**/*.test.ts",
      "**/*.test.tsx",
      "**/*.ts",
      "**/*.tsx",
    ],
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
