import { Languages, ChevronDown, Check } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { cn } from '@/lib/utils';
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from '@/lib/i18n';
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from '@/components/ui/dropdown-menu';

export function LanguageToggle() {
  const { i18n } = useTranslation();

  const currentLanguage = SUPPORTED_LANGUAGES.find(
    (lang) => lang.code === i18n.language
  );

  const handleLanguageChange = (languageCode: SupportedLanguage) => {
    i18n.changeLanguage(languageCode);
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className={cn(
            'flex items-center gap-1.5 rounded-lg px-2.5 py-1.5',
            'transition-all duration-200',
            'border border-border/60 hover:border-border/80',
            'bg-muted/30 hover:bg-muted/50',
            'focus:outline-none focus:ring-2 focus:ring-ring focus:border-primary/50'
          )}
          aria-label={`Select language (Current: ${currentLanguage?.nativeName})`}
          title={currentLanguage?.nativeName}
        >
          <Languages size={16} />
          <ChevronDown
            size={12}
            className="transition-transform duration-200 text-muted-foreground"
          />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-36">
        {SUPPORTED_LANGUAGES.map((option) => {
          const isSelected = option.code === i18n.language;

          return (
            <DropdownMenuItem
              key={option.code}
              onClick={() => handleLanguageChange(option.code)}
              className={cn(
                'flex items-center justify-between gap-2 text-xs',
                isSelected && 'font-medium'
              )}
            >
              <span className="truncate">
                {option.nativeName}
              </span>
              {isSelected && (
                <Check size={12} className="text-primary shrink-0" />
              )}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
