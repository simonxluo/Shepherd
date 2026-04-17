import { useState, useEffect } from 'react';
import { ChevronRight, Folder, File, Home, HardDrive, Loader2 } from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { filesystemApi, type DirectoryItem } from '@/lib/api/filesystem';
import { cn } from '@/lib/utils';

interface DirectoryBrowserProps {
  open: boolean;
  initialPath?: string;
  onSelect: (path: string) => void;
  onClose: () => void;
}

export function DirectoryBrowser({
  open,
  initialPath,
  onSelect,
  onClose,
}: DirectoryBrowserProps) {
  const [currentPath, setCurrentPath] = useState(initialPath || '');
  const [parentPath, setParentPath] = useState('');
  const [folders, setFolders] = useState<DirectoryItem[]>([]);
  const [files, setFiles] = useState<DirectoryItem[]>([]);
  const [roots, setRoots] = useState<DirectoryItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [selectedPath, setSelectedPath] = useState('');

  // 加载目录内容
  const loadDirectory = async (path: string) => {
    setIsLoading(true);
    try {
      const response = await filesystemApi.listDirectory(path);
      if (response.success && response.data) {
        setCurrentPath(response.data.currentPath);
        setParentPath(response.data.parentPath);
        setFolders(response.data.folders || []);
        setFiles(response.data.files || []);
        if (response.data.roots) {
          setRoots(response.data.roots);
        }
      }
    } catch (error) {
      console.error('加载目录失败:', error);
    } finally {
      setIsLoading(false);
    }
  };

  // 初始化加载
  useEffect(() => {
    if (open) {
      loadDirectory(initialPath || '');
    }
  }, [open]);

  // 选择文件夹
  const handleSelectFolder = (folder: DirectoryItem) => {
    setSelectedPath(folder.path);
  };

  // 双击进入文件夹
  const handleFolderDoubleClick = (folder: DirectoryItem) => {
    loadDirectory(folder.path);
  };

  // 向上导航
  const handleGoUp = () => {
    if (parentPath && parentPath !== currentPath) {
      loadDirectory(parentPath);
    }
  };

  // 返回根目录
  const handleGoRoots = () => {
    loadDirectory('');
  };

  // 确认选择
  const handleConfirm = () => {
    if (selectedPath) {
      onSelect(selectedPath);
      onClose();
    } else if (currentPath) {
      onSelect(currentPath);
      onClose();
    }
  };

  // 格式化路径显示
  const formatPath = (path: string) => {
    if (!path) return '根目录';
    if (path === '/') return '根目录';
    // 缩短过长的路径
    if (path.length > 50) {
      const parts = path.split('/');
      return '.../' + parts.slice(-2).join('/');
    }
    return path;
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[700px] max-h-[80vh] flex flex-col p-0">
        <DialogHeader className="px-6 pt-6 pb-4 border-b">
          <DialogTitle className="text-base flex items-center gap-2">
            <Folder className="w-5 h-5" />
            选择目录
          </DialogTitle>
        </DialogHeader>

        <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
          {/* 导航栏 */}
          <div className="px-6 py-3 border-b flex items-center gap-2 bg-muted/30">
            <Button
              variant="outline"
              size="sm"
              onClick={handleGoRoots}
              className="h-7 px-2 text-xs"
            >
              <HardDrive className="w-3 h-3 mr-1" />
              根目录
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleGoUp}
              disabled={!parentPath || parentPath === currentPath}
              className="h-7 px-2 text-xs"
            >
              向上
            </Button>
            <div className="flex-1 min-w-0">
              <div className="text-xs font-mono text-muted-foreground truncate px-2 py-1 bg-background rounded">
                {formatPath(currentPath)}
              </div>
            </div>
          </div>

          {/* 加载状态 */}
          {isLoading && (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-muted-foreground" />
              <span className="ml-2 text-sm text-muted-foreground">加载中...</span>
            </div>
          )}

          {/* 内容区域 */}
          {!isLoading && (
            <div className="flex-1 flex overflow-hidden">
              {/* 文件夹面板 */}
              <div className="flex-1 border-r overflow-auto p-4">
                <div className="flex items-center gap-2 mb-3 pb-2 border-b text-sm font-medium text-muted-foreground">
                  <Folder className="w-4 h-4" />
                  文件夹 ({folders.length})
                </div>
                {folders.length === 0 ? (
                  <div className="text-center py-8 text-xs text-muted-foreground">
                    空目录
                  </div>
                ) : (
                  <div className="space-y-1">
                    {folders.map((folder) => (
                      <div
                        key={folder.path}
                        onClick={() => handleSelectFolder(folder)}
                        onDoubleClick={() => handleFolderDoubleClick(folder)}
                        className={cn(
                          "flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer transition-colors text-sm",
                          "hover:bg-accent",
                          selectedPath === folder.path && "bg-accent"
                        )}
                      >
                        <Folder className="w-4 h-4 text-blue-500 flex-shrink-0" />
                        <span className="truncate flex-1">{folder.name}</span>
                        <ChevronRight className="w-3 h-3 text-muted-foreground flex-shrink-0" />
                      </div>
                    ))}
                  </div>
                )}
              </div>

              {/* 文件面板 */}
              <div className="flex-1 overflow-auto p-4">
                <div className="flex items-center gap-2 mb-3 pb-2 border-b text-sm font-medium text-muted-foreground">
                  <File className="w-4 h-4" />
                  文件 ({files.length})
                </div>
                {files.length === 0 ? (
                  <div className="text-center py-8 text-xs text-muted-foreground">
                    无文件
                  </div>
                ) : (
                  <div className="space-y-1">
                    {files.slice(0, 50).map((file) => (
                      <div
                        key={file.path}
                        className="flex items-center gap-2 px-2 py-1.5 rounded text-sm text-muted-foreground"
                      >
                        <File className="w-4 h-4 flex-shrink-0" />
                        <span className="truncate flex-1">{file.name}</span>
                        <span className="text-xs">
                          {file.size !== undefined && file.size > 1024 * 1024 * 1024
                            ? `${(file.size / (1024 * 1024 * 1024)).toFixed(1)} GB`
                            : file.size !== undefined && file.size > 1024 * 1024
                            ? `${(file.size / (1024 * 1024)).toFixed(1)} MB`
                            : file.size !== undefined && file.size > 1024
                            ? `${(file.size / 1024).toFixed(1)} KB`
                            : file.size !== undefined
                            ? `${file.size} B`
                            : ''}
                        </span>
                      </div>
                    ))}
                    {files.length > 50 && (
                      <div className="text-center py-2 text-xs text-muted-foreground">
                        还有 {files.length - 50} 个文件...
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* 根目录列表 */}
          {!isLoading && !currentPath && roots.length > 0 && (
            <div className="flex-1 overflow-auto p-4">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {roots.map((root) => (
                  <button
                    key={root.path}
                    onClick={() => loadDirectory(root.path)}
                    className="flex items-center gap-3 p-3 rounded-lg border hover:bg-accent transition-colors text-left"
                  >
                    <HardDrive className="w-5 h-5 text-muted-foreground" />
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium truncate">{root.name}</div>
                      <div className="text-xs text-muted-foreground truncate">{root.path}</div>
                    </div>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* 底部按钮 */}
        <div className="px-6 py-4 border-t flex items-center justify-end gap-2 bg-muted/30">
          <Button
            variant="outline"
            size="sm"
            onClick={onClose}
            className="h-8 px-3 text-xs"
          >
            取消
          </Button>
          <Button
            size="sm"
            onClick={handleConfirm}
            disabled={!selectedPath && !currentPath}
            className="h-8 px-3 text-xs"
          >
            选择
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
