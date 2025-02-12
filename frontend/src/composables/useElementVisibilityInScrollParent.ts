import { computed } from "vue";
import type { MaybeElementRef } from "@vueuse/core";
import { unrefElement, useElementVisibility } from "@vueuse/core";
import getScrollParent from "scrollparent";

export const useElementVisibilityInScrollParent = (
  element: MaybeElementRef
) => {
  const scrollTarget = computed((): HTMLElement | null => {
    const elem = unrefElement(element);
    if (!elem) {
      return null;
    }
    return getScrollParent(elem);
  });

  return useElementVisibility(element, { scrollTarget });
};
