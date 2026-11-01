<script lang="ts" setup>
import type { PropType } from 'vue';

import { computed, ref } from 'vue';

import { IconifyIcon } from '@vben/icons';

import { ElButton, ElInput, ElPopover } from 'element-plus';

defineOptions({ name: 'IconPicker' });

const props = defineProps({
  modelValue: {
    default: '',
    type: String as PropType<string>,
  },
  placeholder: {
    default: '点击选择图标',
    type: String as PropType<string>,
  },
});

const emit = defineEmits<{ (e: 'update:modelValue', v: string): void }>();

const visible = ref(false);
const keyword = ref('');

// 预置常用 lucide 图标, 与侧边栏菜单一致, 由 iconify 运行时渲染
const presets = [
  'activity',
  'award',
  'banknote',
  'bell',
  'bookmark',
  'box',
  'briefcase',
  'bug',
  'building',
  'calendar',
  'camera',
  'car',
  'chart-column',
  'chart-pie',
  'circle-alert',
  'circle-check',
  'circle-help',
  'circle-x',
  'clock',
  'cloud',
  'code',
  'compass',
  'copy',
  'credit-card',
  'crown',
  'database',
  'download',
  'eye',
  'eye-off',
  'file-text',
  'files',
  'filter',
  'flag',
  'flame',
  'folder',
  'folder-open',
  'fingerprint',
  'gauge',
  'gift',
  'git-branch',
  'globe',
  'grid-3x3',
  'heart',
  'home',
  'image',
  'info',
  'key-round',
  'layers',
  'layout-dashboard',
  'layout-grid',
  'link',
  'list',
  'lock',
  'log-out',
  'mail',
  'map-pin',
  'menu',
  'message-square',
  'monitor',
  'music',
  'navigation',
  'package',
  'percent',
  'pencil',
  'plane',
  'plus',
  'power',
  'puzzle',
  'refresh-cw',
  'rocket',
  'route',
  'search',
  'server',
  'settings',
  'settings-2',
  'share-2',
  'shield',
  'shield-check',
  'shopping-cart',
  'sliders-horizontal',
  'smartphone',
  'star',
  'target',
  'terminal',
  'tag',
  'thumbs-up',
  'trash-2',
  'trending-up',
  'truck',
  'upload',
  'user',
  'users',
  'video',
  'wallet',
  'wifi',
  'wrench',
  'zap',
  'workflow',
];

const filtered = computed(() => {
  const kw = keyword.value.trim().toLowerCase();
  if (!kw) return presets;
  return presets.filter((name) => name.includes(kw));
});

function pick(icon: string) {
  emit('update:modelValue', icon);
  visible.value = false;
}

function clear() {
  emit('update:modelValue', '');
}
</script>

<template>
  <ElPopover
    v-model:visible="visible"
    trigger="click"
    :width="344"
    :teleported="true"
  >
    <template #reference>
      <ElInput
        :model-value="modelValue"
        :placeholder="placeholder"
        readonly
        class="w-full cursor-pointer"
      >
        <template #prefix>
          <IconifyIcon
            v-if="modelValue"
            :icon="modelValue"
            class="size-4 text-current"
          />
          <span v-else class="text-xs text-gray-400">无</span>
        </template>
        <template #suffix>
          <span
            v-if="modelValue"
            class="cursor-pointer text-xs text-gray-400 hover:text-red-500"
            @click.stop="clear"
            >清除</span
          >
        </template>
      </ElInput>
    </template>

    <div>
      <ElInput
        v-model="keyword"
        placeholder="搜索图标名称"
        clearable
        size="small"
        class="mb-2"
      />
      <div class="grid max-h-[240px] grid-cols-8 gap-1 overflow-y-auto">
        <button
          v-for="icon in filtered"
          :key="icon"
          :title="`lucide:${icon}`"
          type="button"
          class="flex items-center justify-center rounded border p-1.5 transition-colors hover:bg-gray-100 dark:hover:bg-gray-700"
          :class="
            modelValue === `lucide:${icon}`
              ? 'border-primary text-primary'
              : 'border-transparent'
          "
          @click="pick(`lucide:${icon}`)"
        >
          <IconifyIcon :icon="`lucide:${icon}`" class="size-4" />
        </button>
      </div>
      <div class="mt-2 flex items-center justify-between">
        <span class="text-xs text-gray-400">共 {{ filtered.length }} 个图标</span>
        <ElButton size="small" text type="danger" @click="clear">清除图标</ElButton>
      </div>
    </div>
  </ElPopover>
</template>
