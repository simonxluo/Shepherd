import type { BenchmarkParam } from '@/types';

/**
 * Get a unique field name for a param (used as key in maps)
 */
export function getFieldName(param: BenchmarkParam): string {
  return param.fullName;
}

/**
 * Build the benchmark command string from enabled params and their values
 */
export function buildBenchCmd(
  params: BenchmarkParam[],
  enabledMap: Record<string, boolean>,
  valueMap: Record<string, string>
): string {
  const parts: string[] = [];

  for (const p of params) {
    const fieldName = getFieldName(p);
    if (!enabledMap[fieldName]) continue;

    const val = valueMap[fieldName] ?? '';
    const type = (p.type || 'STRING').toUpperCase();

    if (type === 'LOGIC') {
      if (val === '1' || val === 'true') {
        parts.push(p.fullName);
      }
    } else {
      if (val.trim()) {
        parts.push(`${p.fullName} ${val.trim()}`);
      }
    }
  }

  return parts.join(' ');
}

/**
 * Build the -dev argument from selected devices
 */
export function buildDeviceArg(
  devices: string[],
  selectedIndices: number[],
  mainGpu: number
): string {
  const parts: string[] = [];

  if (selectedIndices.length > 0 && selectedIndices.length < devices.length) {
    // Only include selected devices
    const selectedDeviceIds = selectedIndices.map(i => {
      const device = devices[i];
      const colonIdx = device.indexOf(':');
      return colonIdx >= 0 ? device.substring(0, colonIdx).trim() : device.trim();
    });
    parts.push(`-dev ${selectedDeviceIds.join('/')}`);
  } else if (selectedIndices.length > 0) {
    // All devices selected - still include them explicitly
    const allDeviceIds = devices.map(d => {
      const colonIdx = d.indexOf(':');
      return colonIdx >= 0 ? d.substring(0, colonIdx).trim() : d.trim();
    });
    parts.push(`-dev ${allDeviceIds.join('/')}`);
  }

  if (mainGpu > 0) {
    parts.push(`--main-gpu ${mainGpu}`);
  }

  return parts.join(' ');
}

/**
 * Build the complete args array for the benchmark API call
 */
export function buildBenchArgs(
  modelPath: string,
  params: BenchmarkParam[],
  enabledMap: Record<string, boolean>,
  valueMap: Record<string, string>,
  devices: string[],
  selectedDeviceIndices: number[],
  mainGpu: number
): string[] {
  const args: string[] = ['-m', modelPath];

  for (const p of params) {
    const fieldName = getFieldName(p);
    if (!enabledMap[fieldName]) continue;

    const val = valueMap[fieldName] ?? '';
    const type = (p.type || 'STRING').toUpperCase();

    if (type === 'LOGIC') {
      if (val === '1' || val === 'true') {
        args.push(p.fullName);
      }
    } else {
      if (val.trim()) {
        args.push(p.fullName, val.trim());
      }
    }
  }

  // Add device args
  if (selectedDeviceIndices.length > 0) {
    const selectedDeviceIds = selectedDeviceIndices.map(i => {
      const device = devices[i];
      const colonIdx = device.indexOf(':');
      return colonIdx >= 0 ? device.substring(0, colonIdx).trim() : device.trim();
    });
    args.push('-dev', selectedDeviceIds.join('/'));
  }

  if (mainGpu > 0) {
    args.push('--main-gpu', String(mainGpu));
  }

  return args;
}
