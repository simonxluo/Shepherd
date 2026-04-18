/**
 * i18n configuration.
 * Uses i18next and react-i18next for multi-language support.
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

// Import translation files
import zhCN from '@/locales/zh-CN.json';
import enUS from '@/locales/en-US.json';

// Language resources
const resources = {
  'zh-CN': {
    translation: zhCN,
  },
  'en-US': {
    translation: enUS,
  },
};

// Supported languages
export const SUPPORTED_LANGUAGES = [
  { code: 'zh-CN', name: '简体中文', nativeName: '简体中文' },
  { code: 'en-US', name: 'English', nativeName: 'English' },
] as const;

export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]['code'];

// Default language
export const DEFAULT_LANGUAGE: SupportedLanguage = 'zh-CN';

// localStorage key for language preference
export const LANGUAGE_STORAGE_KEY = 'shepherd_language';

/**
 * Get stored language preference
 */
export function getStoredLanguage(): SupportedLanguage | null {
  const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (stored && SUPPORTED_LANGUAGES.some(lang => lang.code === stored)) {
    return stored as SupportedLanguage;
  }
  return null;
}

/**
 * Store language preference
 */
export function setStoredLanguage(language: SupportedLanguage): void {
  localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
}

/**
 * Initialize i18n
 */
export const i18nReady = i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: DEFAULT_LANGUAGE,
    defaultNS: 'translation',
    supportedLngs: ['zh-CN', 'en-US', 'zh', 'en'],
    interpolation: {
      escapeValue: false,
    },
    react: {
      useSuspense: false,
    },
    debug: import.meta.env.DEV,
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: LANGUAGE_STORAGE_KEY,
      caches: ['localStorage'],
    },
  });

export default i18n;
