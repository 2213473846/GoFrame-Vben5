import { requestClient } from '#/api/request';

interface DownloadOptions {
  params?: Record<string, any>;
  /** 导出格式 */
  format?: 'csv' | 'xlsx';
}

function parseFilename(disposition: string): string {
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8Match?.[1]) {
    return decodeURIComponent(utf8Match[1]);
  }
  const plainMatch = disposition.match(/filename="?([^";]+)"?/i);
  return plainMatch?.[1] || 'export.xlsx';
}

/**
 * 下载列表导出文件(带当前筛选条件)
 * 后端直接返回文件流, 这里通过 blob + a[download] 触发浏览器下载
 */
export async function downloadExportApi(
  url: string,
  options: DownloadOptions = {},
) {
  const response = (await requestClient.download(url, {
    params: { ...options.params, format: options.format ?? 'xlsx' },
    responseReturn: 'raw',
  })) as any;
  const disposition = response?.headers?.['content-disposition'] ?? '';
  const filename = parseFilename(disposition);
  const blob = response?.data as Blob;
  const link = document.createElement('a');
  const objectUrl = URL.createObjectURL(blob);
  link.href = objectUrl;
  link.download = filename;
  document.body.append(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(objectUrl);
}
