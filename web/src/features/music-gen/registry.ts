import type { MusicPlugin, LoadedModel } from './types';

/**
 * Registry for Music generation plugins. Plugins register themselves on import,
 * and the page shell queries the registry to determine which tabs to show.
 */
class MusicPluginRegistry {
  private plugins: MusicPlugin[] = [];

  /**
   * Register a Music plugin. Plugins with lower `order` appear first in the tab bar.
   */
  register(plugin: MusicPlugin): void {
    if (this.plugins.some((p) => p.id === plugin.id)) return;
    this.plugins.push(plugin);
    this.plugins.sort((a, b) => (a.order ?? 100) - (b.order ?? 100));
  }

  /**
   * Get the best-matching plugin for a given model.
   * Returns the first plugin (by order) whose `match()` returns true,
   * excluding the 'generic' fallback.
   */
  getPluginForModel(model: LoadedModel): MusicPlugin | null {
    for (const plugin of this.plugins) {
      if (plugin.id === 'generic') continue;
      if (plugin.match(model)) return plugin;
    }
    return this.getGenericPlugin();
  }

  /**
   * Get the generic fallback plugin.
   */
  getGenericPlugin(): MusicPlugin | null {
    return this.plugins.find((p) => p.id === 'generic') ?? null;
  }

  /**
   * Get all registered plugins in display order.
   */
  getAllPlugins(): MusicPlugin[] {
    return [...this.plugins];
  }

  /**
   * Get models matching a specific plugin from a list of loaded music models.
   */
  getModelsForPlugin(plugin: MusicPlugin, allMusicModels: LoadedModel[]): LoadedModel[] {
    if (plugin.id === 'generic') {
      // Generic matches models that no other plugin claims
      return allMusicModels.filter((m) => {
        const specific = this.plugins.find((p) => p.id !== 'generic' && p.match(m));
        return !specific;
      });
    }
    return allMusicModels.filter((m) => plugin.match(m));
  }
}

export const musicRegistry = new MusicPluginRegistry();
