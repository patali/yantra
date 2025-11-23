import js from "@eslint/js";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import tsParser from "@typescript-eslint/parser";
import vuePlugin from "eslint-plugin-vue";
import vueParser from "vue-eslint-parser";

export default [
  js.configs.recommended,
  ...vuePlugin.configs["flat/recommended"],
  {
    files: ["**/*.{js,mjs,cjs,ts,tsx,vue}"],
    languageOptions: {
      ecmaVersion: 2020,
      sourceType: "module",
      parser: vueParser,
      parserOptions: {
        ecmaVersion: 2020,
        sourceType: "module",
        parser: tsParser,
        extraFileExtensions: [".vue"]
      },
      globals: {
        Atomics: "readonly",
        SharedArrayBuffer: "readonly",
        console: "readonly",
        process: "readonly",
        FormData: "readonly",
        Blob: "readonly",
        fetch: "readonly",
        globalThis: "readonly",
        window: "readonly",
        document: "readonly",
        navigator: "readonly",
        localStorage: "readonly"
      }
    },
    plugins: {
      "@typescript-eslint": tsPlugin,
      vue: vuePlugin
    },
    rules: {
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          argsIgnorePattern: "^_",
          varsIgnorePattern: "^_",
          caughtErrorsIgnorePattern: "^_"
        }
      ],
      "vue/multi-word-component-names": "off",
      "vue/no-v-html": "warn",
      "comma-spacing": ["error", { before: false, after: true }],
      "no-trailing-spaces": "error",
      "space-infix-ops": "error",
      semi: "error",
      indent: ["error", 2, { SwitchCase: 1 }],
      "keyword-spacing": [
        "error",
        {
          before: true,
          after: true
        }
      ],
      "brace-style": ["error", "1tbs"],
      quotes: ["error", "double", { avoidEscape: true }],
      "space-before-function-paren": [
        "error",
        { anonymous: "never", named: "never", asyncArrow: "always" }
      ],
      "space-before-blocks": ["error", "always"],
      "comma-dangle": ["error", "only-multiline"]
    }
  },
  {
    ignores: ["**/node_modules/**", "**/dist/**", "*.min.js"]
  }
];
