<script lang="ts" setup>
import type { NoticeFormData, SystemNoticeItem } from '#/api/system/notice';
import type { DictFallbackItem } from '#/composables/useDict';

import { computed, reactive, ref } from 'vue';

import { useUserStore } from '@vben/stores';

import {
  ElButton,
  ElDatePicker,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElPagination,
  ElRadio,
  ElRadioGroup,
  ElSelect,
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  createNoticeApi,
  deleteNoticeApi,
  getSystemNoticeDetailApi,
  getSystemNoticeListApi,
  publishNoticeApi,
  revokeNoticeApi,
  updateNoticeApi,
} from '#/api/system/notice';
import { getUserListApi } from '#/api/system/user';
import DictTag from '#/components/dict-tag/index.vue';
import { useDict } from '#/composables/useDict';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'SystemNotice' });

// 公告类型/级别字典兜底文案(字典未配置时回退), 展示交给 DictTag
const typeFallback: Record<string, DictFallbackItem> = {
  announcement: { label: '公告', tagColor: 'success' },
  notice: { label: '通知', tagColor: 'info' },
};
const levelFallback: Record<string, DictFallbackItem> = {
  important: { label: '重要', tagColor: 'warning' },
  normal: { label: '普通', tagColor: 'info' },
  urgent: { label: '紧急', tagColor: 'danger' },
};

const { items: typeDictItems } = useDict('notice_type', typeFallback);
const { items: levelDictItems } = useDict('notice_level', levelFallback);

const userStore = useUserStore();
// 管理员只能发布定向公告, 全员公告仅超级管理员可发布(后端同样强校验)
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

// 状态: 0草稿 1已发布 2已撤回; 查询时 -1 表示全部
const statusOptions = [
  { label: '全部状态', value: -1 },
  { label: '草稿', value: 0 },
  { label: '已发布', value: 1 },
  { label: '已撤回', value: 2 },
];

interface NoticeQuery {
  keyword: string;
  level: string;
  status: number;
  timeRange: string[];
  type: string;
}

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
} = useTable<SystemNoticeItem, NoticeQuery>({
  async fetch(q) {
    return getSystemNoticeListApi({
      page: q.page,
      pageSize: q.pageSize,
      endTime: q.timeRange?.[1] || undefined,
      keyword: q.keyword || undefined,
      level: q.level || undefined,
      startTime: q.timeRange?.[0] || undefined,
      status: q.status,
      type: q.type || undefined,
    });
  },
  defaultQuery: {
    keyword: '',
    level: '',
    status: -1,
    timeRange: [] as string[],
    type: '',
  },
});

const statusText = (status: number) => {
  if (status === 0) return '草稿';
  if (status === 1) return '已发布';
  if (status === 2) return '已撤回';
  return '未知';
};

const statusTagType = (status: number) => {
  if (status === 0) return 'info';
  if (status === 1) return 'success';
  if (status === 2) return 'warning';
  return 'info';
};

const scopeText = (scope: number, count: number) =>
  scope === 1 ? '全员' : `指定 ${count} 人`;

// 接收用户选项: 超管可分配全部用户, 管理员仅其子孙(后端按数据权限过滤)
const userOptions = ref<{ id: number; label: string }[]>([]);
const userLoading = ref(false);

async function fetchUserOptions() {
  userLoading.value = true;
  try {
    const res = await getUserListApi({ page: 1, pageSize: 100 });
    userOptions.value = res.items.map((user) => ({
      id: user.id,
      label: user.realName
        ? `${user.realName} (${user.username})`
        : user.username,
    }));
  } catch {
    // 拉取失败时保持空选项, 保存时由后端校验目标用户范围
    userOptions.value = [];
  } finally {
    userLoading.value = false;
  }
}

// 新增/编辑弹窗
const dialogVisible = ref(false);
const saving = ref(false);
const isEdit = ref(false);
// 编辑中的公告 id(新增时为 0)
const editingId = ref(0);
const form = reactive<NoticeFormData>({
  content: '',
  expireAt: '',
  level: 'normal',
  publishAt: '',
  targetScope: 1,
  targetUserIds: [],
  title: '',
  type: 'notice',
});

function resetForm() {
  Object.assign(form, {
    content: '',
    expireAt: '',
    level: 'normal',
    publishAt: '',
    // 管理员默认并只能选择指定用户
    targetScope: isSuperAdmin.value ? 1 : 2,
    targetUserIds: [],
    title: '',
    type: 'notice',
  });
}

function openCreate() {
  isEdit.value = false;
  editingId.value = 0;
  resetForm();
  dialogVisible.value = true;
  if (userOptions.value.length === 0) {
    fetchUserOptions();
  }
}

async function openEdit(row: SystemNoticeItem) {
  const res = await getSystemNoticeDetailApi(row.id);
  const detail = res.item;
  isEdit.value = true;
  editingId.value = detail.id;
  Object.assign(form, {
    content: detail.content,
    expireAt: detail.expireAt,
    level: detail.level,
    publishAt: detail.publishAt,
    // 详情接口单独返回 targetUserIds, 自动回填且允许继续修改
    targetScope: isSuperAdmin.value ? detail.targetScope : 2,
    targetUserIds: [...res.targetUserIds],
    title: detail.title,
    type: detail.type,
  });
  dialogVisible.value = true;
  if (userOptions.value.length === 0) {
    fetchUserOptions();
  }
}

function validateForm(): boolean {
  if (!form.title.trim()) {
    ElMessage.warning('请输入标题');
    return false;
  }
  if (!form.type || !form.level) {
    ElMessage.warning('请选择类型与级别');
    return false;
  }
  if (form.targetScope === 2 && form.targetUserIds.length === 0) {
    ElMessage.warning('定向公告必须指定接收用户');
    return false;
  }
  if (!form.content.trim()) {
    ElMessage.warning('请输入公告内容');
    return false;
  }
  // 时间统一为 YYYY-MM-DD HH:mm:ss, 可直接按字符串比较
  if (
    form.publishAt &&
    form.expireAt &&
    form.expireAt <= form.publishAt
  ) {
    ElMessage.warning('失效时间必须晚于发布时间');
    return false;
  }
  return true;
}

async function handleSave() {
  if (!validateForm()) return;
  saving.value = true;
  try {
    // 日期选择器清空后可能为 null, 统一转为空串提交; 管理员强制定向
    const payload: NoticeFormData = {
      content: form.content,
      expireAt: form.expireAt || '',
      level: form.level,
      publishAt: form.publishAt || '',
      targetScope: isSuperAdmin.value ? form.targetScope : 2,
      targetUserIds:
        form.targetScope === 2 ? [...form.targetUserIds] : [],
      title: form.title,
      type: form.type,
    };
    if (isEdit.value) {
      await updateNoticeApi({ ...payload, id: editingId.value });
      ElMessage.success('更新成功');
    } else {
      await createNoticeApi(payload);
      ElMessage.success('创建成功(草稿)');
    }
    dialogVisible.value = false;
    fetchList();
  } finally {
    saving.value = false;
  }
}

async function handlePublish(row: SystemNoticeItem) {
  const confirmed = await ElMessageBox.confirm(
    row.publishAt
      ? `确认发布公告「${row.title}」吗? 发布时间将更新为当前时间。`
      : `确认发布公告「${row.title}」吗?`,
    '发布确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  await publishNoticeApi({ id: row.id });
  ElMessage.success('已发布');
  fetchList();
}

async function handleRevoke(row: SystemNoticeItem) {
  const confirmed = await ElMessageBox.confirm(
    `撤回后用户端将立即不可见公告「${row.title}」, 是否继续?`,
    '撤回确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  await revokeNoticeApi(row.id);
  ElMessage.success('已撤回');
  fetchList();
}

async function handleDelete(row: SystemNoticeItem) {
  const confirmed = await ElMessageBox.confirm(
    `确认删除公告「${row.title}」吗? 同时会清理接收用户与已读记录。`,
    '删除确认',
    { type: 'error' },
  ).catch(() => false);
  if (!confirmed) return;
  await deleteNoticeApi(row.id);
  ElMessage.success('已删除');
  fetchList();
}

// 详情弹窗
const detailVisible = ref(false);
const detailLoading = ref(false);
const detail = ref<SystemNoticeItem>();

async function openDetail(row: SystemNoticeItem) {
  detailVisible.value = true;
  detailLoading.value = true;
  detail.value = undefined;
  try {
    const res = await getSystemNoticeDetailApi(row.id);
    detail.value = res.item;
  } finally {
    detailLoading.value = false;
  }
}
</script>

<template>
  <div class="p-4">
    <div class="rounded-md border border-border bg-card p-4">
      <div class="mb-4 flex flex-wrap items-center gap-2">
        <ElInput
          v-model="query.keyword"
          placeholder="标题"
          clearable
          class="w-44"
          @keyup.enter="handleSearch"
        />
        <ElSelect v-model="query.type" placeholder="类型" clearable class="w-32">
          <ElOption
            v-for="item in typeDictItems"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
        <ElSelect
          v-model="query.level"
          placeholder="级别"
          clearable
          class="w-32"
        >
          <ElOption
            v-for="item in levelDictItems"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
        <ElSelect v-model="query.status" class="w-32">
          <ElOption
            v-for="opt in statusOptions"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </ElSelect>
        <ElDatePicker
          v-model="query.timeRange"
          type="datetimerange"
          value-format="YYYY-MM-DD HH:mm:ss"
          start-placeholder="发布时间起"
          end-placeholder="发布时间止"
          style="width: 360px"
        />
        <ElButton type="primary" @click="handleSearch">查询</ElButton>
        <ElButton @click="handleReset">重置</ElButton>
        <div class="flex-1"></div>
        <ElButton type="primary" @click="openCreate">新建公告</ElButton>
      </div>

      <ElTable
        v-loading="loading"
        :data="items"
        border
        stripe
        empty-text="暂无公告"
      >
        <ElTableColumn prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <ElTableColumn label="类型" width="100">
          <template #default="{ row }">
            <DictTag
              type="notice_type"
              :value="row.type"
              :fallback="typeFallback"
            />
          </template>
        </ElTableColumn>
        <ElTableColumn label="级别" width="100">
          <template #default="{ row }">
            <DictTag
              type="notice_level"
              :value="row.level"
              :fallback="levelFallback"
            />
          </template>
        </ElTableColumn>
        <ElTableColumn label="状态" width="100">
          <template #default="{ row }">
            <ElTag :type="statusTagType(row.status)">
              {{ statusText(row.status) }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="目标范围" width="110">
          <template #default="{ row }">
            {{ scopeText(row.targetScope, row.targetCount) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="发布时间" width="170">
          <template #default="{ row }">
            <span v-if="row.publishAt">{{ row.publishAt }}</span>
            <span v-else class="text-gray-400">—</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="失效时间" width="170">
          <template #default="{ row }">
            <span v-if="row.expireAt">{{ row.expireAt }}</span>
            <span v-else class="text-gray-400">永不过期</span>
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <ElButton link type="primary" @click="openDetail(row as SystemNoticeItem)">
              详情
            </ElButton>
            <ElButton link type="primary" @click="openEdit(row as SystemNoticeItem)">
              编辑
            </ElButton>
            <ElButton
              v-if="row.status !== 1"
              link
              type="success"
              @click="handlePublish(row as SystemNoticeItem)"
            >
              发布
            </ElButton>
            <ElButton
              v-if="row.status === 1"
              link
              type="warning"
              @click="handleRevoke(row as SystemNoticeItem)"
            >
              撤回
            </ElButton>
            <ElButton link type="danger" @click="handleDelete(row as SystemNoticeItem)">
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

    <!-- 新建/编辑 -->
    <ElDialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑公告' : '新建公告'"
      width="640px"
      destroy-on-close
    >
      <ElForm label-width="90px">
        <ElFormItem label="标题" required>
          <ElInput v-model="form.title" maxlength="128" show-word-limit />
        </ElFormItem>
        <ElFormItem label="类型" required>
          <ElSelect v-model="form.type" class="w-full">
            <ElOption
              v-for="item in typeDictItems"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="级别" required>
          <ElSelect v-model="form.level" class="w-full">
            <ElOption
              v-for="item in levelDictItems"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="目标范围" required>
          <ElRadioGroup v-model="form.targetScope">
            <ElRadio :value="1" :disabled="!isSuperAdmin">全员</ElRadio>
            <ElRadio :value="2">指定用户</ElRadio>
          </ElRadioGroup>
          <div v-if="!isSuperAdmin" class="w-full text-xs text-amber-500">
            管理员不能发布全员公告
          </div>
        </ElFormItem>
        <ElFormItem v-if="form.targetScope === 2" label="接收用户" required>
          <ElSelect
            v-model="form.targetUserIds"
            multiple
            filterable
            clearable
            :loading="userLoading"
            placeholder="从可分配用户中选择"
            class="w-full"
          >
            <ElOption
              v-for="user in userOptions"
              :key="user.id"
              :label="user.label"
              :value="user.id"
            />
          </ElSelect>
          <div class="w-full text-xs text-gray-400">
            候选用户已按数据权限过滤(超管全部, 管理员仅其下属);
            编辑时已自动回填当前接收用户, 可继续修改
          </div>
        </ElFormItem>
        <ElFormItem label="内容" required>
          <ElInput
            v-model="form.content"
            type="textarea"
            :rows="8"
            maxlength="20000"
            placeholder="请输入公告正文, 不支持 HTML"
          />
          <div class="w-full text-xs text-gray-400">
            仅支持纯文本, 不支持 HTML; 换行会原样展示
          </div>
        </ElFormItem>
        <ElFormItem label="发布时间">
          <ElDatePicker
            v-model="form.publishAt"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="留空则发布时取当前时间"
            class="w-full"
          />
        </ElFormItem>
        <ElFormItem label="失效时间">
          <ElDatePicker
            v-model="form.expireAt"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="留空则永不过期"
            class="w-full"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="saving" @click="handleSave">
          保存
        </ElButton>
      </template>
    </ElDialog>

    <!-- 详情 -->
    <ElDialog v-model="detailVisible" title="公告详情" width="640px">
      <div v-loading="detailLoading" class="min-h-20">
        <template v-if="detail">
          <div class="mb-3 flex flex-wrap items-center gap-2">
            <div class="text-base font-medium">{{ detail.title }}</div>
            <DictTag
              type="notice_type"
              :value="detail.type"
              :fallback="typeFallback"
            />
            <DictTag
              type="notice_level"
              :value="detail.level"
              :fallback="levelFallback"
            />
            <ElTag :type="statusTagType(detail.status)">
              {{ statusText(detail.status) }}
            </ElTag>
            <ElTag type="info">{{ scopeText(detail.targetScope, detail.targetCount) }}</ElTag>
          </div>
          <div class="mb-3 text-xs text-gray-400">
            发布时间: {{ detail.publishAt || '未发布' }} · 失效时间:
            {{ detail.expireAt || '永不过期' }}
          </div>
          <div
            class="rounded-md border border-border bg-accent p-3 text-sm leading-6"
            style="white-space: pre-wrap; word-break: break-word"
          >
            {{ detail.content }}
          </div>
        </template>
        <div v-else-if="!detailLoading" class="text-sm text-gray-400">
          暂无数据
        </div>
      </div>
    </ElDialog>
  </div>
</template>

<style scoped>
:deep(.el-table th.el-table__cell),
:deep(.el-table td.el-table__cell) {
  text-align: center;
}
</style>
