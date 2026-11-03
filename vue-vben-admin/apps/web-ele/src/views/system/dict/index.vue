<script lang="ts" setup>
import type { DictDataItem, DictTypeItem } from '#/api/system/dict';

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
  ElTable,
  ElTableColumn,
  ElTag,
} from 'element-plus';

import {
  createDictDataApi,
  createDictTypeApi,
  deleteDictDataApi,
  deleteDictTypeApi,
  getDictDataListApi,
  getDictTypeListApi,
  updateDictDataApi,
  updateDictDataStatusApi,
  updateDictTypeApi,
  updateDictTypeStatusApi,
} from '#/api/system/dict';
import { invalidateDictCache } from '#/composables/useDict';
import { useTable } from '#/composables/useTable';

defineOptions({ name: 'SystemDict' });

const userStore = useUserStore();
// 写操作仅超级管理员可用(后端服务层同样限制, 管理员只读)
const isSuperAdmin = computed(
  () => userStore.userInfo?.roles?.includes('super_admin') ?? false,
);

const tagColorOptions = [
  { label: '默认', value: '' },
  { label: '成功(success)', value: 'success' },
  { label: '警告(warning)', value: 'warning' },
  { label: '危险(danger)', value: 'danger' },
  { label: '信息(info)', value: 'info' },
];

// ------------------------- 字典类型 -------------------------
// 主表(非分页, total 恒为 0 仅用于满足 useTable 的返回结构)
const selectedType = ref<DictTypeItem>();
const {
  fetchList: fetchTypes,
  handleReset: handleTypeReset,
  handleSearch: handleTypeSearch,
  items: typeItems,
  loading: typeLoading,
  query: typeQuery,
} = useTable<DictTypeItem, { keyword: string; status: number }>({
  async fetch(q) {
    const res = await getDictTypeListApi({
      keyword: q.keyword || undefined,
      status: q.status,
    });
    return { items: res?.items ?? [], total: 0 };
  },
  defaultQuery: { keyword: '', status: -1 },
  afterFetch: () => {
    // 选中类型被删除/不再返回时同步选中态并清空从表(与原 fetchTypes 尾部逻辑一致)
    if (selectedType.value) {
      const current = typeItems.value.find(
        (item) => item.id === selectedType.value?.id,
      );
      if (current) {
        selectedType.value = current;
      } else {
        selectedType.value = undefined;
        dataItems.value = [];
        dataTotal.value = 0;
      }
    }
  },
});

function selectType(row: unknown) {
  selectedType.value = row as DictTypeItem;
  // handleSearch 内部会重置回第 1 页再拉取, 与原 dataQuery.page = 1 + fetchData 一致
  handleDataSearch();
}

// 类型编辑
const typeDialogVisible = ref(false);
const typeSaving = ref(false);
const typeIsEdit = ref(false);
const typeForm = reactive({
  id: 0,
  name: '',
  remark: '',
  status: 1,
  type: '',
});

function openCreateType() {
  typeIsEdit.value = false;
  Object.assign(typeForm, {
    id: 0,
    name: '',
    remark: '',
    status: 1,
    type: '',
  });
  typeDialogVisible.value = true;
}

function openEditType(row: unknown) {
  const item = row as DictTypeItem;
  typeIsEdit.value = true;
  Object.assign(typeForm, {
    id: item.id,
    name: item.name,
    remark: item.remark,
    status: item.status,
    type: item.type,
  });
  typeDialogVisible.value = true;
}

async function handleSaveType() {
  if (!typeForm.name) {
    ElMessage.warning('请输入类型名称');
    return;
  }
  typeSaving.value = true;
  try {
    if (typeIsEdit.value) {
      await updateDictTypeApi({
        id: typeForm.id,
        name: typeForm.name,
        remark: typeForm.remark,
        status: typeForm.status,
      });
      ElMessage.success('更新成功');
    } else {
      if (!typeForm.type) {
        ElMessage.warning('请输入类型编码');
        return;
      }
      await createDictTypeApi({
        name: typeForm.name,
        remark: typeForm.remark,
        status: typeForm.status,
        type: typeForm.type,
      });
      ElMessage.success('创建成功');
    }
    typeDialogVisible.value = false;
    await fetchTypes();
    invalidateDictCache();
  } finally {
    typeSaving.value = false;
  }
}

async function handleDeleteType(row: unknown) {
  const item = row as DictTypeItem;
  const confirmed = await ElMessageBox.confirm(
    `删除字典类型「${item.name}」将同时删除其下所有字典项, 是否继续?`,
    '删除确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  await deleteDictTypeApi(item.id);
  ElMessage.success('已删除');
  await fetchTypes();
  invalidateDictCache();
}

async function handleToggleType(row: unknown) {
  const item = row as DictTypeItem;
  const next = item.status === 1 ? 0 : 1;
  await updateDictTypeStatusApi(item.id, next);
  ElMessage.success(next === 1 ? '已启用' : '已停用');
  await fetchTypes();
  invalidateDictCache();
}

// ------------------------- 字典项 -------------------------
// 从表查询: 依赖选中的字典类型, 未选中时通过 beforeFetch 跳过请求
const {
  fetchList: fetchData,
  handlePageChange,
  handlePageSizeChange: handleSizeChange,
  handleReset: handleDataReset,
  handleSearch: handleDataSearch,
  items: dataItems,
  loading: dataLoading,
  query: dataQuery,
  total: dataTotal,
} = useTable<DictDataItem, { keyword: string; status: number }>({
  async fetch(q) {
    const res = await getDictDataListApi({
      dictType: selectedType.value?.type ?? '',
      keyword: q.keyword || undefined,
      page: q.page,
      pageSize: q.pageSize,
      status: q.status,
    });
    return { items: res?.items ?? [], total: res?.total ?? 0 };
  },
  defaultQuery: { keyword: '', pageSize: 20, status: -1 },
  // 初始未选中类型, 由 selectType 触发首次拉取
  immediate: false,
  beforeFetch: () => {
    // 未选择字典类型时不发请求(与原 fetchData 的守卫一致)
    if (!selectedType.value) {
      return false;
    }
  },
});

const dataDialogVisible = ref(false);
const dataSaving = ref(false);
const dataIsEdit = ref(false);
const dataForm = reactive({
  id: 0,
  label: '',
  remark: '',
  sort: 0,
  status: 1,
  tagColor: '',
  value: '',
});

function openCreateData() {
  if (!selectedType.value) {
    ElMessage.warning('请先选择字典类型');
    return;
  }
  dataIsEdit.value = false;
  Object.assign(dataForm, {
    id: 0,
    label: '',
    remark: '',
    sort: 0,
    status: 1,
    tagColor: '',
    value: '',
  });
  dataDialogVisible.value = true;
}

function openEditData(row: unknown) {
  const item = row as DictDataItem;
  dataIsEdit.value = true;
  Object.assign(dataForm, {
    id: item.id,
    label: item.label,
    remark: item.remark,
    sort: item.sort,
    status: item.status,
    tagColor: item.tagColor,
    value: item.value,
  });
  dataDialogVisible.value = true;
}

async function handleSaveData() {
  if (!selectedType.value) return;
  if (!dataForm.label || !dataForm.value) {
    ElMessage.warning('显示名称与字典值不能为空');
    return;
  }
  dataSaving.value = true;
  try {
    if (dataIsEdit.value) {
      await updateDictDataApi({
        dictType: selectedType.value.type,
        id: dataForm.id,
        label: dataForm.label,
        remark: dataForm.remark,
        sort: dataForm.sort,
        status: dataForm.status,
        tagColor: dataForm.tagColor,
        value: dataForm.value,
      });
      ElMessage.success('更新成功');
    } else {
      await createDictDataApi({
        dictType: selectedType.value.type,
        label: dataForm.label,
        remark: dataForm.remark,
        sort: dataForm.sort,
        status: dataForm.status,
        tagColor: dataForm.tagColor,
        value: dataForm.value,
      });
      ElMessage.success('创建成功');
    }
    dataDialogVisible.value = false;
    await fetchData();
    invalidateDictCache();
  } finally {
    dataSaving.value = false;
  }
}

async function handleDeleteData(row: unknown) {
  const item = row as DictDataItem;
  const confirmed = await ElMessageBox.confirm(
    `确认删除字典项「${item.label}」吗?`,
    '删除确认',
    { type: 'warning' },
  ).catch(() => false);
  if (!confirmed) return;
  await deleteDictDataApi(item.id);
  ElMessage.success('已删除');
  await fetchData();
  invalidateDictCache();
}

async function handleToggleData(row: unknown) {
  const item = row as DictDataItem;
  const next = item.status === 1 ? 0 : 1;
  await updateDictDataStatusApi(item.id, next);
  ElMessage.success(next === 1 ? '已启用' : '已停用');
  await fetchData();
  invalidateDictCache();
}
</script>

<template>
  <div class="p-4">
    <div class="mb-3 rounded-md border border-border bg-card p-4">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <ElInput
          v-model="typeQuery.keyword"
          placeholder="类型编码/名称"
          clearable
          class="w-52"
          @keyup.enter="handleTypeSearch"
        />
        <ElSelect v-model="typeQuery.status" class="w-32">
          <ElOption :value="-1" label="全部状态" />
          <ElOption :value="1" label="启用" />
          <ElOption :value="0" label="停用" />
        </ElSelect>
        <ElButton type="primary" @click="handleTypeSearch">查询</ElButton>
        <ElButton @click="handleTypeReset">重置</ElButton>
        <div class="flex-1"></div>
        <ElButton type="primary" :disabled="!isSuperAdmin" @click="openCreateType">
          新增类型
        </ElButton>
      </div>

      <ElTable
        v-loading="typeLoading"
        :data="typeItems"
        border
        stripe
        highlight-current-row
        row-key="id"
        empty-text="暂无字典类型"
        @row-click="selectType"
      >
        <ElTableColumn prop="type" label="类型编码" min-width="180" />
        <ElTableColumn prop="name" label="类型名称" min-width="150" />
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="remark" label="备注" min-width="150" show-overflow-tooltip />
        <ElTableColumn prop="createdAt" label="创建时间" width="170" />
        <ElTableColumn label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <ElButton
              link
              type="primary"
              :disabled="!isSuperAdmin"
              @click.stop="openEditType(row)"
            >
              编辑
            </ElButton>
            <ElButton
              link
              :type="row.status === 1 ? 'warning' : 'success'"
              :disabled="!isSuperAdmin"
              @click.stop="handleToggleType(row)"
            >
              {{ row.status === 1 ? '停用' : '启用' }}
            </ElButton>
            <ElButton
              link
              type="danger"
              :disabled="!isSuperAdmin"
              @click.stop="handleDeleteType(row)"
            >
              删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>

      <div v-if="!isSuperAdmin" class="mt-2 text-xs text-gray-400">
        当前账号仅可查看字典配置; 字典写操作(新增/编辑/停用/删除)仅超级管理员可执行
      </div>
    </div>

    <div class="rounded-md border border-border bg-card p-4">
      <div class="mb-3 flex flex-wrap items-center gap-2">
        <div class="text-sm font-medium">
          字典项
          <span v-if="selectedType" class="ml-1 text-gray-400">
            ({{ selectedType.name }} / {{ selectedType.type }})
          </span>
          <span v-else class="ml-1 text-gray-400">请先在上方选择字典类型</span>
        </div>
        <div class="flex-1"></div>
        <ElInput
          v-model="dataQuery.keyword"
          placeholder="名称/字典值"
          clearable
          class="w-44"
          :disabled="!selectedType"
          @keyup.enter="handleDataSearch"
        />
        <ElSelect
          v-model="dataQuery.status"
          class="w-32"
          :disabled="!selectedType"
        >
          <ElOption :value="-1" label="全部状态" />
          <ElOption :value="1" label="启用" />
          <ElOption :value="0" label="停用" />
        </ElSelect>
        <ElButton type="primary" :disabled="!selectedType" @click="handleDataSearch">
          查询
        </ElButton>
        <ElButton :disabled="!selectedType" @click="handleDataReset">重置</ElButton>
        <ElButton
          type="primary"
          :disabled="!isSuperAdmin || !selectedType"
          @click="openCreateData"
        >
          新增字典项
        </ElButton>
      </div>

      <ElTable
        v-loading="dataLoading"
        :data="dataItems"
        border
        stripe
        empty-text="请选择左侧字典类型或暂无数据"
      >
        <ElTableColumn prop="label" label="显示名称" min-width="140">
          <template #default="{ row }">
            <ElTag v-if="row.tagColor" :type="row.tagColor" class="mr-1">
              {{ row.label }}
            </ElTag>
            <span v-else>{{ row.label }}</span>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="value" label="字典值" min-width="120" />
        <ElTableColumn prop="sort" label="排序" width="80" />
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn prop="tagColor" label="标签颜色" width="110" />
        <ElTableColumn prop="remark" label="备注" min-width="150" show-overflow-tooltip />
        <ElTableColumn label="操作" width="210" fixed="right">
          <template #default="{ row }">
            <ElButton
              link
              type="primary"
              :disabled="!isSuperAdmin"
              @click="openEditData(row)"
            >
              编辑
            </ElButton>
            <ElButton
              link
              :type="row.status === 1 ? 'warning' : 'success'"
              :disabled="!isSuperAdmin"
              @click="handleToggleData(row)"
            >
              {{ row.status === 1 ? '停用' : '启用' }}
            </ElButton>
            <ElButton
              link
              type="danger"
              :disabled="!isSuperAdmin"
              @click="handleDeleteData(row)"
            >
              删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>

      <div class="mt-4 flex justify-end">
        <ElPagination
          :current-page="dataQuery.page"
          :page-size="dataQuery.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="dataTotal"
          background
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 类型新增/编辑 -->
    <ElDialog
      v-model="typeDialogVisible"
      :title="typeIsEdit ? `编辑字典类型: ${typeForm.name}` : '新增字典类型'"
      width="480px"
      destroy-on-close
    >
      <ElForm label-width="90px">
        <ElFormItem label="类型编码" :required="!typeIsEdit">
          <ElInput
            v-model="typeForm.type"
            :disabled="typeIsEdit"
            placeholder="字母开头, 如 sys_user_status"
            maxlength="64"
          />
        </ElFormItem>
        <ElFormItem label="类型名称" required>
          <ElInput v-model="typeForm.name" maxlength="64" />
        </ElFormItem>
        <ElFormItem label="状态">
          <ElSelect v-model="typeForm.status" class="w-full">
            <ElOption :value="1" label="启用" />
            <ElOption :value="0" label="停用" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput v-model="typeForm.remark" maxlength="255" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="typeDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="typeSaving" @click="handleSaveType">
          保存
        </ElButton>
      </template>
    </ElDialog>

    <!-- 字典项新增/编辑 -->
    <ElDialog
      v-model="dataDialogVisible"
      :title="dataIsEdit ? `编辑字典项: ${dataForm.label}` : '新增字典项'"
      width="480px"
      destroy-on-close
    >
      <ElForm label-width="90px">
        <ElFormItem label="显示名称" required>
          <ElInput v-model="dataForm.label" maxlength="64" />
        </ElFormItem>
        <ElFormItem label="字典值" required>
          <ElInput v-model="dataForm.value" maxlength="64" />
        </ElFormItem>
        <ElFormItem label="排序">
          <ElInputNumber v-model="dataForm.sort" :min="0" class="w-full" />
        </ElFormItem>
        <ElFormItem label="状态">
          <ElSelect v-model="dataForm.status" class="w-full">
            <ElOption :value="1" label="启用" />
            <ElOption :value="0" label="停用" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="标签颜色">
          <ElSelect v-model="dataForm.tagColor" class="w-full">
            <ElOption
              v-for="opt in tagColorOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="备注">
          <ElInput v-model="dataForm.remark" maxlength="255" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="dataDialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="dataSaving" @click="handleSaveData">
          保存
        </ElButton>
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
