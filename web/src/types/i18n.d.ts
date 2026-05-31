import 'i18next';
import type zhCN from '@/locales/zh-CN.json';

declare module 'i18next' {
  interface CustomTypeOptions {
    resources: {
      translation: typeof zhCN;
    };
    defaultNS: 'translation';
  }
}
