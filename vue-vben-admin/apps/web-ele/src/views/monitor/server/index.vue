<script lang="ts" setup>
import type { ServerDep, ServerInfo } from '#/api/monitor';

import { computed, onMounted, onUnmounted, ref } from 'vue';

import { ElButton, ElProgress, ElTable, ElTableColumn, ElTag } from 'element-plus';

import { getServerInfoApi, isForbiddenError } from '#/api/monitor';

defineOptions({ name: 'MonitorServer' });

// 后端未下发间隔时的兜底刷新秒数(与后端默认值一致)
const fallbackRefreshSeconds = 30;

const info = ref<ServerInfo>();
const loading = ref(false);
// 仅超级管理员可访问; 其他角色后端返回 403, 页面展示空状态且不再自动刷新
const denied = ref(false);
const lastUpdated = ref('');
const failed = ref(false);

let timer: ReturnType<typeof setInterval> | undefined;
// 组件卸载后请求可能才返回, 用标记避免再启动定时器
let disposed = false;

function stopTimer() {
  if (timer !== undefined) {
    clearInterval(timer);
    timer = undefined;
  }
}

function startTimer(seconds: number) {
  stopTimer();
  if (disposed) return;
  const delay = seconds > 0 ? seconds : fallbackRefreshSeconds;
  timer = setInterval(() => {
    fetchInfo(true);
  }, delay * 1000);
}

/**
 * 拉取监控数据
 * @param quiet 自动刷新场景不展示 loading, 避免页面闪烁
 */
async function fetchInfo(quiet = false) {
  if (!quiet) loading.value = true;
  try {
    const res = await getServerInfoApi();
    info.value = res ?? undefined;
    denied.value = false;
    failed.value = false;
    lastUpdated.value = new Date().toLocaleTimeString('zh-CN', { hour12: false });
    startTimer(res?.refreshSeconds ?? fallbackRefreshSeconds);
  } catch (error) {
    // 403: 非超管访问, 展示固定提示, 停止自动刷新, 不展示任何后端错误详情
    if (isForbiddenError(error)) {
      denied.value = true;
      info.value = undefined;
      stopTimer();
      return;
    }
    // 其他异常: 保留上一次数据继续按间隔重试; 首次即失败时展示空态
    failed.value = true;
    if (info.value) {
      startTimer(info.value.refreshSeconds ?? fallbackRefreshSeconds);
    } else {
      startTimer(fallbackRefreshSeconds);
    }
  } finally {
    loading.value = false;
  }
}

// 页面隐藏时暂停轮询, 重新可见时立即刷新并恢复
function handleVisibilityChange() {
  if (document.hidden) {
    stopTimer();
    return;
  }
  if (!denied.value) {
    fetchInfo(true);
  }
}

function clampPercent(value: number) {
  if (!Number.isFinite(value) || value <= 0) return 0;
  return Math.min(100, value);
}

function percentText(value: number) {
  if (!Number.isFinite(value)) return '0.0';
  return value.toFixed(1);
}

// 使用率越高颜色越警示
function progressStatus(value: number): '' | 'exception' | 'success' | 'warning' {
  if (!Number.isFinite(value)) return 'success';
  if (value >= 85) return 'exception';
  if (value >= 65) return 'warning';
  return 'success';
}

// 字节数转可读单位
function humanBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let value = bytes;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${index === 0 ? value : value.toFixed(1)} ${units[index]}`;
}

// 进程运行时长格式化: x天x小时x分
const uptimeText = computed(() => {
  const total = Math.max(0, Math.floor(info.value?.uptimeSeconds ?? 0));
  const days = Math.floor(total / 86_400);
  const hours = Math.floor((total % 86_400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  if (days > 0) return `${days}天${hours}小时${minutes}分`;
  if (hours > 0) return `${hours}小时${minutes}分`;
  if (minutes > 0) return `${minutes}分${seconds}秒`;
  return `${seconds}秒`;
});

const refreshSeconds = computed(
  () => info.value?.refreshSeconds ?? fallbackRefreshSeconds,
);

function depText(dep?: ServerDep) {
  if (!dep) return '未知';
  return dep.status === 'ok' ? '正常' : '异常';
}

function depTagType(dep?: ServerDep): 'danger' | 'info' | 'success' {
  if (!dep) return 'info';
  return dep.status === 'ok' ? 'success' : 'danger';
}

onMounted(() => {
  fetchInfo();
  document.addEventListener('visibilitychange', handleVisibilityChange);
});

onUnmounted(() => {
  disposed = true;
  stopTimer();
  document.removeEventListener('visibilitychange', handleVisibilityChange);
});
</script>

<template>
  <div class="p-4">
    <!-- 无权限: 仅展示固定提示, 不展示后端错误详情 -->
    <div
      v-if="denied"
      class="rounded-md border border-border bg-card p-10 text-center text-sm text-gray-400"
    >
      仅超级管理员可查看服务监控
    </div>

    <div v-else v-loading="loading" class="min-h-40">
      <div v-if="!info" class="rounded-md border border-border bg-card p-10 text-center text-sm text-gray-400">
        {{ failed ? '监控数据暂不可用, 已自动重试' : '暂无监控数据' }}
      </div>

      <template v-else>
        <div
          class="mb-3 flex flex-wrap items-center gap-2 text-sm text-gray-500"
        >
          <span>自动刷新: 每 {{ refreshSeconds }} 秒</span>
          <span v-if="lastUpdated" class="text-xs text-gray-400">
            最近刷新 {{ lastUpdated }}
          </span>
          <div class="flex-1"></div>
          <ElButton :loading="loading" @click="fetchInfo()">刷新</ElButton>
        </div>

        <!-- 顶部指标卡片 -->
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <!-- CPU -->
          <div class="rounded-md border border-border bg-card p-4">
            <div class="flex items-center justify-between">
              <span class="text-sm text-gray-500">CPU</span>
              <span class="text-xs text-gray-400">
                {{ info.cpu.cores }} 核
              </span>
            </div>
            <template v-if="info.cpu.available">
              <div class="mt-2 text-2xl font-semibold">
                {{ percentText(info.cpu.usagePercent) }}%
              </div>
              <ElProgress
                class="mt-2"
                :percentage="clampPercent(info.cpu.usagePercent)"
                :status="progressStatus(info.cpu.usagePercent)"
                :stroke-width="10"
                :show-text="false"
              />
            </template>
            <div v-else class="py-6 text-center text-sm text-gray-400">
              暂无数据
            </div>
          </div>

          <!-- 内存 -->
          <div class="rounded-md border border-border bg-card p-4">
            <div class="flex items-center justify-between">
              <span class="text-sm text-gray-500">内存</span>
              <span class="text-xs text-gray-400">
                已用/总量
              </span>
            </div>
            <template v-if="info.memory.available">
              <div class="mt-2 text-2xl font-semibold">
                {{ percentText(info.memory.usagePercent) }}%
              </div>
              <div class="mt-1 text-xs text-gray-400">
                {{ humanBytes(info.memory.used) }} /
                {{ humanBytes(info.memory.total) }}
              </div>
              <ElProgress
                class="mt-2"
                :percentage="clampPercent(info.memory.usagePercent)"
                :status="progressStatus(info.memory.usagePercent)"
                :stroke-width="10"
                :show-text="false"
              />
            </template>
            <div v-else class="py-6 text-center text-sm text-gray-400">
              暂无数据
            </div>
          </div>

          <!-- Go 运行时 -->
          <div class="rounded-md border border-border bg-card p-4">
            <div class="text-sm text-gray-500">Go 运行时</div>
            <div class="mt-3 space-y-1 text-sm">
              <div class="flex justify-between">
                <span class="text-gray-500">Goroutine</span>
                <span class="font-medium">{{ info.runtime.goroutines }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-500">堆内存</span>
                <span class="font-medium">
                  {{ humanBytes(info.runtime.heapAllocBytes) }}
                </span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-500">堆申请</span>
                <span class="font-medium">
                  {{ humanBytes(info.runtime.heapSysBytes) }}
                </span>
              </div>
              <div class="flex justify-between">
                <span class="text-gray-500">GC 次数</span>
                <span class="font-medium">{{ info.runtime.numGC }}</span>
              </div>
            </div>
          </div>

          <!-- 运行时长 -->
          <div class="rounded-md border border-border bg-card p-4">
            <div class="text-sm text-gray-500">进程运行时长</div>
            <div class="mt-2 text-2xl font-semibold">{{ uptimeText }}</div>
            <div class="mt-1 text-xs text-gray-400">
              自服务启动起计算, 不代表主机开机时长
            </div>
          </div>
        </div>

        <!-- 磁盘分区 -->
        <div class="mt-4 rounded-md border border-border bg-card p-4">
          <div class="mb-3 text-sm font-medium">磁盘分区</div>
          <ElTable
            v-if="info.disksAvailable"
            :data="info.disks"
            border
            stripe
            size="small"
            empty-text="暂无磁盘分区数据"
          >
            <ElTableColumn prop="mountpoint" label="挂载点" min-width="160" />
            <ElTableColumn label="总量" width="120">
              <template #default="{ row }">
                {{ humanBytes(row.total) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="已用" width="120">
              <template #default="{ row }">
                {{ humanBytes(row.used) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="使用率" min-width="220">
              <template #default="{ row }">
                <div class="flex items-center gap-2">
                  <ElProgress
                    class="flex-1"
                    :percentage="clampPercent(row.usagePercent)"
                    :status="progressStatus(row.usagePercent)"
                    :stroke-width="10"
                    :show-text="false"
                  />
                  <span class="w-14 text-right text-xs text-gray-500">
                    {{ percentText(row.usagePercent) }}%
                  </span>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
          <div v-else class="py-8 text-center text-sm text-gray-400">
            暂无数据
          </div>
        </div>

        <!-- 依赖健康 -->
        <div class="mt-4 rounded-md border border-border bg-card p-4">
          <div class="mb-3 text-sm font-medium">依赖健康</div>
          <ElTable :data="[info.deps.mysql, info.deps.redis]" border size="small">
            <ElTableColumn label="组件" width="140">
              <template #default="{ $index }">
                {{ $index === 0 ? 'MySQL' : 'Redis' }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="状态" width="110">
              <template #default="{ row }">
                <ElTag :type="depTagType(row as ServerDep)" size="small">
                  {{ depText(row as ServerDep) }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="延迟" min-width="140">
              <template #default="{ row }">
                {{ row.latencyMs }} ms
              </template>
            </ElTableColumn>
          </ElTable>
          <div class="mt-2 text-xs text-gray-400">
            仅展示连通性与延迟, 后端不返回具体错误详情
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
</style>
