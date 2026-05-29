import type { ASRPlugin, LoadedModel } from './types';

/**
 * Registry for ASR plugins. Plugins register themselves on import,
 * and the page shell queries the registry to determine which tabs to show.
 */
class ASRPluginRegistry {
  private plugins: ASRPlugin[] = [];

  /**
   * Register an ASR plugin. Plugins with lower `order` appear first in the tab bar.
   */
  register(plugin: ASRPlugin): void {
    // Avoid duplicate registration
    if (this.plugins.some((p) => p.id === plugin.id)) return;
    this.plugins.push(plugin);
    this.plugins.sort((a, b) => (a.order ?? 100) - (b.order ?? 100));
  }

  /**
   * Get the best-matching plugin for a given model.
   * Returns the first plugin (by order) whose `match()` returns true,
   * excluding the 'generic' fallback.
   */
  getPluginForModel(model: LoadedModel): ASRPlugin | null {
    for (const plugin of this.plugins) {
      if (plugin.id === 'generic') continue;
      if (plugin.match(model)) return plugin;
    }
    return this.getGenericPlugin();
  }

  /**
   * Get the generic fallback plugin.
   */
  getGenericPlugin(): ASRPlugin | null {
    return this.plugins.find((p) => p.id === 'generic') ?? null;
  }

  /**
   * Get all registered plugins in display order.
   */
  getAllPlugins(): ASRPlugin[] {
    return [...this.plugins];
  }

  /**
   * Get models matching a specific plugin from a list of loaded ASR models.
   */
  getModelsForPlugin(plugin: ASRPlugin, allASRModels: LoadedModel[]): LoadedModel[] {
    if (plugin.id === 'generic') {
      // Generic matches models that no other plugin claims
      return allASRModels.filter((m) => {
        const specific = this.plugins.find((p) => p.id !== 'generic' && p.match(m));
        return !specific;
      });
    }
    return allASRModels.filter((m) => plugin.match(m));
  }
}

export const asrRegistry = new ASRPluginRegistry();
