import { apiClient } from '@/lib/api/client';

// ============ Chat Template ============

interface ChatTemplateResponse {
  success: boolean;
  data: {
    modelId: string;
    exists: boolean;
    chatTemplate: string;
  };
}

export async function getModelChatTemplate(modelId: string) {
  const res = await apiClient.get<ChatTemplateResponse>(`/models/${modelId}/chat-template`);
  return res.data;
}

export async function saveModelChatTemplate(modelId: string, chatTemplate: string) {
  await apiClient.post(`/models/${modelId}/chat-template`, { chatTemplate });
}

export async function deleteModelChatTemplate(modelId: string) {
  await apiClient.delete(`/models/${modelId}/chat-template`);
}

export async function getModelDefaultChatTemplate(modelId: string) {
  const res = await apiClient.get<ChatTemplateResponse>(`/models/${modelId}/chat-template/default`);
  return res.data;
}

// ============ Chat Template Kwargs ============

interface KwargsResponse {
  success: boolean;
  data: {
    modelId: string;
    chat_template_kwargs: Record<string, unknown>;
  };
}

export async function getModelKwargs(modelId: string) {
  const res = await apiClient.get<KwargsResponse>(`/models/${modelId}/chat-template-kwargs`);
  return res.data.chat_template_kwargs;
}

export async function saveModelKwargs(modelId: string, kwargs: Record<string, unknown>) {
  await apiClient.post(`/models/${modelId}/chat-template-kwargs`, { chat_template_kwargs: kwargs });
}

export async function deleteModelKwargs(modelId: string) {
  await apiClient.delete(`/models/${modelId}/chat-template-kwargs`);
}

// ============ Tokenize ============

interface TokenizeResponse {
  tokens: number[];
}

export async function tokenizeText(modelId: string, content: string, addSpecial = true, parseSpecial = true) {
  const res = await apiClient.post<TokenizeResponse>(`/models/${modelId}/tokenize`, {
    content,
    add_special: addSpecial,
    parse_special: parseSpecial,
    with_pieces: false,
  });
  return res;
}

interface ApplyTemplateResponse {
  prompt: string;
}

export async function applyTemplate(modelId: string, messages: Array<{ role: string; content: string }>) {
  const res = await apiClient.post<ApplyTemplateResponse>(`/models/${modelId}/apply-template`, {
    messages,
  });
  return res;
}

// ============ Slots ============

export interface SlotInfo {
  id: number;
  is_processing: boolean;
  n_ctx?: number;
  n_past?: number;
  [key: string]: unknown;
}

interface SlotsResponse {
  success: boolean;
  data: {
    slots: SlotInfo[];
  };
}

export async function getModelSlots(modelId: string) {
  const res = await apiClient.get<SlotsResponse>(`/models/${modelId}/slots`);
  return res.data.slots;
}

// ============ Sampling ============

export interface SamplingConfig {
  temperature?: number;
  top_p?: number;
  top_k?: number;
  min_p?: number;
  top_n_sigma?: number;
  presence_penalty?: number;
  repeat_penalty?: number;
  frequency_penalty?: number;
  dry_multiplier?: number;
  dry_base?: number;
  dry_allowed_length?: number;
  dry_penalty_last_n?: number;
  dry_sequence_breakers?: string[];
  seed?: number;
  samplers?: string[];
  force_enable_thinking?: boolean;
  enable_thinking?: boolean;
  [key: string]: unknown;
}

interface SamplingConfigsResponse {
  success: boolean;
  data: {
    configs: Record<string, SamplingConfig>;
  };
}

export async function listSamplingConfigs() {
  const res = await apiClient.get<SamplingConfigsResponse>('/models/sampling/configs');
  return res.data.configs;
}

export async function saveSamplingConfig(samplingConfigName: string, sampling: SamplingConfig) {
  await apiClient.post('/models/sampling/configs', { samplingConfigName, sampling });
}

export async function deleteSamplingConfig(name: string) {
  await apiClient.delete(`/models/sampling/configs/${name}`);
}

interface SamplingSelectionResponse {
  success: boolean;
  data: {
    modelId: string;
    samplingConfigName: string;
  };
}

export async function getModelSamplingSelection(modelId: string) {
  const res = await apiClient.get<SamplingSelectionResponse>(`/models/${modelId}/sampling/selection`);
  return res.data.samplingConfigName;
}

export async function setModelSamplingSelection(modelId: string, samplingConfigName: string) {
  await apiClient.post(`/models/${modelId}/sampling/selection`, { samplingConfigName });
}
