/**
 * i18n 国际化配置
 * 使用 i18next 和 react-i18next 实现多语言支持
 */

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

// 导入翻译文件
import zhCN from '@/locales/zh-CN.json';
import enUS from '@/locales/en-US.json';

// 语言资源
const resources = {
  'zh-CN': {
    translation: zhCN,
  },
  'en-US': {
    translation: enUS,
  },
};

// 支持的语言列表
export const SUPPORTED_LANGUAGES = [
  { code: 'zh-CN', name: '简体中文', nativeName: '简体中文' },
  { code: 'en-US', name: 'English', nativeName: 'English' },
] as const;

export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]['code'];

// 默认语言
export const DEFAULT_LANGUAGE: SupportedLanguage = 'zh-CN';

// localStorage key for language preference
export const LANGUAGE_STORAGE_KEY = 'shepherd_language';

/**
 * 获取存储的语言设置
 */
export function getStoredLanguage(): SupportedLanguage | null {
  const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
  if (stored && SUPPORTED_LANGUAGES.some(lang => lang.code === stored)) {
    return stored as SupportedLanguage;
  }
  return null;
}

/**
 * 存储语言设置
 */
export function setStoredLanguage(language: SupportedLanguage): void {
  localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
}

/**
 * 初始化 i18n
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
