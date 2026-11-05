<script lang="ts" setup>
import type { JobItem, JobLogItem, JobPayload } from '#/api/system/job';

import { computed, reactive, ref } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElInputNumber,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElPagination,
  ElSelect,
  ElSwitch,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  changeJobStatusApi,
  createJobApi,
  deleteJobApi,
  listJobLogsApi,
  listJobsApi,
  runJobApi,
  updateJobApi,
} from '#/api/system/job';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'MonitorJob' });

const userStore = useUserStore();
// 写操作仅超级管理员(后端同样强制校验), 管理员只读
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
];

// 后端 invoke_target 为白名单校验, 当前仅注册了 clearlog
const invokeTargetOptions = [
  { label: '清理过期日志 (clearlog)', value: 'clearlog' },
];

// 从 request 错误对象中取出可展示的提示(优先后端业务文案), 不展示错误详情堆栈
function errorMessage(error: unknown): string {
  if (typeof error !== 'object' || error === null) {
    return '操作失败, 请稍后重试';
  }
  const response = (
    error as {
      response?: {
        data?: { error?: unknown; message?: unknown };
        status?: number;
      };
    }
  ).response;
  for (const candidate of [response?.data?.error, response?.data?.message]) {
    if (typeof candidate === 'string' && candidate.trim()) {
      return candidate;
    }
  }
  if (response?.status === 403) {
    return '无权执行该操作';
  }
  const message = (error as { message?: unknown }).message;
  return typeof message === 'string' && message.trim()
    ? message
    : '操作失败, 请稍后重试';
}

// 列表(分页/加载/查询条件由 useTable 统一管理)
const {
  fetchList,
  handlePageChange,
  handlePageSizeChange,
  handleReset,
  handleSearch,
  items,
  loading,
  query,
  total,
} = useTable<JobItem, { name: string; status: number }>({
  async fetch(q) {
    try {
      return await listJobsApi({
        page: q.page,
        pageSize: q.pageSize,
        name: q.name || undefined,
        status: q.status,
      });
    } catch (error) {
      ElMessage.error(errorMessage(error));
      return { items: [], total: 0 };
    }
  },
  defaultQuery: { name: '', status: -1 },
});

const statusUpdatingId = ref<number>();

async function handleStatusChange(
  row: JobItem,
  value: boolean | number | string,
) {
  const next = Number(value) === 1 ? 1 : 0;
  statusUpdatingId.value = row.id;
  try {
    await changeJobStatusApi({ id: row.id, status: next });
    ElMessage.success(next === 1 ? '已启用' : '已停用');
  } catch (error) {
    // 失败回滚开关状态
    row.status = next === 1 ? 0 : 1;
    ElMessage.error(errorMessage(error));
  } finally {
    statusUpdatingId.value = undefined;
  }
}

const defaultForm: JobPayload = {
  name: '',
  jobGroup: 'default',
  invokeTarget: 'clearlog',
  cronExpression: '',
  status: 1,
  concurrencyPolicy: 0,
  timeoutSeconds: 60,
  remark: '',
};

const dialogVisible = ref(false);
const saving = ref(false);
const editingId = ref(0);
const form = reactive<JobPayload>({ ...defaultForm });

function openCreate() {
  editingId.value = 0;
  Object.assign(form, defaultForm);
  dialogVisible.value = true;
}

function openEdit(row: JobItem) {
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name,
    jobGroup: row.jobGroup || 'default',
    invokeTarget: row.invokeTarget || 'clearlog',
    cronExpression: row.cronExpression,
    status: row.status,
    concurrencyPolicy: row.concurrencyPolicy,
    timeoutSeconds: row.timeoutSeconds,
    remark: row.remark,
  });
  dialogVisible.value = true;
}

async function handleSave() {
  if (!form.name.trim()) {
    ElMessage.warning('请输入任务名称');
    return;
  }
  if (!form.invokeTarget) {
    ElMessage.warning('请选择任务标识');
    return;
  }
  if (!form.cronExpression.trim()) {
    ElMessage.warning('请输入 cron 表达式');
    return;
  }
  saving.value = true;
  try {
    if (editingId.value > 0) {
      await updateJobApi({
        ...form,
        id: editingId.value,
        name: form.name.trim(),
        cronExpression: form.cronExpression.trim(),
      });
      ElMessage.success('更新成功');
    } else {
      await createJobApi({
        ...form,
        name: form.name.trim(),
        cronExpression: form.cronExpression.trim(),
      });
      ElMessage.success('创建成功');
    }
    dialogVisible.value = false;
    fetchList();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    saving.value = false;
  }
}

async function handleDelete(row: JobItem) {
  const confirmed = await ElMessageBox.confirm(
    `确定删除任务「${row.name}」吗？删除后不再调度, 执行记录会保留。`,
    '删除确认',
    { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
  ).catch(() => false);
  if (!confirmed) return;
  try {
    await deleteJobApi(row.id);
    ElMessage.success('已删除');
    fetchList();
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}

// 手动执行(异步受理, 不轮询阻塞页面)
async function handleRun(row: JobItem) {
  try {
    await runJobApi(row.id);
    ElMessage.success('已受理，稍后请查看执行记录');
  } catch (error) {
    ElMessage.error(errorMessage(error));
  }
}

// 执行记录弹窗(独立分页状态, 不与主表混用)
const logsVisible = ref(false);
const logsLoading = ref(false);
const logs = ref<JobLogItem[]>([]);
const logsTotal = ref(0);
const logsJobId = ref(0);
const logsJobName = ref('');
const logsQuery = reactive({ page: 1, pageSize: 10, status: -1 });

async function fetchLogs() {
  logsLoading.value = true;
  try {
    const res = await listJobLogsApi({
      page: logsQuery.page,
      pageSize: logsQuery.pageSize,
      jobId: logsJobId.value || undefined,
      status: logsQuery.status,
    });
    logs.value = res?.items ?? [];
    logsTotal.value = res?.total ?? 0;
  } catch (error) {
    ElMessage.error(errorMessage(error));
    logs.value = [];
    logsTotal.value = 0;
  } finally {
    logsLoading.value = false;
  }
}

function openLogs(row: JobItem) {
  logsJobId.value = row.id;
  logsJobName.value = row.name;
  logsQuery.page = 1;
  logsQuery.pageSize = 10;
  logsQuery.status = -1;
  logsVisible.value = true;
  fetchLogs();
}

function handleLogSearch() {
  logsQuery.page = 1;
  fetchLogs();
}

function handleLogPageChange(page: number) {
  logsQuery.page = page;
  fetchLogs();
}

function handleLogPageSizeChange(size: number) {
  logsQuery.pageSize = size;
  logsQuery.page = 1;
  fetchLogs();
}

function orDash(value?: string) {
  return value && value.trim() ? value : '—';
}

function concurrencyText(value: number) {
  return value === 1 ? '允许并发' : '禁止并发';
}

function concurrencyType(value: number): 'info' | 'success' {
  return value === 1 ? 'success' : 'info';
}

function triggerText(value: number) {
  if (value === 1) return '定时';
  if (value === 2) return '手动';
  return '未知';
}

function logStatusText(value: number) {
  return value === 1 ? '成功' : '失败';
}

function logStatusType(value: number): 'danger' | 'success' {
  return value === 1 ? 'success' : 'danger';
}
</script>

<template>
  <div class="p-4">
    <div class="rounded-md border border-border bg-card p-4">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <ElInput
          v-model="query.name"
          placeholder="任务名称"
          clearable
          class="w-44"
          @keyup.enter="handleSearch"
        />
        <ElSelect v-model="query.status" class="w-32">
          <ElOption :value="-1" label="全部状态" />
          <ElOption
            v-for="opt in statusOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </ElSelect>
        <ElButton type="primary" @click="handleSearch">查询</ElButton>
        <ElButton @click="handleReset">重置</ElButton>
        <div class="flex-1"></div>
        <ElButton @click="fetchList">刷新</ElButton>
        <ElButton v-if="isSuperAdmin" type="primary" @click="openCreate">
          新增
        </ElButton>
      </div>

      <ElTable
        v-loading="loading"
        :data="items"
        border
        stripe
        empty-text="暂无定时任务"
      >
        <ElTableColumn prop="name" label="名称" min-width="140" />
        <ElTableColumn prop="jobGroup" label="分组" width="110" />
        <ElTableColumn label="任务标识" min-width="130">
          <template #default="{ row }">
            <span class="font-mono">{{ row.invokeTarget }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="cron" min-width="150">
          <template #default="{ row }">
            <span class="font-mono">{{ orDash(row.cronExpression) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="并发策略" width="110">
          <template #default="{ row }">
            <ElTag :type="concurrencyType(row.concurrencyPolicy)" size="small">
              {{ concurrencyText(row.concurrencyPolicy) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="超时" width="90">
          <template #default="{ row }">
            {{ row.timeoutSeconds }} 秒
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElSwitch
              v-model="row.status"
              :active-value="1"
              :inactive-value="0"
              :disabled="!isSuperAdmin"
              :loading="statusUpdatingId === row.id"
              @change="handleStatusChange(row as JobItem, $event)"
            />
          </template>
        </ElTableColumn>
        <ElTableColumn label="最近执行时间" width="170">
          <template #default="{ row }">
            {{ orDash(row.lastRunAt) }}
          </template>
        </ElTableColumn>
        <ElTableColumn
          label="最近结果"
          min-width="180"
          show-overflow-tooltip
        >
          <template #default="{ row }">
            <span :title="row.lastResult">{{ orDash(row.lastResult) }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn
          :width="isSuperAdmin ? 260 : 100"
          label="操作"
          fixed="right"
        >
          <template #default="{ row }">
            <template v-if="isSuperAdmin">
              <ElButton link type="primary" @click="openEdit(row as JobItem)">
                编辑
              </ElButton>
              <ElButton link type="primary" @click="handleRun(row as JobItem)">
                执行一次
              </ElButton>
            </template>
            <ElButton link type="warning" @click="openLogs(row as JobItem)">
              执行记录
            </ElButton>
            <ElButton
              v-if="isSuperAdmin"
              link
              type="danger"
              @click="handleDelete(row as JobItem)"
            >
              删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>

      <div class="mt-4 flex justify-end">
        <ElPagination
          :current-page="query.page"
          :page-size="query.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          background
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handlePageSizeChange"
        />
      </div>
    </div>

    <!-- 新增/编辑 -->
    <ElDialog
      v-model="dialogVisible"
      :title="editingId > 0 ? '编辑定时任务' : '新增定时任务'"
      width="560px"
      destroy-on-close
    >
      <ElForm label-width="100px">
        <ElFormItem label="名称" required>
          <ElInput
            v-model="form.name"
            placeholder="任务名称, 唯一"
            maxlength="64"
          />
        </ElFormItem>
        <ElFormItem label="分组">
          <ElInput v-model="form.jobGroup" placeholder="默认 default" maxlength="64" />
        </ElFormItem>
        <ElFormItem label="任务标识" required>
          <ElSelect v-model="form.invokeTarget" class="w-full">
            <ElOption
              v-for="opt in invokeTargetOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </ElSelect>
          <div class="w-full text-xs text-gray-400">
            仅支持已注册的内置任务, 未在服务端注册的标识会被拒绝
          </div>
        </ElFormItem>
        <ElFormItem label="cron 表达式" required>
          <ElInput
            v-model="form.cronExpression"
            placeholder="5 段(分 时 日 月 周)或 6 段(秒 分 时 日 月 周), 如 0 0 2 * * *"
            maxlength="64"
          />
          <div class="w-full text-xs text-gray-400">
            6 段示例: 每天 02:00 执行 <span class="font-mono">0 0 2 * * *</span>
          </div>
        </ElFormItem>
        <ElFormItem label="状态">
          <ElSelect v-model="form.status" class="w-full">
            <ElOption
              v-for="opt in statusOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="并发策略">
          <ElSelect v-model="form.concurrencyPolicy" class="w-full">
            <ElOption :value="1" label="允许并发" />
            <ElOption :value="0" label="禁止并发(上次未结束时跳过)" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="超时秒数">
          <ElInputNumber
            v-model="form.timeoutSeconds"
            :min="0"
            :max="86400"
            class="w-full"
          />
          <div class="w-full text-xs text-gray-400">
            0 表示不限制, 最长 86400 秒
          </div>
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput v-model="form.remark" maxlength="255" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="handleSave">
          保存
        </ElButton>
      </template>
    </ElDialog>

    <!-- 执行记录 -->
    <ElDialog
      v-model="logsVisible"
      :title="`执行记录: ${logsJobName}`"
      width="900px"
      destroy-on-close
    >
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <ElSelect
          v-model="logsQuery.status"
          class="w-32"
          @change="handleLogSearch"
        >
          <ElOption :value="-1" label="全部状态" />
          <ElOption :value="1" label="成功" />
          <ElOption :value="0" label="失败" />
        </ElSelect>
        <ElButton @click="fetchLogs">刷新</ElButton>
        <div class="flex-1"></div>
        <span class="text-xs text-gray-400">
          手动执行为异步受理, 记录可能稍后出现
        </span>
      </div>

      <ElTable
        v-loading="logsLoading"
        :data="logs"
        border
        size="small"
        empty-text="暂无执行记录"
      >
        <ElTableColumn label="触发方式" width="100">
          <template #default="{ row }">
            {{ triggerText(row.triggerType) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="开始时间" width="170">
          <template #default="{ row }">
            {{ orDash(row.startedAt) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="结束时间" width="170">
          <template #default="{ row }">
            {{ orDash(row.finishedAt) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="耗时" width="100">
          <template #default="{ row }">
            {{ row.durationMs }} ms
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="logStatusType(row.status)" size="small">
              {{ logStatusText(row.status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="信息" min-width="240" show-overflow-tooltip>
          <template #default="{ row }">
            {{ orDash(row.message) }}
          </template>
        </ElTableColumn>
      </ElTable>

      <div class="mt-4 flex justify-end">
        <ElPagination
          :current-page="logsQuery.page"
          :page-size="logsQuery.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="logsTotal"
          background
          layout="total, sizes, prev, pager, next, jumper"
          small
          @current-change="handleLogPageChange"
          @size-change="handleLogPageSizeChange"
        />
      </div>

      <template #footer>
        <ElButton @click="logsVisible = false">关闭</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
</style>
