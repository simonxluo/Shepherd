import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { 
  User, 
  Mail, 
  Shield, 
  Calendar, 
  Camera,
  Edit3,
  Check,
  X,
  AtSign,
  BadgeCheck,
  Sparkles
} from 'lucide-react';
import { useUserStore } from '@/stores/userStore';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { cn } from '@/lib/utils';

export function UserProfileDialog() {
  const { t } = useTranslation();
  const { user, showProfileDialog, setShowProfileDialog, updateProfile } = useUserStore();
  const [isEditing, setIsEditing] = useState(false);
  const [formData, setFormData] = useState({
    displayName: user?.displayName || '',
    email: user?.email || '',
  });
  const [isHoveringAvatar, setIsHoveringAvatar] = useState(false);

  if (!user) return null;

  const handleSave = () => {
    updateProfile({
      displayName: formData.displayName,
      email: formData.email,
    });
    setIsEditing(false);
  };

  const handleCancel = () => {
    setFormData({
      displayName: user?.displayName || '',
      email: user?.email || '',
    });
    setIsEditing(false);
  };

  const getAvatarUrl = () => {
    if (user.avatar) return user.avatar;
    return null;
  };

  const getInitials = () => {
    const name = user.displayName || user.username;
    return name.charAt(0).toUpperCase();
  };

  return (
    <Dialog open={showProfileDialog} onOpenChange={setShowProfileDialog}>
      <DialogContent className="sm:max-w-lg p-0 overflow-hidden">
        {/* Header Background */}
        <div className="h-32 bg-gradient-to-r from-primary/80 via-primary to-primary/80 relative overflow-hidden">
          <div className="absolute inset-0 opacity-20">
            <div className="absolute top-0 left-0 w-40 h-40 bg-white/20 rounded-full -translate-x-1/2 -translate-y-1/2" />
            <div className="absolute bottom-0 right-0 w-32 h-32 bg-white/10 rounded-full translate-x-1/3 translate-y-1/3" />
          </div>
        </div>

        <div className="px-6 pb-6">
          {/* Avatar Section */}
          <div className="relative -mt-16 mb-6 flex justify-center">
            <div 
              className="relative group"
              onMouseEnter={() => setIsHoveringAvatar(true)}
              onMouseLeave={() => setIsHoveringAvatar(false)}
            >
              <div className={cn(
                'w-32 h-32 rounded-full border-4 border-background shadow-xl overflow-hidden transition-transform duration-300',
                isHoveringAvatar && !isEditing ? 'scale-105' : ''
              )}>
                {getAvatarUrl() ? (
                  <img
                    src={getAvatarUrl()!}
                    alt={user.displayName}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="w-full h-full bg-gradient-to-br from-primary to-primary/70 flex items-center justify-center">
                    <span className="text-4xl font-bold text-primary-foreground">
                      {getInitials()}
                    </span>
                  </div>
                )}
              </div>

              {/* Camera Button */}
              {!isEditing && (
                <button 
                  className={cn(
                    'absolute bottom-1 right-1 p-2.5 rounded-full bg-primary text-primary-foreground shadow-lg transition-all duration-200',
                    isHoveringAvatar ? 'opacity-100 scale-100' : 'opacity-0 scale-90'
                  )}
                >
                  <Camera className="w-4 h-4" />
                </button>
              )}

              {/* Verified Badge */}
              {user.role === 'admin' && (
                <div className="absolute -bottom-1 -right-1 p-1.5 rounded-full bg-background">
                  <BadgeCheck className="w-6 h-6 text-blue-500" />
                </div>
              )}
            </div>
          </div>

          {/* User Info */}
          <div className="text-center mb-8">
            <h2 className="text-2xl font-bold flex items-center justify-center gap-2">
              {user.displayName || user.username}
              <Badge 
                variant={user.role === 'admin' ? 'default' : 'secondary'}
                className="text-xs"
              >
                {user.role === 'admin' ? (
                  <>
                    <Shield className="w-3 h-3 mr-1" />
                    {t('user.admin')}
                  </>
                ) : (
                  t('user.user')
                )}
              </Badge>
            </h2>
            
            {user.email && (
              <p className="text-muted-foreground flex items-center justify-center gap-1 mt-1">
                <Mail className="w-4 h-4" />
                {user.email}
              </p>
            )}
          </div>

          {/* Profile Form */}
          <Card className="border-none shadow-sm">
            <CardContent className="p-0">
              <div className="divide-y">
                {/* Username Field */}
                <ProfileField
                  icon={AtSign}
                  label={t('user.username')}
                  value={user.username}
                  disabled
                />

                {/* Display Name Field */}
                <ProfileField
                  icon={User}
                  label={t('user.displayName')}
                  value={isEditing ? formData.displayName : (user.displayName || user.username)}
                  isEditing={isEditing}
                  onChange={(value) => setFormData({ ...formData, displayName: value })}
                  placeholder={t('user.displayNamePlaceholder')}
                />

                {/* Email Field */}
                <ProfileField
                  icon={Mail}
                  label={t('user.email')}
                  value={isEditing ? formData.email : (user.email || '')}
                  isEditing={isEditing}
                  onChange={(value) => setFormData({ ...formData, email: value })}
                  placeholder={t('user.emailPlaceholder')}
                  type="email"
                />

                {/* Role Field */}
                <ProfileField
                  icon={Shield}
                  label={t('user.role')}
                  value={user.role === 'admin' ? t('user.admin') : t('user.user')}
                  disabled
                  badge={user.role === 'admin' ? 'admin' : 'user'}
                />

                {/* Joined Date Field */}
                {user.createdAt && (
                  <ProfileField
                    icon={Calendar}
                    label={t('user.joinedAt')}
                    value={new Date(user.createdAt).toLocaleDateString(undefined, {
                      year: 'numeric',
                      month: 'long',
                      day: 'numeric'
                    })}
                    disabled
                  />
                )}
              </div>
            </CardContent>
          </Card>

          {/* Action Buttons */}
          <div className="flex gap-3 mt-6">
            {isEditing ? (
              <>
                <Button 
                  variant="outline" 
                  className="flex-1 gap-2"
                  onClick={handleCancel}
                >
                  <X className="w-4 h-4" />
                  {t('common.cancel')}
                </Button>
                <Button 
                  className="flex-1 gap-2"
                  onClick={handleSave}
                >
                  <Check className="w-4 h-4" />
                  {t('common.save')}
                </Button>
              </>
            ) : (
              <Button 
                className="w-full gap-2"
                onClick={() => setIsEditing(true)}
              >
                <Edit3 className="w-4 h-4" />
                {t('common.editProfile')}
              </Button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

interface ProfileFieldProps {
  icon: React.ElementType;
  label: string;
  value: string;
  isEditing?: boolean;
  onChange?: (value: string) => void;
  placeholder?: string;
  type?: string;
  disabled?: boolean;
  badge?: 'admin' | 'user';
}

function ProfileField({
  icon: Icon,
  label,
  value,
  isEditing,
  onChange,
  placeholder,
  type = 'text',
  disabled,
  badge
}: ProfileFieldProps) {
  return (
    <div className="flex items-center gap-4 p-4">
      <div className={cn(
        'p-2.5 rounded-xl',
        disabled ? 'bg-muted' : 'bg-primary/10'
      )}>
        <Icon className={cn(
          'w-5 h-5',
          disabled ? 'text-muted-foreground' : 'text-primary'
        )} />
      </div>

      <div className="flex-1 min-w-0">
        <p className="text-xs text-muted-foreground font-medium uppercase tracking-wider">
          {label}
        </p>
        
        {isEditing && !disabled ? (
          <input
            type={type}
            value={value}
            onChange={(e) => onChange?.(e.target.value)}
            placeholder={placeholder}
            className="w-full mt-1 px-0 py-1 bg-transparent border-0 border-b-2 border-primary/30 focus:border-primary focus:outline-none transition-colors"
            autoFocus
          />
        ) : (
          <div className="flex items-center gap-2 mt-0.5">
            <p className={cn(
              'font-medium truncate',
              disabled && 'text-muted-foreground'
            )}>
              {value || <span className="text-muted-foreground italic">{placeholder || '-'}</span>}
            </p>
            
            {badge && (
              <Badge 
                variant={badge === 'admin' ? 'default' : 'secondary'}
                className="text-xs"
              >
                {badge === 'admin' ? (
                  <>
                    <Sparkles className="w-3 h-3 mr-1" />
                    Admin
                  </>
                ) : 'User'}
              </Badge>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
