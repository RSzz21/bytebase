<template>
  <div>
    <BBStepTab
      :sticky="true"
      :step-item-list="STEP_LIST"
      :allow-next="allowNext"
      :finish-title="$t(`common.confirm-and-${policy ? 'update' : 'add'}`)"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="onCancel"
    >
      <template #0>
        <SQLReviewInfo
          :name="state.name"
          :selected-environment="state.selectedEnvironment"
          :available-environment-list="availableEnvironmentList"
          :selected-template="
            state.pendingApplyTemplate || state.selectedTemplate
          "
          :is-edit="!!policy"
          @select-template="tryApplyTemplate"
          @name-change="(val: string) => (state.name = val)"
          @env-change="(env: Environment) => onEnvChange(env)"
        />
      </template>
      <template #1>
        <SQLReviewConfig
          :selected-rule-list="state.selectedRuleList"
          @level-change="onLevelChange"
          @payload-change="onPayloadChange"
          @comment-change="onCommentChange"
        />
      </template>
    </BBStepTab>
    <BBAlertDialog
      ref="alertDialog"
      :style="'CRITICAL'"
      :ok-text="$t('common.confirm')"
      :title="$t('sql-review.create.configure-rule.confirm-override-title')"
      :description="
        $t('sql-review.create.configure-rule.confirm-override-description')
      "
    >
    </BBAlertDialog>
  </div>
</template>

<script lang="ts" setup>
import { reactive, computed, withDefaults, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBStepTabItem } from "@/bbkit/types";
import {
  RuleLevel,
  Environment,
  RuleTemplate,
  convertToCategoryList,
  convertRuleTemplateToPolicyRule,
  ruleIsAvailableInSubscription,
  RuleConfigComponent,
  SQLReviewPolicyTemplate,
  SQLReviewPolicy,
} from "@/types";
import { BBAlertDialog, BBStepTab } from "@/bbkit";
import SQLReviewInfo from "./SQLReviewInfo.vue";
import SQLReviewConfig from "./SQLReviewConfig.vue";
import {
  useCurrentUser,
  pushNotification,
  useEnvironmentList,
  useSQLReviewStore,
  useSubscriptionStore,
} from "@/store";
import { hasWorkspacePermission } from "@/utils";
import { rulesToTemplate } from "./components";

interface LocalState {
  currentStep: number;
  name: string;
  selectedEnvironment: Environment;
  selectedRuleList: RuleTemplate[];
  selectedTemplate: SQLReviewPolicyTemplate | undefined;
  ruleUpdated: boolean;
  showAlertModal: boolean;
  pendingApplyTemplate: SQLReviewPolicyTemplate | undefined;
}

const props = withDefaults(
  defineProps<{
    policy?: SQLReviewPolicy;
    name?: string;
    selectedEnvironment?: Environment;
    selectedRuleList?: RuleTemplate[];
  }>(),
  {
    policy: undefined,
    name: "",
    selectedEnvironment: undefined,
    selectedRuleList: () => [],
  }
);

const emit = defineEmits(["cancel"]);

const alertDialog = ref<InstanceType<typeof BBAlertDialog>>();
const { t } = useI18n();
const router = useRouter();
const store = useSQLReviewStore();
const currentUser = useCurrentUser();
const subscriptionStore = useSubscriptionStore();

const BASIC_INFO_STEP = 0;
const CONFIGURE_RULE_STEP = 1;
const PREVIEW_STEP = 2;
const ROUTE_NAME = "setting.workspace.sql-review";

const STEP_LIST: BBStepTabItem[] = [
  { title: t("sql-review.create.basic-info.name") },
  { title: t("sql-review.create.configure-rule.name") },
];

const state = reactive<LocalState>({
  currentStep: BASIC_INFO_STEP,
  name: props.name || t("sql-review.create.basic-info.display-name-default"),
  selectedEnvironment: props.selectedEnvironment,
  selectedRuleList: [...props.selectedRuleList],
  selectedTemplate: props.policy
    ? rulesToTemplate(props.policy, false /* withDisabled=false */)
    : undefined,
  ruleUpdated: false,
  showAlertModal: false,
  pendingApplyTemplate: undefined,
});

const onTemplateApply = (template: SQLReviewPolicyTemplate | undefined) => {
  if (!template) {
    return;
  }
  state.selectedTemplate = template;
  state.pendingApplyTemplate = undefined;

  const categoryList = convertToCategoryList(template.ruleList);
  state.selectedRuleList = categoryList.reduce<RuleTemplate[]>(
    (res, category) => {
      res.push(
        ...category.ruleList.map((rule) => ({
          ...rule,
          level: ruleIsAvailableInSubscription(
            rule.type,
            subscriptionStore.currentPlan
          )
            ? rule.level
            : RuleLevel.DISABLED,
        }))
      );
      return res;
    },
    []
  );
};

const availableEnvironmentList = computed((): Environment[] => {
  const environmentList = useEnvironmentList(["NORMAL"]);

  const filteredList = store.availableEnvironments(
    environmentList.value,
    props.policy?.id
  );

  return filteredList;
});

const onCancel = () => {
  if (props.policy) {
    emit("cancel");
  } else {
    router.push({
      name: ROUTE_NAME,
    });
  }
};

const allowNext = computed((): boolean => {
  switch (state.currentStep) {
    case BASIC_INFO_STEP:
      return (
        !!state.name &&
        state.selectedRuleList.length > 0 &&
        !!state.selectedEnvironment
      );
    case CONFIGURE_RULE_STEP:
      return state.selectedRuleList.length > 0;
    case PREVIEW_STEP:
      return true;
  }
  return false;
});

const tryChangeStep = (
  oldStep: number,
  newStep: number,
  allowChangeCallback: () => void
) => {
  if (oldStep === 0 && newStep === 1) {
    if (state.pendingApplyTemplate) {
      alertDialog.value!.open().then((result) => {
        if (result) {
          onTemplateApply(state.pendingApplyTemplate);
          allowChangeCallback();
        }
      });
      return;
    }
  }
  state.currentStep = newStep;
  allowChangeCallback();
};

const tryFinishSetup = (allowChangeCallback: () => void) => {
  if (
    !hasWorkspacePermission(
      "bb.permission.workspace.manage-sql-review-policy",
      currentUser.value.role
    )
  ) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("sql-review.no-permission"),
    });
  }
  const upsert = {
    name: state.name,
    ruleList: state.selectedRuleList.map((rule) =>
      convertRuleTemplateToPolicyRule(rule)
    ),
  };

  if (props.policy) {
    store
      .updateReviewPolicy({
        id: props.policy.id,
        ...upsert,
      })
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("sql-review.policy-updated"),
        });
      });
  } else {
    store
      .addReviewPolicy({
        ...upsert,
        environmentId: state.selectedEnvironment?.id,
      })
      .then(() => {
        pushNotification({
          module: "bytebase",
          style: "SUCCESS",
          title: t("sql-review.policy-created"),
        });
      });
  }

  allowChangeCallback();
  onCancel();
};

const onEnvChange = (env: Environment) => {
  state.selectedEnvironment = env;
};

const tryApplyTemplate = (template: SQLReviewPolicyTemplate) => {
  if (state.ruleUpdated || props.policy) {
    if (template.id === state.selectedTemplate?.id) {
      state.pendingApplyTemplate = undefined;
      return;
    }
    state.pendingApplyTemplate = template;
    return;
  }
  onTemplateApply(template);
};

const change = (rule: RuleTemplate, overrides: Partial<RuleTemplate>) => {
  if (
    !ruleIsAvailableInSubscription(rule.type, subscriptionStore.currentPlan)
  ) {
    return;
  }

  const index = state.selectedRuleList.findIndex((r) => r.type === rule.type);
  if (index < 0) {
    return;
  }
  const newRule = {
    ...state.selectedRuleList[index],
    ...overrides,
  };
  state.selectedRuleList[index] = newRule;
  state.ruleUpdated = true;
};

const onPayloadChange = (
  rule: RuleTemplate,
  componentList: RuleConfigComponent[]
) => {
  change(rule, { componentList });
};

const onLevelChange = (rule: RuleTemplate, level: RuleLevel) => {
  change(rule, { level });
};

const onCommentChange = (rule: RuleTemplate, comment: string) => {
  change(rule, { comment });
};
</script>
