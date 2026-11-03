import { requestClient } from '#/api/request';
import { downloadExportApi } from '#/api/system/export';

export interface ImportRowError {
  /** Excel 中的行号(含表头) */
  row: number;
  reason: string;
  username: string;
}

export interface ImportResult {
  atomic: boolean;
  errors: ImportRowError[];
  failed: number;
  fileName: string;
  success: number;
  total: number;
}

/**
 * 下载用户导入模板(超管 7 列含最大设备数, 管理员 6 列)
 */
export async function downloadUserImportTemplateApi() {
  await downloadExportApi('/user/import/template', { format: 'xlsx' });
}

/**
 * 上传 Excel 导入用户
 * @param file 选择的上传文件
 * @param atomic true 时任一行失败则全量回滚; 默认 false 部分成功
 */
export async function importUsersApi(file: File, atomic = false) {
  return requestClient.upload<ImportResult>('/user/import', { file, atomic });
}
