<template>
  <div
    class="flex flex-col md:flex-row md:justify-between md:items-center gap-2 md:gap-4"
  >
    <div class="flex space-x-4 flex-1">
      <div class="py-2 text-sm font-medium text-control">
        {{ $t("common.sql") }}
        <button
          v-if="!hasFeature('bb.feature.sql-review')"
          type="button"
          class="ml-1 btn-small py-0.5 inline-flex items-center text-accent"
          @click.prevent="state.showFeatureModal = true"
        >
          🎈{{ $t("sql-review.unlock-full-feature") }}
        </button>
      </div>
    </div>
  </div>
  <div class="w-full">
    <div
      v-if="sdlState.loading"
      class="h-20 flex flex-col items-center justify-center"
    >
      <BBSpin />
    </div>
    <template v-else>
      <NTabs
        v-if="sdlState.detail"
        v-model:value="state.tab"
        pane-style="padding-top: 0.25rem"
      >
        <NTabPane name="diff" :tab="$t('issue.sdl.schema-change')">
          <CodeDiff
            :old-string="sdlState.detail.previousSDL"
            :new-string="sdlState.detail.expectedSDL"
            output-format="side-by-side"
            data-label="bb-change-detail-code-diff-block"
        /></NTabPane>
        <NTabPane
          name="statement"
          :tab="$t('issue.sdl.generated-ddl-statements')"
        >
          <MonacoEditor
            ref="editorRef"
            class="w-full border h-auto max-h-[360px]"
            data-label="bb-issue-sql-editor"
            :value="sdlState.detail.diffDDL"
            :readonly="true"
            :auto-focus="false"
            language="sql"
            @ready="handleMonacoEditorReady"
          />
        </NTabPane>
        <NTabPane name="schema" :tab="$t('issue.sdl.full-schema')">
          <MonacoEditor
            ref="editorRef"
            class="w-full border h-auto max-h-[360px]"
            data-label="bb-issue-sql-editor"
            :value="sdlState.detail.expectedSDL"
            :readonly="true"
            :auto-focus="false"
            :advices="markers"
            language="sql"
            @ready="handleMonacoEditorReady"
          />
        </NTabPane>
      </NTabs>
    </template>
  </div>
  <FeatureModal
    v-if="state.showFeatureModal"
    feature="bb.feature.sql-review"
    @cancel="state.showFeatureModal = false"
  />
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { reactive, watch, computed, ref } from "vue";
import { CodeDiff } from "v-code-diff";

import { hasFeature, useDatabaseStore, useInstanceStore } from "@/store";
import { useIssueLogic } from "./logic";
import { Task, TaskDatabaseSchemaUpdateSDLPayload, TaskId } from "@/types";
import MonacoEditor from "../MonacoEditor";
import axios from "axios";
import { useSQLAdviceMarkers } from "./logic/useSQLAdviceMarkers";

type TabView = "diff" | "statement" | "schema";

type SDLDetail = {
  previousSDL: string;
  expectedSDL: string;
  diffDDL: string;
};

type SDLState = {
  task: Task;
  loading: boolean;
  detail: SDLDetail | undefined;
};

interface LocalState {
  showFeatureModal: boolean;
  tab: TabView;
}

const { selectedTask } = useIssueLogic();

const state = reactive<LocalState>({
  showFeatureModal: false,
  tab: "diff",
});
const editorRef = ref<InstanceType<typeof MonacoEditor>>();

const useSDLState = () => {
  const emptyState = (task: Task): SDLState => {
    return {
      task,
      loading: true,
      detail: undefined,
    };
  };

  const map = reactive(new Map<TaskId, SDLState>());

  const findLatestMigrationId = (task: Task) => {
    if (task.status !== "DONE") return undefined;
    const list = task.taskRunList;
    for (let i = list.length - 1; i >= 0; i--) {
      const taskRun = list[i];
      if (taskRun.status === "DONE") {
        return taskRun.result.migrationId;
      }
    }
    return undefined;
  };

  const migrationId = computed(() => {
    const task = selectedTask.value as Task;
    return findLatestMigrationId(task);
  });

  const fetchOngoingSDLDetail = async (
    task: Task
  ): Promise<SDLDetail | undefined> => {
    const database = task.database;
    if (!database) return undefined;
    const previousSDL = await useDatabaseStore().fetchDatabaseSchemaById(
      task.database!.id,
      true // fetch SDL format
    );
    const payload = task.payload as TaskDatabaseSchemaUpdateSDLPayload;
    if (!payload) return undefined;
    const expectedSDL = payload.statement;

    const getSchemaDiff = async () => {
      const { data } = await axios.post("/v1/sql/schema/diff", {
        engineType: database.instance.engine,
        sourceSchema: previousSDL ?? "",
        targetSchema: expectedSDL ?? "",
      });
      return data;
    };
    const diffDDL = (await getSchemaDiff()) ?? "";

    if (task.status === "DONE") {
      throw new Error();
    }

    return { previousSDL, expectedSDL, diffDDL };
  };

  const fetchSDLDetailFromMigrationHistory = async (
    task: Task,
    migrationId: string | undefined
  ): Promise<SDLDetail | undefined> => {
    if (!migrationId) {
      return undefined;
    }
    const history = await useInstanceStore().fetchMigrationHistoryById({
      instanceId: task.instance.id,
      migrationHistoryId: migrationId,
      sdl: true,
    });
    // The latestMigrationId might change during fetching the
    // migrationHistory.
    // Should give up the result.
    const latestMigrationId = findLatestMigrationId(task);
    if (history.id !== latestMigrationId) {
      throw new Error();
    }
    return {
      previousSDL: history.schemaPrev,
      expectedSDL: history.schema,
      diffDDL: history.statement,
    };
  };

  watch(
    [
      () => (selectedTask.value as Task).id,
      () => (selectedTask.value as Task).status,
      migrationId,
    ],
    ([taskId, taskStatus, migrationId]) => {
      const task = selectedTask.value as Task;
      if (!map.has(taskId)) {
        map.set(taskId, emptyState(task));
      }
      const finish = (detail?: SDLState["detail"]) => {
        const state = map.get(taskId)!;
        state.loading = false;
        state.detail = detail;
      };
      try {
        if (taskStatus === "DONE") {
          fetchSDLDetailFromMigrationHistory(task, migrationId).then(finish);
        } else {
          fetchOngoingSDLDetail(task).then(finish);
        }
      } catch {
        // The task has been changed during the fetch
        // The result is meaningless.
      }
    },
    { immediate: true }
  );

  return computed(() => {
    const task = selectedTask.value as Task;
    return map.get(task.id) ?? emptyState(task);
  });
};

const sdlState = useSDLState();

const updateEditorHeight = () => {
  const contentHeight =
    editorRef.value?.editorInstance?.getContentHeight() as number;
  editorRef.value?.setEditorContentHeight(contentHeight);
};

const handleMonacoEditorReady = () => {
  updateEditorHeight();
};

const { markers } = useSQLAdviceMarkers();
</script>
